package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jjudge-oj/apiserver/config"
	"github.com/jjudge-oj/apiserver/internal/db"
	"github.com/jjudge-oj/apiserver/internal/handlers"
	"github.com/jjudge-oj/apiserver/internal/mq"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/storage"
	"github.com/jjudge-oj/apiserver/internal/store"
)

// Server wraps the HTTP server and router.
type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	db         *sql.DB
	mq         *mq.MQ
}

// New constructs a Server with basic middleware and defaults.
func New(ctx context.Context, cfg config.Config) (*Server, error) {
	dbConn, err := db.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}

	problemRepo := store.NewProblemRepository(dbConn)
	userRepo := store.NewUserRepository(dbConn)
	submissionRepo := store.NewSubmissionRepository(dbConn)

	storageClient, err := storage.NewStorageFromConfig(ctx, cfg)
	if err != nil {
		_ = dbConn.Close()
		return nil, err
	}
	if err := storageClient.EnsureBucket(ctx); err != nil {
		_ = dbConn.Close()
		return nil, err
	}

	mqClient, err := mq.NewRabbitMQClient(cfg.RabbitMQ)
	if err != nil {
		_ = dbConn.Close()
		return nil, err
	}
	mqWrapper := mq.New(mqClient)

	problemService := services.NewProblemService(problemRepo, storageClient)
	userService := services.NewUserService(userRepo)
	submissionService := services.NewSubmissionService(submissionRepo, storageClient, mqWrapper)

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		_ = dbConn.Close()
		return nil, errors.New("JWT_SECRET is required")
	}

	authMiddleware := handlers.RequireAuth(jwtSecret)

	router := chi.NewRouter()
	router.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Recoverer,
		middleware.Logger,
		handlers.CORSMiddleware,
		middleware.Timeout(60*time.Second),
	)
	router.Get("/healthz", handlers.Healthz)
	router.Route("/problems", func(r chi.Router) {
		handlers.ProblemRouter(r, problemService, userService, authMiddleware)
	})
	router.Route("/problems/{problemID}/submissions", func(r chi.Router) {
		handlers.SubmissionRouter(r, submissionService, problemService, authMiddleware)
	})
	router.Route("/auth", func(r chi.Router) {
		handlers.AuthRouter(r, userService, jwtSecret)
	})

	port := cfg.ServerPort
	if port == 0 {
		port = 8080
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		router:     router,
		db:         dbConn,
		mq:         mqWrapper,
	}, nil
}

// Router exposes the chi router for route registration.
func (s *Server) Router() *chi.Mux {
	return s.router
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown attempts a graceful shutdown.
func (s *Server) Shutdown() error {
	if s.db != nil {
		_ = s.db.Close()
	}
	if s.mq != nil {
		_ = s.mq.Close()
	}
	return s.httpServer.Close()
}

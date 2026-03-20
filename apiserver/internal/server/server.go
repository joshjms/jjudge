package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jjudge-oj/api/types"
	"github.com/jjudge-oj/apiserver/config"
	"github.com/jjudge-oj/apiserver/internal/db"
	"github.com/jjudge-oj/apiserver/internal/handlers"
	"github.com/jjudge-oj/apiserver/internal/mq"
	"github.com/jjudge-oj/apiserver/internal/services"
	"github.com/jjudge-oj/apiserver/internal/storage"
	"github.com/jjudge-oj/apiserver/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// Server wraps the HTTP server and router.
type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	db         *sql.DB
	mq         *mq.MQ
}

// New constructs a Server with basic middleware and defaults.
func New(ctx context.Context, cfg *config.Config) (*Server, error) {
	dbConn, err := db.Open(ctx, cfg.Database)
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

	contestRepo := store.NewContestRepository(dbConn)
	blogRepo := store.NewBlogRepository(dbConn)

	problemService := services.NewProblemService(problemRepo, storageClient)
	userService := services.NewUserService(userRepo)
	submissionService := services.NewSubmissionService(submissionRepo, storageClient, mqWrapper)
	contestService := services.NewContestService(contestRepo, storageClient, mqWrapper)
	blogService := services.NewBlogService(blogRepo)

	if err := ensureAdminUser(ctx, userService, cfg); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("ensure admin user: %w", err)
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		_ = dbConn.Close()
		return nil, errors.New("JWT_SECRET is required")
	}

	authMiddleware := handlers.RequireAuth(jwtSecret)
	optionalAuthMiddleware := handlers.OptionalAuth(jwtSecret)

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
		handlers.ProblemRouter(r, problemService, userService, authMiddleware, optionalAuthMiddleware)
	})
	router.Route("/problems/{problemID}/submissions", func(r chi.Router) {
		handlers.SubmissionRouter(r, submissionService, problemService, userService, authMiddleware)
	})
	router.Route("/auth", func(r chi.Router) {
		handlers.AuthRouter(r, userService, jwtSecret)
	})
	router.Route("/users", func(r chi.Router) {
		handlers.UserRouter(r, userService, jwtSecret, storageClient)
	})

	submissionHandler := handlers.NewSubmissionHandler(submissionService, problemService, userService)
	router.Get("/submissions", submissionHandler.ListSubmissions)
	router.Get("/submissions/{submissionID}", submissionHandler.GetSubmission)

	router.Route("/contests", func(r chi.Router) {
		handlers.ContestRouter(r, contestService, problemService, userService, authMiddleware, optionalAuthMiddleware)
	})

	router.Route("/blog", func(r chi.Router) {
		handlers.BlogRouter(r, blogService, userService, authMiddleware, optionalAuthMiddleware)
	})

	router.Route("/admin/approvals", func(r chi.Router) {
		handlers.ApprovalRouter(r, problemService, contestService, userService, authMiddleware)
	})

	router.Route("/manager", func(r chi.Router) {
		handlers.ManagerRouter(r, problemService, contestService, userService, authMiddleware)
	})

	// Start background result consumer for regular submissions
	go func() {
		const resultQueue = "submission-results"
		err := mqWrapper.Subscribe(ctx, resultQueue, func(ctx context.Context, msg mq.Message) error {
			var submission types.Submission
			if err := json.Unmarshal(msg.Data, &submission); err != nil {
				log.Printf("result consumer: bad message, discarding: %v", err)
				return nil // ack — malformed, retrying won't help
			}
			if _, err := submissionService.Update(ctx, submission); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					log.Printf("result consumer: submission %d not found, discarding", submission.ID)
					return nil // ack — permanent, retrying won't help
				}
				log.Printf("result consumer: failed to update submission %d: %v", submission.ID, err)
				return err // nack+requeue — potentially transient (e.g. DB down)
			}
			log.Printf("result consumer: updated submission %d verdict=%s", submission.ID, submission.Verdict)
			return nil
		})
		if err != nil && ctx.Err() == nil {
			log.Printf("result consumer exited: %v", err)
		}
	}()

	// Start background result consumer for contest submissions
	go func() {
		const contestResultQueue = "contest-submission-results"
		err := mqWrapper.Subscribe(ctx, contestResultQueue, func(ctx context.Context, msg mq.Message) error {
			var cs types.ContestSubmission
			if err := json.Unmarshal(msg.Data, &cs); err != nil {
				log.Printf("contest result consumer: bad message, discarding: %v", err)
				return nil // ack — malformed, retrying won't help
			}
			if _, err := contestService.UpdateContestSubmission(ctx, cs); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					log.Printf("contest result consumer: submission %d not found, discarding", cs.ID)
					return nil // ack — permanent, retrying won't help
				}
				log.Printf("contest result consumer: failed to update submission %d: %v", cs.ID, err)
				return err // nack+requeue — potentially transient
			}
			log.Printf("contest result consumer: updated submission %d verdict=%s", cs.ID, cs.Verdict)
			return nil
		})
		if err != nil && ctx.Err() == nil {
			log.Printf("contest result consumer exited: %v", err)
		}
	}()

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

// ensureAdminUser creates or updates an admin user based on environment config.
func ensureAdminUser(ctx context.Context, userService *services.UserService, cfg *config.Config) error {
	if cfg.AdminUser == "" || cfg.AdminPassword == "" {
		return nil
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	existing, err := userService.GetByUsername(ctx, cfg.AdminUser)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("lookup admin user: %w", err)
	}

	if errors.Is(err, store.ErrNotFound) {
		_, err := userService.Create(ctx, types.User{
			Username:     cfg.AdminUser,
			Email:        cfg.AdminUser + "@admin.local",
			Name:         cfg.AdminUser,
			Role:         "admin",
			PasswordHash: string(hashed),
		})
		if err != nil {
			return fmt.Errorf("create admin user: %w", err)
		}
		log.Printf("admin user %q created", cfg.AdminUser)
		return nil
	}

	existing.Role = "admin"
	existing.PasswordHash = string(hashed)
	if _, err := userService.Update(ctx, existing); err != nil {
		return fmt.Errorf("update admin user: %w", err)
	}
	log.Printf("admin user %q updated", cfg.AdminUser)
	return nil
}

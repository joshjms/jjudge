//go:build e2e

package e2e

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jjudge-oj/apiserver/config"
	"github.com/jjudge-oj/apiserver/internal/server"
	_ "github.com/lib/pq"
)

const (
	serverPort = 18080
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	root, err := repoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to locate repo root: %v\n", err)
		os.Exit(1)
	}

	if err := dockerCompose(ctx, root, "up", "-d"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start docker compose: %v\n", err)
		os.Exit(1)
	}

	if err := waitForPostgres(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "postgres not ready: %v\n", err)
		_ = dockerCompose(context.Background(), root, "down")
		os.Exit(1)
	}

	if err := runMigrations(root); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		_ = dockerCompose(context.Background(), root, "down")
		os.Exit(1)
	}

	srv, err := startServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start server: %v\n", err)
		_ = dockerCompose(context.Background(), root, "down")
		os.Exit(1)
	}

	baseURL := fmt.Sprintf("http://localhost:%d", serverPort)
	if err := waitForHealth(ctx, baseURL+"/healthz"); err != nil {
		fmt.Fprintf(os.Stderr, "server not healthy: %v\n", err)
		_ = srv.Shutdown()
		_ = dockerCompose(context.Background(), root, "down")
		os.Exit(1)
	}

	code := m.Run()

	_ = srv.Shutdown()
	_ = dockerCompose(context.Background(), root, "down")
	os.Exit(code)
}

func TestProblemLifecycle(t *testing.T) {
	baseURL := fmt.Sprintf("http://localhost:%d", serverPort)
	username := fmt.Sprintf("admin_%d", time.Now().UnixNano())
	password := "testpass123!"

	token, err := registerUser(t, baseURL, username, password)
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	if err := promoteUserToAdmin(username); err != nil {
		t.Fatalf("promote user: %v", err)
	}

	bundleName, bundleData, err := buildTestBundle()
	if err != nil {
		t.Fatalf("build bundle: %v", err)
	}

	resp, err := createProblem(t, baseURL, token, bundleName, bundleData)
	if err != nil {
		t.Fatalf("create problem: %v", err)
	}

	if resp.Title != "Cat Test Problem" {
		t.Fatalf("unexpected problem title: %q", resp.Title)
	}
	if resp.ID == 0 {
		t.Fatalf("expected problem ID to be set")
	}

	updated, err := updateProblem(t, baseURL, token, resp.ID, bundleName, bundleData)
	if err != nil {
		t.Fatalf("update problem: %v", err)
	}
	if updated.Title != "Cat Test Problem Updated" {
		t.Fatalf("unexpected updated problem title: %q", updated.Title)
	}

	fetched, err := getProblem(t, baseURL, resp.ID)
	if err != nil {
		t.Fatalf("get problem: %v", err)
	}
	if fetched.ID != resp.ID {
		t.Fatalf("unexpected problem id: %d", fetched.ID)
	}

	if err := deleteProblem(t, baseURL, token, resp.ID); err != nil {
		t.Fatalf("delete problem: %v", err)
	}

	if err := expectProblemNotFound(t, baseURL, resp.ID); err != nil {
		t.Fatalf("expected deleted problem to be missing: %v", err)
	}
}

type problemResponse struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type authResponse struct {
	Token string `json:"token"`
}

func registerUser(t *testing.T, baseURL, username, password string) (string, error) {
	t.Helper()

	payload := map[string]string{
		"username": username,
		"email":    fmt.Sprintf("%s@example.com", username),
		"name":     "Test Admin",
		"password": password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/register", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("register status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var parsed authResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.Token == "" {
		return "", fmt.Errorf("missing token in register response")
	}
	return parsed.Token, nil
}

func promoteUserToAdmin(username string) error {
	cfg := config.LoadConfig()
	dsn := buildPostgresURL(cfg)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, "UPDATE users SET role = 'admin', updated_at = NOW() WHERE username = $1", username)
	return err
}

func createProblem(t *testing.T, baseURL, token, bundleName string, bundle []byte) (problemResponse, error) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("title", "Cat Test Problem")
	_ = writer.WriteField("description", "This is the hardest problem to have ever existed.")
	_ = writer.WriteField("difficulty", "800")
	_ = writer.WriteField("time_limit", "1000")
	_ = writer.WriteField("memory_limit", strconv.FormatInt(256<<20, 10))
	_ = writer.WriteField("tags", "testing,cats")
	_ = writer.WriteField("testcase_groups", buildTestcaseGroupsJSON())

	part, err := writer.CreateFormFile("bundle", bundleName)
	if err != nil {
		return problemResponse{}, err
	}
	if _, err := part.Write(bundle); err != nil {
		return problemResponse{}, err
	}
	if err := writer.Close(); err != nil {
		return problemResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/problems", &body)
	if err != nil {
		return problemResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return problemResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return problemResponse{}, fmt.Errorf("create problem status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var parsed problemResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return problemResponse{}, err
	}
	return parsed, nil
}

func updateProblem(t *testing.T, baseURL, token string, id int, bundleName string, bundle []byte) (problemResponse, error) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("title", "Cat Test Problem Updated")
	_ = writer.WriteField("description", "Did they change the problem?")
	_ = writer.WriteField("difficulty", "900")
	_ = writer.WriteField("time_limit", "1500")
	_ = writer.WriteField("memory_limit", strconv.FormatInt(512<<20, 10))
	_ = writer.WriteField("tags", "math,arrays,update")
	_ = writer.WriteField("testcase_groups", buildTestcaseGroupsJSON())

	part, err := writer.CreateFormFile("bundle", bundleName)
	if err != nil {
		return problemResponse{}, err
	}
	if _, err := part.Write(bundle); err != nil {
		return problemResponse{}, err
	}
	if err := writer.Close(); err != nil {
		return problemResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/problems/%d", baseURL, id), &body)
	if err != nil {
		return problemResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return problemResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return problemResponse{}, fmt.Errorf("update problem status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var parsed problemResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return problemResponse{}, err
	}
	return parsed, nil
}

func getProblem(t *testing.T, baseURL string, id int) (problemResponse, error) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/problems/%d", baseURL, id), nil)
	if err != nil {
		return problemResponse{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return problemResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return problemResponse{}, fmt.Errorf("get problem status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var parsed problemResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return problemResponse{}, err
	}
	return parsed, nil
}

func deleteProblem(t *testing.T, baseURL, token string, id int) error {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/problems/%d", baseURL, id), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete problem status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

func expectProblemNotFound(t *testing.T, baseURL string, id int) error {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/problems/%d", baseURL, id), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected 404 after delete, got %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

func buildTestBundle() (string, []byte, error) {
	tarData, err := buildTarGzBundle()
	if err != nil {
		return "", nil, err
	}
	return "testcases.tar.gz", tarData, nil
}

func buildTarGzBundle() ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	if err := addTarFile(tw, "0_0.in", "1 2\n"); err != nil {
		return nil, err
	}
	if err := addTarFile(tw, "0_0.out", "3\n"); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addTarFile(tw *tar.Writer, name, content string) error {
	header := &tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err := tw.Write([]byte(content))
	return err
}

func buildTestcaseGroupsJSON() string {
	groups := []map[string]any{
		{
			"order_id": 0,
			"name":     "Sample",
			"points":   100,
		},
	}
	data, err := json.Marshal(groups)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func waitForPostgres(ctx context.Context) error {
	cfg := config.LoadConfig()
	dsn := buildPostgresURL(cfg)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("postgres ping timeout: %w", err)
		case <-ticker.C:
		}
	}
}

func waitForHealth(ctx context.Context, url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return fmt.Errorf("health check failed: %w", err)
			}
			return fmt.Errorf("health check failed with status")
		case <-ticker.C:
		}
	}
}

func runMigrations(root string) error {
	cfg := config.LoadConfig()
	dsn := buildPostgresURL(cfg)
	migrationsPath := filepath.Join(root, "internal", "db", "migrations")
	migrationsURL := "file://" + migrationsPath

	migrator, err := migrate.New(migrationsURL, dsn)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = migrator.Close()
	}()

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func buildPostgresURL(cfg config.Config) string {
	sslmode := "disable"
	if cfg.Database.UseSSL {
		sslmode = "require"
	}
	host := fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port)
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		host,
		cfg.Database.DBName,
		sslmode,
	)
}

func startServer() (*server.Server, error) {
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("SERVER_PORT", fmt.Sprintf("%d", serverPort))
	_ = os.Setenv("DB_HOST", "localhost")
	_ = os.Setenv("DB_PORT", "5432")
	_ = os.Setenv("DB_USER", "jjudge")
	_ = os.Setenv("DB_PASSWORD", "jjudge")
	_ = os.Setenv("DB_NAME", "jjudge")
	_ = os.Setenv("DB_USE_SSL", "false")
	_ = os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	_ = os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	_ = os.Setenv("MINIO_BUCKET", "jjudge")

	cfg := config.LoadConfig()
	srv, err := server.New(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	go func() {
		_ = srv.Start()
	}()

	return srv, nil
}

func dockerCompose(ctx context.Context, root string, args ...string) error {
	composeFile := filepath.Join(root, "development", "docker-compose.yml")
	baseArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmd := exec.CommandContext(ctx, "docker", baseArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

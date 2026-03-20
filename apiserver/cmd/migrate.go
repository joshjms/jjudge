package cmd

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jjudge-oj/apiserver/config"
	"github.com/jjudge-oj/apiserver/internal/db"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all up migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.LoadConfig()
		dsn := buildPostgresURL(cfg.Database)

		src, err := iofs.New(db.Migrations, "migrations")
		if err != nil {
			return fmt.Errorf("init migration source: %w", err)
		}

		migrator, err := migrate.NewWithSourceInstance("iofs", src, dsn)
		if err != nil {
			return fmt.Errorf("init migrator failed: %w", err)
		}
		defer func() {
			_, _ = migrator.Close()
		}()

		if err := migrator.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				return nil
			}
			return fmt.Errorf("migrate up failed: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
}

func buildPostgresURL(cfg *config.DatabaseConfig) string {
	sslmode := "disable"
	if cfg.UseSSL {
		sslmode = "require"
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		User:   url.UserPassword(cfg.User, cfg.Password),
		Path:   cfg.DBName,
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String()
}

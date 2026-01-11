package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/jjudge-oj/apiserver/config"
	_ "github.com/lib/pq"
)

const (
	defaultDBDriver     = "postgres"
	defaultPingTimeout  = 5 * time.Second
	defaultConnMaxIdle  = 2 * time.Minute
	defaultConnMaxLife  = 30 * time.Minute
	defaultMaxIdleConns = 5
	defaultMaxOpenConns = 25
)

func Open(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	sslmode := "disable"
	if cfg.Database.UseSSL {
		sslmode = "require"
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port),
		User:   url.UserPassword(cfg.Database.User, cfg.Database.Password),
		Path:   cfg.Database.DBName,
	}

	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()

	dsn := u.String()

	db, err := sql.Open(defaultDBDriver, dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(defaultConnMaxIdle)
	db.SetConnMaxLifetime(defaultConnMaxLife)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetMaxOpenConns(defaultMaxOpenConns)

	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

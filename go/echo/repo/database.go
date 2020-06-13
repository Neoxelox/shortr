package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func connect(ctx context.Context, dsn string, retries int) (*pgxpool.Pool, error) {
	delay := time.NewTicker(1 * time.Second)
	timeout := (time.Duration(retries) * time.Second)

	defer delay.Stop()

	timeoutExceeded := time.After(timeout)
	for {
		select {
		case <-timeoutExceeded:
			return nil, errors.New("unable to connect to database")

		case <-delay.C:
			db, err := pgxpool.Connect(ctx, dsn)
			if err == nil {
				return db, nil
			}
		}
	}
}
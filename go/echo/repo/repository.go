package repo

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"shortr/model"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgxutil"
)

var ErrNoRows = pgx.ErrNoRows
var rErrNoRows = regexp.MustCompile(fmt.Sprintf("^%s$", pgx.ErrNoRows))
var ErrIntegrityViolation = errors.New("integrity constraint violation")
var rErrIntegrityViolation = regexp.MustCompile(fmt.Sprintf("(SQLSTATE %s)", pgerrcode.UniqueViolation))

// Repo describes the URLs repository
type Repo struct {
	db   *pgxpool.Pool
	conn connection
}

type connection interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// Connect tries to connect to an specified database via the dsn connection string
func Connect(dsn string, retries int, logger pgx.Logger) (*Repo, error) {
	db, err := connect(context.Background(), dsn, retries, logger)
	if err != nil {
		return nil, err
	}
	return &Repo{
		db:   db,
		conn: db,
	}, nil
}

// Disconnect closes the connection with the database
func (r *Repo) Disconnect() {
	r.db.Close()
}

func (r *Repo) Transaction(ctx context.Context, fn func(*Repo) error) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		}
	}()

	err = fn(&Repo{
		db:   r.db,
		conn: tx,
	})
	if err != nil {
		tx.Rollback(ctx)
		return err
	}
	err = tx.Commit(ctx)
	return err
}

// GetByID retrieves the URL by its id
func (r *Repo) GetByID(ctx context.Context, id int) (model.URL, error) {
	var URL model.URL
	query := `SELECT * FROM "urls"
			  WHERE "id" = $1;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, id)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// GetByName retrieves the URL by its name
func (r *Repo) GetByName(ctx context.Context, name string) (model.URL, error) {
	var URL model.URL
	query := `SELECT * FROM "urls"
			  WHERE "name" = $1;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, name)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// Create creates a new entry for the url and returns the new URL
func (r *Repo) Create(ctx context.Context, url string) (model.URL, error) {
	var URL model.URL
	createdAt := time.Now()
	modifiedAt := createdAt
	query := `INSERT INTO "urls" ("url", "created_at", "modified_at")
			  VALUES ($1, $2, $3)
			  RETURNING *;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, url, createdAt, modifiedAt)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// UpdateNameByID updates the name for the url by its id and returns the updated URL
func (r *Repo) UpdateNameByID(ctx context.Context, id int, name string) (model.URL, error) {
	var URL model.URL
	modifiedAt := time.Now()
	query := `UPDATE "urls"
			  SET "name" = $1, "modified_at" = $2
			  WHERE "id" = $3
			  RETURNING *;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, name, modifiedAt, id)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// UpdateURLByID updates the url by its id and returns the updated URL
func (r *Repo) UpdateURLByID(ctx context.Context, id int, url string) (model.URL, error) {
	var URL model.URL
	modifiedAt := time.Now()
	query := `UPDATE "urls"
	          SET "url" = $1, "modified_at" = $2
			  WHERE "id" = $3
			  RETURNING *;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, url, modifiedAt, id)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// UpdateURLByName updates the url by its name and returns the updated URL
func (r *Repo) UpdateURLByName(ctx context.Context, name string, url string) (model.URL, error) {
	var URL model.URL
	modifiedAt := time.Now()
	query := `UPDATE "urls"
			  SET "url" = $1, "modified_at" = $2
			  WHERE "name" = $3
			  RETURNING *;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, url, modifiedAt, name)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// UpdateMetricsByID updates the metrics for the url by its id and returns the updated URL
func (r *Repo) UpdateMetricsByID(ctx context.Context, id int) (model.URL, error) {
	var URL model.URL
	lastHitAt := time.Now()
	query := `UPDATE "urls"
			  SET "hits" = "hits" + 1, "last_hit_at" = $1
			  WHERE "id" = $2
			  RETURNING *;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, lastHitAt, id)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// UpdateMetricsByName updates the metrics for the url by its name and returns the updated URL
func (r *Repo) UpdateMetricsByName(ctx context.Context, name string) (model.URL, error) {
	var URL model.URL
	lastHitAt := time.Now()
	query := `UPDATE "urls"
			  SET "hits" = "hits" + 1, "last_hit_at" = $1
			  WHERE "name" = $2
			  RETURNING *;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, lastHitAt, name)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// DeleteByID deletes de url entry by its id and returns the deleted URL
func (r *Repo) DeleteByID(ctx context.Context, id int) (model.URL, error) {
	var URL model.URL
	query := `DELETE FROM "urls"
			  WHERE "id" = $1
			  RETURNING *;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, id)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// DeleteByName deletes de url entry by its name and returns the deleted URL
func (r *Repo) DeleteByName(ctx context.Context, name string) (model.URL, error) {
	var URL model.URL
	query := `DELETE FROM "urls"
			  WHERE "name" = $1
			  RETURNING *;`
	err := pgxutil.SelectStruct(ctx, r.conn, &URL, query, name)
	if err != nil {
		switch {
		case rErrNoRows.MatchString(err.Error()):
			return URL, ErrNoRows
		case rErrIntegrityViolation.MatchString(err.Error()):
			return URL, ErrIntegrityViolation
		}
	}
	return URL, err
}

// Health checks the database connection health
func (r Repo) Health() error {
	if _, err := r.db.Exec(context.Background(), ";"); err != nil {
		return err
	}
	if r.db.Stat().TotalConns() < r.db.Config().MinConns {
		return errors.New("database connections below the threshold")
	}
	return nil
}

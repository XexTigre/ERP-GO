package db

import (
	"errors"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func Classify(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fault.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fault.ErrConflict
		case "23503":
			return fault.ErrNotFound
		case "23514", "22001", "22P02":
			return fault.ErrInvalid
		}
	}
	return err
}

func NullID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

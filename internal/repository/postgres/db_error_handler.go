// internal/repository/postgres/db_error_handler.go
package postgres

import (
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"payflow/internal/domain"
)

// DBErrToDomainErr translates GORM and PostgreSQL errors into our domain errors.
func DBErrToDomainErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return domain.ErrConflict
		}
	}
	return err // Return original error if no mapping is found
}

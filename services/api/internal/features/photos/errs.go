package photos

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// isInvalidUUID reports a malformed id (SQLSTATE 22P02), treated as "not found".
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

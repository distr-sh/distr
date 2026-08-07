package db

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// isStillReferencedError reports whether err is Postgres refusing a delete because another row still
// references the deleted one. A foreign key with ON DELETE RESTRICT reports restrict_violation
// (23001) and one with the default NO ACTION foreign_key_violation (23503), so a delete has to accept
// both to answer with a conflict rather than an internal error.
func isStillReferencedError(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && (pgErr.Code == pgerrcode.RestrictViolation || pgErr.Code == pgerrcode.ForeignKeyViolation)
}

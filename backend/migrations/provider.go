package migrations

import (
	"database/sql"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Provider serializes all migrators for this database. Give Up/Down a deadline too.
func Provider(db *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockTimeout(1, 30), lock.WithUnlockTimeout(1, 5))
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectPostgres, db, Files, goose.WithSessionLocker(locker))
}

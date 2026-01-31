package migrations

import (
	"embed"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed 0*.sql
var embedMigrations embed.FS

func Migrate(cfg *pgxpool.Config) error {
	fmt.Println("start migrations")
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal("goose.SetDialect", err)
	}

	db := stdlib.OpenDB(*(cfg.ConnConfig))
	defer func() { _ = db.Close() }()

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose.Up %w", err)
	}

	fmt.Println("migrations done")

	return nil
}

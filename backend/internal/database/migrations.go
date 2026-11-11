package database

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	Version string
	Name    string
	SQL     string
}

func Migrate(db *gorm.DB, log *slog.Logger) error {
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS public.go_schema_migrations (
			version character varying PRIMARY KEY,
			name character varying NOT NULL,
			applied_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, item := range migrations {
		var count int64
		if err := db.Table("public.go_schema_migrations").Where("version = ?", item.Version).Count(&count).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", item.Version, err)
		}
		if count > 0 {
			continue
		}

		if item.Version == "0001" {
			var usersTable *string
			if err := db.Raw("SELECT to_regclass('public.users')::text").Scan(&usersTable).Error; err != nil {
				return fmt.Errorf("detect existing schema: %w", err)
			}
			if usersTable != nil {
				log.Info("existing PostgreSQL schema detected; baselining migration", "version", item.Version)
				if err := markMigration(db, item); err != nil {
					return err
				}
				continue
			}
		}

		log.Info("applying migration", "version", item.Version, "name", item.Name)
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(item.SQL).Error; err != nil {
				return fmt.Errorf("execute migration: %w", err)
			}
			return tx.Exec(
				"INSERT INTO public.go_schema_migrations (version, name) VALUES (?, ?)",
				item.Version, item.Name,
			).Error
		})
		if err != nil {
			return fmt.Errorf("migration %s %s: %w", item.Version, item.Name, err)
		}
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	out := make([]migration, 0, len(names))
	for _, name := range names {
		content, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		version, label, _ := strings.Cut(strings.TrimSuffix(name, ".sql"), "_")
		out = append(out, migration{Version: version, Name: label, SQL: string(content)})
	}
	return out, nil
}

func markMigration(db *gorm.DB, item migration) error {
	if err := db.Exec(
		"INSERT INTO public.go_schema_migrations (version, name) VALUES (?, ?) ON CONFLICT (version) DO NOTHING",
		item.Version, item.Name,
	).Error; err != nil {
		return fmt.Errorf("record migration %s: %w", item.Version, err)
	}
	return nil
}

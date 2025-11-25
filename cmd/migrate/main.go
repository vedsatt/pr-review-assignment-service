package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/vedsatt/pr-review-assignment-service/internal/config"
)

func main() {
	var migrationsPath string
	var cmd string

	flag.StringVar(&migrationsPath, "path", "./migrations", "Path to migrations files")
	flag.StringVar(&cmd, "command", "up", "Migrations command")
	flag.Parse()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	addr := net.JoinHostPort(cfg.PostgresCfg.Host, cfg.PostgresCfg.Port)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.PostgresCfg.User,
		cfg.PostgresCfg.Password,
		addr,
		cfg.PostgresCfg.DBName,
	)

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		dsn,
	)

	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	switch cmd {
	case "up":
		if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Printf("migration failed: %v", err)
			return
		}
		log.Println("migration up completed successfully")
	case "down":
		if err = m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Printf("migration failed: %v", err)
			return
		}
		log.Println("migration down completed successfully")
	default:
		log.Printf("unknown command: %s", cmd)
	}
}

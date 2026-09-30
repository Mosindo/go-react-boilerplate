// Command admin performs privileged operations that are deliberately not
// exposed over HTTP.
//
//	go run ./cmd/admin role <email> moderator   # grant the moderator role
//	go run ./cmd/admin role <email> member      # revoke it
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/moderation"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: admin role <email> <member|moderator>")
	os.Exit(2)
}

func main() {
	if len(os.Args) != 4 || os.Args[1] != "role" {
		usage()
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 10*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	service := moderation.NewService(moderation.NewPGRepository(pool), nil)
	email := auth.NormalizeEmail(os.Args[2])
	if err := service.SetRole(ctx, email, os.Args[3]); err != nil {
		log.Fatal(err)
	}
	log.Printf("role of %s set to %s", email, os.Args[3])
}

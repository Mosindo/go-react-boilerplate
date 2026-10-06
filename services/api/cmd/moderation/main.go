// Command moderation is the operator's tool for reviewing user reports.
//
//	go run ./cmd/moderation reports            list open reports, grouped by reported user
//	go run ./cmd/moderation dismiss   <userId> mark the open reports as not actionable (profile reappears)
//	go run ./cmd/moderation suspend   <userId> suspend the account, end its sessions, close the reports
//	go run ./cmd/moderation unsuspend <userId> lift a suspension
//
// A profile with 3+ distinct open reporters is hidden from discovery until an
// operator resolves it with dismiss or suspend.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	url := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if url == "" {
		fatal("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		fatal(err.Error())
	}
	defer pool.Close()

	switch os.Args[1] {
	case "reports":
		rows, err := pool.Query(ctx, `
			SELECT r.reported_id, COALESCE(p.first_name, '?'), u.status,
			       count(DISTINCT r.reporter_id), string_agg(DISTINCT r.reason, ','), max(r.created_at)
			FROM reports r
			JOIN users u ON u.id = r.reported_id
			LEFT JOIN profiles p ON p.user_id = r.reported_id
			WHERE r.status = 'open'
			GROUP BY r.reported_id, p.first_name, u.status
			ORDER BY count(DISTINCT r.reporter_id) DESC, max(r.created_at) DESC
		`)
		if err != nil {
			fatal(err.Error())
		}
		defer rows.Close()
		fmt.Printf("%-36s  %-14s %-9s %-9s %s\n", "USER", "NAME", "STATUS", "REPORTERS", "REASONS (last report)")
		for rows.Next() {
			var id, name, status, reasons string
			var n int
			var last time.Time
			if err := rows.Scan(&id, &name, &status, &n, &reasons, &last); err != nil {
				fatal(err.Error())
			}
			fmt.Printf("%-36s  %-14s %-9s %-9d %s (%s)\n", id, name, status, n, reasons, last.Format("2006-01-02 15:04"))
		}
	case "dismiss":
		fmt.Printf("dismissed %d reports\n", exec(ctx, pool, `UPDATE reports SET status = 'dismissed' WHERE reported_id = $1 AND status = 'open'`))
	case "suspend":
		fmt.Printf("suspended %d account(s)\n", exec(ctx, pool, `UPDATE users SET status = 'suspended', updated_at = NOW() WHERE id = $1`))
		fmt.Printf("revoked %d session(s)\n", exec(ctx, pool, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`))
		fmt.Printf("closed %d report(s)\n", exec(ctx, pool, `UPDATE reports SET status = 'reviewed' WHERE reported_id = $1 AND status = 'open'`))
	case "unsuspend":
		fmt.Printf("restored %d account(s)\n", exec(ctx, pool, `UPDATE users SET status = 'active', updated_at = NOW() WHERE id = $1`))
	default:
		usage()
	}
}

// exec runs a statement parameterised by the user id given on the command line.
func exec(ctx context.Context, pool *pgxpool.Pool, query string) int64 {
	if len(os.Args) < 3 {
		usage()
	}
	tag, err := pool.Exec(ctx, query, os.Args[2])
	if err != nil {
		fatal(err.Error())
	}
	return tag.RowsAffected()
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: moderation reports | dismiss <userId> | suspend <userId> | unsuspend <userId>")
	os.Exit(2)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "moderation:", msg)
	os.Exit(1)
}

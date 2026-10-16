// Command moderate is the operator's tool for handling reports. There is no admin UI
// on purpose: moderation is a small, auditable CLI that talks to the same database.
//
//	moderate reports [-status open|reviewed|dismissed|all]
//	moderate dismiss <report-id>
//	moderate suspend <user-id> [-report <report-id>]
//	moderate restore <user-id>
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "reports":
		fs := flag.NewFlagSet("reports", flag.ExitOnError)
		status := fs.String("status", "open", "open, reviewed, dismissed or all")
		_ = fs.Parse(args)
		listReports(ctx, pool, *status)
	case "dismiss":
		need(args, 1)
		setReport(ctx, pool, args[0], "dismissed")
	case "suspend":
		fs := flag.NewFlagSet("suspend", flag.ExitOnError)
		report := fs.String("report", "", "report to mark as reviewed")
		need(args, 1)
		_ = fs.Parse(args[1:])
		suspend(ctx, pool, args[0], *report)
	case "restore":
		need(args, 1)
		restore(ctx, pool, args[0])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: moderate reports [-status open|reviewed|dismissed|all] | dismiss <report-id> | suspend <user-id> [-report <report-id>] | restore <user-id>")
	os.Exit(2)
}

func need(args []string, n int) {
	if len(args) < n {
		usage()
	}
}

func listReports(ctx context.Context, pool *pgxpool.Pool, status string) {
	rows, err := pool.Query(ctx, `
		SELECT r.id, r.created_at, r.status, r.reason, r.details,
		       r.reported_id, COALESCE(p.first_name, ''), (u.banned_at IS NOT NULL),
		       (SELECT COUNT(*) FROM reports x WHERE x.reported_id = r.reported_id)
		FROM reports r
		JOIN users u ON u.id = r.reported_id
		LEFT JOIN profiles p ON p.user_id = r.reported_id
		WHERE $1 = 'all' OR r.status = $1
		ORDER BY r.created_at DESC
		LIMIT 200`, status)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, st, reason, details, reported, name string
		var at time.Time
		var banned bool
		var total int
		if err := rows.Scan(&id, &at, &st, &reason, &details, &reported, &name, &banned, &total); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s  %s  [%s] %s\n  reported: %s (%s) suspended=%v, %d report(s) total\n  details:  %q\n",
			id, at.Format(time.RFC3339), st, reason, reported, name, banned, total, details)
		n++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d report(s)\n", n)
}

func setReport(ctx context.Context, pool *pgxpool.Pool, id, status string) {
	tag, err := pool.Exec(ctx, `UPDATE reports SET status = $2, reviewed_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d report(s) marked %s\n", tag.RowsAffected(), status)
}

// suspend blocks sign-in, ends every session, hides the profile and dissolves the
// user's matches (and with them their conversations). Messages already sent are kept
// until the matches go; the account itself is kept as evidence.
func suspend(ctx context.Context, pool *pgxpool.Pool, userID, reportID string) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `UPDATE users SET banned_at = COALESCE(banned_at, NOW()) WHERE id = $1`, userID)
	if err != nil {
		log.Fatal(err)
	}
	if tag.RowsAffected() == 0 {
		log.Fatalf("no such user: %s", userID)
	}
	for _, q := range []string{
		`UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`,
		`UPDATE profiles SET discoverable = FALSE WHERE user_id = $1`,
		`DELETE FROM matches WHERE user_a = $1 OR user_b = $1`,
		`DELETE FROM swipes WHERE actor_id = $1`,
	} {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			log.Fatal(err)
		}
	}
	if reportID != "" {
		if _, err := tx.Exec(ctx, `UPDATE reports SET status = 'reviewed', reviewed_at = NOW() WHERE id = $1`, reportID); err != nil {
			log.Fatal(err)
		}
	} else if _, err := tx.Exec(ctx, `UPDATE reports SET status = 'reviewed', reviewed_at = NOW() WHERE reported_id = $1 AND status = 'open'`, userID); err != nil {
		log.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("user %s suspended (sessions revoked, profile hidden, matches removed)\n", userID)
}

func restore(ctx context.Context, pool *pgxpool.Pool, userID string) {
	tag, err := pool.Exec(ctx, `UPDATE users SET banned_at = NULL WHERE id = $1`, userID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d user(s) restored; they must re-enable discovery from their privacy settings\n", tag.RowsAffected())
}

// Package testutil provides helpers for tests only: an isolated Postgres
// database per test and synthetic images.
package testutil

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/url"
	"os"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDB returns a pool on a freshly created and migrated database. The test is
// skipped when DATABASE_URL is unset. When the role may not create databases,
// the database named by DATABASE_URL is used as is.
func NewDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()

	target := base
	if u, err := url.Parse(base); err == nil {
		admin := *u
		admin.Path = "/postgres"
		buf := make([]byte, 6)
		_, _ = rand.Read(buf)
		name := "t_" + hex.EncodeToString(buf)
		if adminPool, err := pgxpool.New(ctx, admin.String()); err == nil {
			if _, err := adminPool.Exec(ctx, `CREATE DATABASE `+name); err == nil {
				child := *u
				child.Path = "/" + name
				target = child.String()
				t.Cleanup(func() {
					c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
					defer cancel()
					_, _ = adminPool.Exec(c, `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
					adminPool.Close()
				})
			} else {
				adminPool.Close()
			}
		}
	}

	pool, err := pgxpool.New(ctx, target)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return pool
}

// JPEG returns a gradient JPEG of the given size.
func JPEG(w, h int) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, gradient(w, h), &jpeg.Options{Quality: 80}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// PNG returns a gradient PNG, optionally with a transparent corner.
func PNG(w, h int) []byte {
	img := gradient(w, h)
	img.SetRGBA(0, 0, color.RGBA{})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// GIF returns a valid GIF (which the API must reject).
func GIF() []byte {
	pal := color.Palette{color.Black, color.White}
	img := image.NewPaletted(image.Rect(0, 0, 8, 8), pal)
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func gradient(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: 160, A: 255})
		}
	}
	return img
}

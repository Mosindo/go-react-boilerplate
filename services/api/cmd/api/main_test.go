package main

import (
	"io"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("TEST_VERBOSE_LOGS") == "" {
		log.SetOutput(io.Discard)
	}
	os.Exit(m.Run())
}

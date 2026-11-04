package chat

import (
	"strings"
	"testing"
)

func TestCleanMessage(t *testing.T) {
	got, err := CleanMessage("  salut\x00 \x07toi \n ")
	if err != nil || got != "salut toi" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := CleanMessage(" \n\t "); err == nil {
		t.Fatal("blank must be rejected")
	}
	if _, err := CleanMessage(strings.Repeat("é", MaxMessageLength+1)); err == nil {
		t.Fatal("over-long message must be rejected (counted in characters, not bytes)")
	}
	if _, err := CleanMessage(strings.Repeat("é", MaxMessageLength)); err != nil {
		t.Fatalf("2000 characters must be accepted: %v", err)
	}
}

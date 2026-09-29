package profiles_test

import (
	"net/url"
	"strconv"
	"testing"
)

func mustExp(t *testing.T, raw string) int64 {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.ParseInt(u.Query().Get("exp"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func mustSig(raw string) string {
	u, _ := url.Parse(raw)
	return u.Query().Get("sig")
}

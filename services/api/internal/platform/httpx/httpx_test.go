package httpx

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	got, err := DecodeCursor(Cursor{At: at, ID: "abc"}.Encode())
	if err != nil || got == nil || !got.At.Equal(at) || got.ID != "abc" {
		t.Fatalf("round trip failed: %+v %v", got, err)
	}
	if c, err := DecodeCursor(""); c != nil || err != nil {
		t.Fatal("empty cursor means first page")
	}
	for _, bad := range []string{"!!!", "bm90LWEtY3Vyc29y"} {
		if _, err := DecodeCursor(bad); err == nil {
			t.Fatalf("%q should be rejected", bad)
		}
	}
}

func TestPaginate(t *testing.T) {
	page := Paginate([]int{1, 2, 3}, 2, func(i int) Cursor { return Cursor{At: time.Unix(int64(i), 0), ID: "x"} })
	if len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("unexpected page %+v", page)
	}
	last := Paginate([]int{1, 2}, 2, func(i int) Cursor { return Cursor{} })
	if last.NextCursor != nil || len(last.Items) != 2 {
		t.Fatalf("unexpected last page %+v", last)
	}
	empty := Paginate[int](nil, 2, func(int) Cursor { return Cursor{} })
	if empty.Items == nil {
		t.Fatal("items must be [] not null")
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("123e4567-e89b-12d3-a456-426614174000") || !IsUUID("123E4567-E89B-12D3-A456-426614174000") {
		t.Fatal("valid uuid rejected")
	}
	for _, bad := range []string{"", "x", "123e4567e89b12d3a456426614174000", "123e4567-e89b-12d3-a456-42661417400g", "123e4567-e89b-12d3-a456-4266141740000"} {
		if IsUUID(bad) {
			t.Fatalf("%q accepted", bad)
		}
	}
}

package matches

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	c := Cursor{SortAt: time.Now().UTC().Truncate(time.Microsecond), ID: uuid.NewString()}
	got, err := decodeCursor(encodeCursor(c))
	if err != nil || !got.SortAt.Equal(c.SortAt) || got.ID != c.ID {
		t.Fatalf("round trip failed: %+v %v", got, err)
	}
	if got, err := decodeCursor(""); got != nil || err != nil {
		t.Fatal("empty cursor means first page")
	}
	for _, bad := range []string{"!!!", "YWJj", "MTIzfG5vdC1hLXV1aWQ"} {
		if _, err := decodeCursor(bad); err == nil {
			t.Errorf("cursor %q should be rejected", bad)
		}
	}
}

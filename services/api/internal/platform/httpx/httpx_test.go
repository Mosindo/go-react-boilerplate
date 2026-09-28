package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestUUID(t *testing.T) {
	if id, ok := NormalizeUUID("6F9619FF-8B86-D011-B42D-00C04FC964FF"); !ok || id != "6f9619ff-8b86-d011-b42d-00c04fc964ff" {
		t.Errorf("normalize: %q %v", id, ok)
	}
	for _, bad := range []string{"", "x", "6f9619ff8b86d011b42d00c04fc964ff", "6f9619ff-8b86-d011-b42d-00c04fc964f", "6f9619ff-8b86-d011-b42d-00c04fc964fg", "' OR 1=1 --"} {
		if IsUUID(bad) {
			t.Errorf("%q must not be a UUID", bad)
		}
	}
}

func TestCursorRoundTrip(t *testing.T) {
	type cur struct {
		T  time.Time `json:"t"`
		ID string    `json:"id"`
	}
	in := cur{T: time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC), ID: "abc"}
	var out cur
	if err := DecodeCursor(EncodeCursor(in), &out); err != nil || !out.T.Equal(in.T) || out.ID != in.ID {
		t.Fatalf("round trip: %+v %v", out, err)
	}
	for _, bad := range []string{"%%%", "e30", "bm90LWpzb24", string(make([]byte, 600))} {
		var c cur
		err := DecodeCursor(bad, &c)
		if bad == "e30" { // {} decodes fine; validity of fields is checked by the services
			continue
		}
		if err == nil {
			t.Errorf("DecodeCursor(%q) should fail", bad)
		}
	}
}

func TestParseLimitAndFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for q, want := range map[string]int{"": 20, "limit=1": 1, "limit=50": 50} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/x?"+q, nil)
		got, err := ParseLimit(c, 20, 50)
		if err != nil || got != want {
			t.Errorf("ParseLimit(%q) = %d, %v", q, got, err)
		}
	}
	for _, q := range []string{"limit=0", "limit=51", "limit=-1", "limit=x", "limit=1.5"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/x?"+q, nil)
		if _, err := ParseLimit(c, 20, 50); err == nil {
			t.Errorf("ParseLimit(%q) should fail", q)
		}
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x", nil)
	Fail(c, Blocked())
	if w.Code != http.StatusForbidden || w.Body.String() != `{"code":"blocked","error":"this conversation is not available"}` {
		t.Errorf("app error: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x", nil)
	Fail(c, http.ErrAbortHandler)
	if w.Code != 500 || w.Body.String() != `{"code":"internal","error":"internal error"}` {
		t.Errorf("unknown error must be opaque: %d %s", w.Code, w.Body.String())
	}
}

package realtime

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOrigin(t *testing.T) {
	req := func(origin, host string) *http.Request {
		r := httptest.NewRequest("GET", "http://"+host+"/ws", nil)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		return r
	}
	strict := NewHandler(NewHub(), nil, nil, []string{"https://app.example.com"})
	for origin, want := range map[string]bool{
		"":                        true, // native apps send no Origin
		"https://app.example.com": true,
		"https://APP.example.com": true,
		"https://evil.example":    false,
		"http://app.example.com":  false,
		"null":                    false,
	} {
		if got := strict.checkOrigin(req(origin, "api.example.com")); got != want {
			t.Errorf("allowlist: origin %q => %v, want %v", origin, got, want)
		}
	}
	open := NewHandler(NewHub(), nil, nil, nil)
	if !open.checkOrigin(req("http://api.example.com", "api.example.com")) {
		t.Error("same host is allowed when no allowlist is configured")
	}
	if open.checkOrigin(req("https://evil.example", "api.example.com")) {
		t.Error("foreign origin refused without allowlist")
	}
}

func TestHubPublishWithoutClientsIsNoop(t *testing.T) {
	h := NewHub()
	h.Publish("nobody", "message.new", map[string]string{"a": "b"})
	h.Publish("nobody", "message.new", make(chan int)) // unmarshalable: must not panic
	if h.Connections("nobody") != 0 {
		t.Error("no connections expected")
	}
	h.Close()
}

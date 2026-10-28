package realtime

import (
	"net/http"
	"testing"
)

func TestHubPublishAndDropSlowConsumer(t *testing.T) {
	h := NewHub()
	fast := &client{userID: "u", send: make(chan []byte, sendBuffer)}
	slow := &client{userID: "u", send: make(chan []byte, 1)}
	h.add(fast)
	h.add(slow)

	h.Publish("u", Event{Type: "a"})
	h.Publish("u", Event{Type: "b"}) // overflows `slow`
	if len(fast.send) != 2 {
		t.Fatalf("fast client should get both events, got %d", len(fast.send))
	}
	if _, open := <-slow.send; !open {
		t.Fatal("first event should still be readable")
	}
	if _, open := <-slow.send; open {
		t.Fatal("slow consumer must be dropped and its channel closed")
	}
	h.remove(fast)
	if h.Connected("u") {
		t.Fatal("user should be disconnected")
	}
	h.Publish("nobody", Event{Type: "x"}) // must not panic
}

func TestOriginChecker(t *testing.T) {
	check := originChecker([]string{"https://app.example.com/"})
	req := func(origin, host string) *http.Request {
		r, _ := http.NewRequest("GET", "http://"+host+"/ws", nil)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		return r
	}
	if !check(req("", "api.local")) {
		t.Error("native clients without Origin are allowed")
	}
	if !check(req("https://app.example.com", "api.local")) {
		t.Error("configured origin allowed")
	}
	if !check(req("http://api.local", "api.local")) {
		t.Error("same host allowed")
	}
	if check(req("https://evil.example", "api.local")) {
		t.Error("foreign browser origin must be refused")
	}
}

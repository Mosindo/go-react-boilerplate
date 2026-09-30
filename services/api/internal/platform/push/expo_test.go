package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExpoSenderParsesTickets(t *testing.T) {
	var received []Message
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Errorf("missing access token")
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		_, _ = w.Write([]byte(`{"data":[{"status":"ok","id":"1"},{"status":"error","message":"x","details":{"error":"DeviceNotRegistered"}}]}`))
	}))
	defer server.Close()

	sender := NewExpoSender(server.URL, "secret-token")
	results, err := sender.Send(context.Background(), []Message{{To: "ExponentPushToken[aaaaaaaaaaaa]", Title: "t"}, {To: "ExponentPushToken[bbbbbbbbbbbb]", Title: "t"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(received) != 2 || received[0].Title != "t" {
		t.Fatalf("unexpected payload %+v", received)
	}
	if len(results) != 2 || results[0].Unregistered || !results[1].Unregistered {
		t.Fatalf("unexpected results %+v", results)
	}
}

func TestExpoSenderReportsHTTPErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	if _, err := NewExpoSender(server.URL, "").Send(context.Background(), []Message{{To: "x"}}); err == nil {
		t.Fatal("expected an error on non-200 responses")
	}
}

// Package push delivers native push notifications through the Expo Push
// service (which relays to APNs and FCM). No secret is required for the
// default Expo endpoint; an optional access token can be configured.
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const DefaultExpoURL = "https://exp.host/--/api/v2/push/send"

type Message struct {
	To    string            `json:"to"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
	Sound string            `json:"sound,omitempty"`
	// ChannelID matches the Android channel created by the app.
	ChannelID string `json:"channelId,omitempty"`
}

// Result reports, per token, whether the device is no longer registered and
// the token should be forgotten.
type Result struct {
	Token        string
	Unregistered bool
}

type Sender interface {
	Send(ctx context.Context, messages []Message) ([]Result, error)
}

// NopSender is used when push is disabled.
type NopSender struct{}

func (NopSender) Send(context.Context, []Message) ([]Result, error) { return nil, nil }

type ExpoSender struct {
	url         string
	accessToken string
	client      *http.Client
}

func NewExpoSender(url, accessToken string) *ExpoSender {
	if url == "" {
		url = DefaultExpoURL
	}
	return &ExpoSender{url: url, accessToken: accessToken, client: &http.Client{Timeout: 10 * time.Second}}
}

type expoTicket struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details struct {
		Error string `json:"error"`
	} `json:"details"`
}

func (s *ExpoSender) Send(ctx context.Context, messages []Message) ([]Result, error) {
	if len(messages) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.accessToken)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expo push: status %d", res.StatusCode)
	}
	var body struct {
		Data []expoTicket `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("expo push: decode: %w", err)
	}
	results := make([]Result, 0, len(messages))
	for i, ticket := range body.Data {
		if i >= len(messages) {
			break
		}
		results = append(results, Result{
			Token:        messages[i].To,
			Unregistered: ticket.Status == "error" && ticket.Details.Error == "DeviceNotRegistered",
		})
	}
	return results, nil
}

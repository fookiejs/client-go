package fookie

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type wsMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func (c *Client) Subscribe(ctx context.Context, query string, variables map[string]interface{}) (<-chan SubscriptionEvent, error) {
	wsURL := strings.Replace(c.baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL += "/graphql/ws"

	headers := http.Header{}
	headers.Set("Sec-WebSocket-Protocol", "graphql-transport-ws")

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("fookie: ws dial: %w", err)
	}

	initPayload := map[string]string{}
	if c.token != "" {
		initPayload["token"] = c.token
	}
	if c.adminKey != "" {
		initPayload["adminKey"] = c.adminKey
	}
	initPayloadJSON, _ := json.Marshal(initPayload)
	if err := conn.WriteJSON(wsMessage{Type: "connection_init", Payload: initPayloadJSON}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("fookie: ws init: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var ack wsMessage
	if err := conn.ReadJSON(&ack); err != nil || ack.Type != "connection_ack" {
		conn.Close()
		return nil, fmt.Errorf("fookie: ws ack failed (got type=%s)", ack.Type)
	}
	conn.SetReadDeadline(time.Time{})

	payloadBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	payloadJSON, _ := json.Marshal(payloadBody)
	if err := conn.WriteJSON(wsMessage{ID: "1", Type: "subscribe", Payload: payloadJSON}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("fookie: ws subscribe: %w", err)
	}

	ch := make(chan SubscriptionEvent, 64)

	go func() {
		defer close(ch)
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				conn.WriteJSON(wsMessage{ID: "1", Type: "complete"})
				return
			default:
			}

			var msg wsMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if ctx.Err() == nil {
					ch <- SubscriptionEvent{Error: fmt.Errorf("fookie: ws read: %w", err)}
				}
				return
			}

			switch msg.Type {
			case "next":
				var wrapper struct {
					Data map[string]interface{} `json:"data"`
				}
				if err := json.Unmarshal(msg.Payload, &wrapper); err != nil {
					ch <- SubscriptionEvent{Error: err}
					continue
				}
				ch <- SubscriptionEvent{Data: wrapper.Data}
			case "error":
				ch <- SubscriptionEvent{Error: fmt.Errorf("fookie: subscription error: %s", string(msg.Payload))}
			case "complete":
				return
			case "ping":
				conn.WriteJSON(wsMessage{Type: "pong"})
			}
		}
	}()

	return ch, nil
}

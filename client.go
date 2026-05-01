package fookie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL    string
	token      string
	adminKey   string
	httpClient *http.Client
}

func New(baseURL, token, adminKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		adminKey:   adminKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	return c.do(ctx, query, "", variables, result)
}

func (c *Client) Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error {
	return c.do(ctx, mutation, "", variables, result)
}

func (c *Client) do(ctx context.Context, query, opName string, variables map[string]interface{}, result interface{}) error {
	body, err := json.Marshal(GraphQLRequest{
		Query:         query,
		OperationName: opName,
		Variables:     variables,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.adminKey != "" {
		req.Header.Set("X-Fookie-Admin-Key", c.adminKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var gqlResp struct {
		Data   json.RawMessage `json:"data"`
		Errors []GraphQLError  `json:"errors"`
	}
	if err := json.Unmarshal(raw, &gqlResp); err != nil {
		return fmt.Errorf("fookie: decode error: %w (body: %s)", err, string(raw))
	}
	if len(gqlResp.Errors) > 0 {
		return gqlResp.Errors[0]
	}
	if result != nil && gqlResp.Data != nil {
		return json.Unmarshal(gqlResp.Data, result)
	}
	return nil
}

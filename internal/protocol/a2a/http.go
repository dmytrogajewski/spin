package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sync/atomic"
)

// ProtocolBindingHTTPS is the remote HTTPS JSON-RPC binding name on AgentInterface.
const ProtocolBindingHTTPS = "HTTPS-JSONRPC"

// ErrNotAllowlisted is returned when a card or RPC URL is not on the allowlist.
var ErrNotAllowlisted = errors.New("a2a: url not on allowlist")

// ErrNotHTTPS is returned when a remote URL is not https.
var ErrNotHTTPS = errors.New("a2a: url must be https")

// Allowlist is the exact set of https card/RPC URLs permitted to be dialed.
type Allowlist []string

// Contains reports whether rawURL is an exact allowlist entry.
func (list Allowlist) Contains(rawURL string) bool {
	return slices.Contains(list, rawURL)
}

// HTTPClient is an A2A JSON-RPC client over HTTPS.
type HTTPClient struct {
	allowlist Allowlist
	card      AgentCard
	endpoint  string
	http      *http.Client
	nextID    atomic.Int64
}

// HTTPOption configures DialHTTP.
type HTTPOption func(*HTTPClient)

// WithHTTPClient injects an HTTP client (tests use [httptest.Server.Client]).
func WithHTTPClient(client *http.Client) HTTPOption {
	return func(httpClient *HTTPClient) {
		httpClient.http = client
	}
}

// Card returns the Agent Card fetched at DialHTTP.
func (httpClient *HTTPClient) Card() AgentCard {
	return httpClient.card
}

// DialHTTP fetches a remote Agent Card over HTTPS if cardURL is allowlisted.
func DialHTTP(ctx context.Context, cardURL string, allowlist Allowlist, opts ...HTTPOption) (*HTTPClient, error) {
	httpClient := &HTTPClient{allowlist: allowlist, endpoint: cardURL}
	for _, opt := range opts {
		opt(httpClient)
	}

	if httpClient.http == nil {
		httpClient.http = &http.Client{}
	}

	httpClient.guardRedirects()

	if !allowlist.Contains(cardURL) {
		return nil, fmt.Errorf("%w: %s", ErrNotAllowlisted, cardURL)
	}

	if err := requireHTTPS(cardURL); err != nil {
		return nil, err
	}

	if err := httpClient.fetchCard(ctx, cardURL); err != nil {
		return nil, err
	}

	if err := httpClient.bindRPCEndpoint(); err != nil {
		return nil, err
	}

	return httpClient, nil
}

func (httpClient *HTTPClient) guardRedirects() {
	base := *httpClient.http
	previous := base.CheckRedirect
	base.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		target := request.URL.String()
		if !httpClient.allowlist.Contains(target) {
			return fmt.Errorf("%w: %s", ErrNotAllowlisted, target)
		}

		if previous != nil {
			if prevErr := previous(request, via); prevErr != nil {
				return fmt.Errorf("redirect: %w", prevErr)
			}
		}

		return nil
	}
	httpClient.http = &base
}

func (httpClient *HTTPClient) bindRPCEndpoint() error {
	for _, iface := range httpClient.card.SupportedInterfaces {
		if iface.ProtocolBinding != ProtocolBindingHTTPS || iface.URL == "" {
			continue
		}

		if err := requireHTTPS(iface.URL); err != nil {
			return err
		}

		if !httpClient.allowlist.Contains(iface.URL) {
			return fmt.Errorf("%w: %s", ErrNotAllowlisted, iface.URL)
		}

		httpClient.endpoint = iface.URL

		return nil
	}

	return nil
}

func (httpClient *HTTPClient) fetchCard(ctx context.Context, cardURL string) error {
	request, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, cardURL, http.NoBody)
	if reqErr != nil {
		return fmt.Errorf("a2a card request: %w", reqErr)
	}

	response, doErr := httpClient.http.Do(request)
	if doErr != nil {
		return fmt.Errorf("a2a card get: %w", doErr)
	}

	defer func() { _ = response.Body.Close() }()

	if decodeErr := json.NewDecoder(response.Body).Decode(&httpClient.card); decodeErr != nil {
		return fmt.Errorf("a2a card decode: %w", decodeErr)
	}

	return nil
}

// Call sends a JSON-RPC request over HTTPS and decodes the result.
func (httpClient *HTTPClient) Call(ctx context.Context, method string, params, result any) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("call %s: %w", method, err)
	}

	body, buildErr := httpClient.marshalRequest(method, params)
	if buildErr != nil {
		return buildErr
	}

	return httpClient.postCall(ctx, method, body, result)
}

func (httpClient *HTTPClient) marshalRequest(method string, params any) ([]byte, error) {
	paramsRaw, marshalErr := json.Marshal(params)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal %s params: %w", method, marshalErr)
	}

	idRaw, idErr := json.Marshal(httpClient.nextID.Add(1))
	if idErr != nil {
		return nil, fmt.Errorf("marshal %s id: %w", method, idErr)
	}

	body, envErr := json.Marshal(envelope{
		JSONRPC: jsonRPCVersion,
		ID:      idRaw,
		Method:  method,
		Params:  paramsRaw,
	})
	if envErr != nil {
		return nil, fmt.Errorf("marshal %s envelope: %w", method, envErr)
	}

	return body, nil
}

func (httpClient *HTTPClient) postCall(ctx context.Context, method string, body []byte, result any) error {
	request, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, httpClient.endpoint, bytes.NewReader(body))
	if reqErr != nil {
		return fmt.Errorf("post %s: %w", method, reqErr)
	}

	request.Header.Set("Content-Type", "application/json")

	response, doErr := httpClient.http.Do(request)
	if doErr != nil {
		return fmt.Errorf("post %s: %w", method, doErr)
	}

	defer func() { _ = response.Body.Close() }()

	return decodeCallResult(method, response, result)
}

func decodeCallResult(method string, response *http.Response, result any) error {
	var env envelope
	if decodeErr := json.NewDecoder(response.Body).Decode(&env); decodeErr != nil {
		return fmt.Errorf("decode %s: %w", method, decodeErr)
	}

	if env.Error != nil {
		return env.Error
	}

	if result == nil || len(env.Result) == 0 {
		return nil
	}

	if unmarshalErr := json.Unmarshal(env.Result, result); unmarshalErr != nil {
		return fmt.Errorf("decode %s result: %w", method, unmarshalErr)
	}

	return nil
}

// SendMessage calls message/send and returns the created Task.
func (httpClient *HTTPClient) SendMessage(ctx context.Context, message Message) (*Task, error) {
	var out SendMessageResult
	if err := httpClient.Call(ctx, MethodMessageSend, SendMessageParams{Message: message}, &out); err != nil {
		return nil, err
	}

	if out.Task == nil {
		return nil, NewRPCError(CodeInvalidAgentResponse, msgInvalidAgentResponse)
	}

	return out.Task, nil
}

// GetTask calls tasks/get.
func (httpClient *HTTPClient) GetTask(ctx context.Context, taskID string) (*Task, error) {
	var task Task
	if err := httpClient.Call(ctx, MethodTasksGet, GetTaskParams{ID: taskID}, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

func requireHTTPS(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%w: %s", ErrNotHTTPS, rawURL)
	}

	return nil
}

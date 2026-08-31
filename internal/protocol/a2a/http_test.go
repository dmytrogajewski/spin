package a2a

// Journey: specs/journeys/JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

var errProbeDial = errors.New("probe: transport must not dial")

type probeTransport struct {
	hits int
}

func (probe *probeTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	probe.hits++

	return nil, errProbeDial
}

func TestDialHTTP_RejectsOffAllowlistBeforeDial(t *testing.T) {
	t.Parallel()

	probe := &probeTransport{}
	_, err := DialHTTP(context.Background(), "https://evil.example/card", nil, WithHTTPClient(&http.Client{
		Transport: probe,
	}))
	require.ErrorIs(t, err, ErrNotAllowlisted)
	require.Zero(t, probe.hits)
}

func newTLSCardServer(t *testing.T, card AgentCard) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(writer, "method", http.StatusMethodNotAllowed)

			return
		}

		writeJSON(writer, card)
	}))
	t.Cleanup(server.Close)

	return server
}

func TestDialHTTP_FetchesAllowlistedCard(t *testing.T) {
	t.Parallel()

	card := fixtureCard()
	server := newTLSCardServer(t, card)
	card.SupportedInterfaces = []AgentInterface{{
		URL:             server.URL,
		ProtocolBinding: ProtocolBindingHTTPS,
		ProtocolVersion: ProtocolVersion,
	}}

	client, err := DialHTTP(context.Background(), server.URL, Allowlist{server.URL}, WithHTTPClient(server.Client()))
	require.NoError(t, err)
	require.Equal(t, card.Name, client.Card().Name)
}

func TestHTTPClient_SendAndGetTask(t *testing.T) {
	t.Parallel()

	server := newTLSA2AServer(t, NewMemoryHandler(fixtureCard()))
	client, err := DialHTTP(context.Background(), server.URL, Allowlist{server.URL}, WithHTTPClient(server.Client()))
	require.NoError(t, err)

	task, err := client.SendMessage(context.Background(), Message{
		MessageID: "msg-remote-1",
		Role:      RoleUser,
		Parts:     []Part{{Text: "hello-remote", MediaType: mediaTextPlain}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, task.ID)

	got, err := client.GetTask(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, task.ID, got.ID)
	require.Equal(t, TaskStateCompleted, got.Status.State)
}

func newTLSA2AServer(t *testing.T, handler Handler) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			writeJSON(writer, handler.Card())

			return
		}

		if request.Method != http.MethodPost {
			http.Error(writer, "method", http.StatusMethodNotAllowed)

			return
		}

		serveHTTPSJSONRPC(writer, request, handler)
	}))
	t.Cleanup(server.Close)

	return server
}

func serveHTTPSJSONRPC(writer http.ResponseWriter, request *http.Request, handler Handler) {
	var env envelope
	if err := json.NewDecoder(request.Body).Decode(&env); err != nil {
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	result, rpcErr := handler.Handle(request.Context(), env.Method, env.Params)
	if rpcErr != nil {
		writeJSON(writer, envelope{
			JSONRPC: jsonRPCVersion,
			ID:      env.ID,
			Error:   rpcErr,
		})

		return
	}

	writeJSON(writer, envelope{
		JSONRPC: jsonRPCVersion,
		ID:      env.ID,
		Result:  result,
	})
}

func writeJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")

	if encErr := json.NewEncoder(writer).Encode(value); encErr != nil {
		http.Error(writer, encErr.Error(), http.StatusInternalServerError)
	}
}

func TestDialHTTP_RejectsOffAllowlistRedirect(t *testing.T) {
	t.Parallel()

	const offList = "https://169.254.169.254/latest/meta-data"

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, offList, http.StatusFound)
	}))
	t.Cleanup(server.Close)

	cardURL := server.URL
	_, err := DialHTTP(context.Background(), cardURL, Allowlist{cardURL}, WithHTTPClient(server.Client()))
	require.ErrorIs(t, err, ErrNotAllowlisted)
}

func TestDialHTTP_FollowsAllowlistedRedirect(t *testing.T) {
	t.Parallel()

	card := fixtureCard()

	var server *httptest.Server

	mux := http.NewServeMux()
	mux.HandleFunc("/from", func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, server.URL+"/card", http.StatusFound)
	})
	mux.HandleFunc("/card", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, card)
	})

	server = httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)

	fromURL := server.URL + "/from"
	cardURL := server.URL + "/card"
	client, err := DialHTTP(context.Background(), fromURL, Allowlist{fromURL, cardURL}, WithHTTPClient(server.Client()))
	require.NoError(t, err)
	require.Equal(t, card.Name, client.Card().Name)
}

func TestDialHTTP_RejectsOffListInterfaceURL(t *testing.T) {
	t.Parallel()

	card := fixtureCard()
	card.SupportedInterfaces = []AgentInterface{{
		URL:             "https://evil.example/rpc",
		ProtocolBinding: ProtocolBindingHTTPS,
		ProtocolVersion: ProtocolVersion,
	}}

	server := newTLSCardServer(t, card)
	_, err := DialHTTP(context.Background(), server.URL, Allowlist{server.URL}, WithHTTPClient(server.Client()))
	require.ErrorIs(t, err, ErrNotAllowlisted)
}

func TestDialHTTP_RejectsHTTPSchemeBeforeDial(t *testing.T) {
	t.Parallel()

	const rawURL = "http://evil.example/card"

	probe := &probeTransport{}
	_, err := DialHTTP(context.Background(), rawURL, Allowlist{rawURL}, WithHTTPClient(&http.Client{
		Transport: probe,
	}))
	require.ErrorIs(t, err, ErrNotHTTPS)
	require.Zero(t, probe.hits)
}

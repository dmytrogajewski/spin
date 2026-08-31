package child

// Journey: specs/journeys/JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

var errProbeDial = errors.New("child: transport must not dial")

func TestDialRemote_RejectsOffAllowlist(t *testing.T) {
	t.Parallel()

	probe := &childProbeTransport{}
	_, err := DialRemote(context.Background(), "https://evil.example/card", nil, a2a.WithHTTPClient(&http.Client{
		Transport: probe,
	}))
	require.ErrorIs(t, err, a2a.ErrNotAllowlisted)
	require.Zero(t, probe.hits)
}

type childProbeTransport struct {
	hits int
}

func (probe *childProbeTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	probe.hits++

	return nil, errProbeDial
}

func TestDialRemote_SendAndGetTask(t *testing.T) {
	t.Parallel()

	handler := a2a.NewMemoryHandler(a2a.AgentCard{
		Name:    "remote-child",
		Version: a2a.ProtocolVersion,
	})
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		serveChildHTTPS(writer, request, handler)
	}))
	t.Cleanup(server.Close)

	client, err := DialRemote(context.Background(), server.URL, []string{server.URL}, a2a.WithHTTPClient(server.Client()))
	require.NoError(t, err)

	task, err := client.SendMessage(context.Background(), a2a.Message{
		MessageID: "child-remote-1",
		Role:      a2a.RoleUser,
		Parts:     []a2a.Part{{Text: "ping"}},
	})
	require.NoError(t, err)

	got, err := client.GetTask(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, task.ID, got.ID)
}

func serveChildHTTPS(writer http.ResponseWriter, request *http.Request, handler a2a.Handler) {
	if request.Method == http.MethodGet {
		writeChildJSON(writer, handler.Card())

		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(request.Body).Decode(&raw); err != nil {
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	var method string
	if unmarshalErr := json.Unmarshal(raw["method"], &method); unmarshalErr != nil {
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	result, rpcErr := handler.Handle(request.Context(), method, raw["params"])
	reply := childRPCReply{JSONRPC: "2.0", ID: raw["id"], Result: result, Error: rpcErr}
	writeChildJSON(writer, reply)
}

type childRPCReply struct {
	Error   *a2a.RPCError   `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func writeChildJSON(writer http.ResponseWriter, value any) {
	if encErr := json.NewEncoder(writer).Encode(value); encErr != nil {
		http.Error(writer, encErr.Error(), http.StatusInternalServerError)
	}
}

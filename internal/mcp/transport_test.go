package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		transport TransportType
		want      string
	}{
		{
			name:      "stdio transport",
			transport: TransportStdio,
			want:      "stdio",
		},
		{
			name:      "sse transport",
			transport: TransportSSE,
			want:      "sse",
		},
		{
			name:      "streamable-http transport",
			transport: TransportStreamableHTTP,
			want:      "streamable-http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, string(tt.transport))
		})
	}
}

func TestTransportType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		transport TransportType
		want      bool
	}{
		{
			name:      "empty is valid (defaults to stdio)",
			transport: "",
			want:      true,
		},
		{
			name:      "stdio is valid",
			transport: TransportStdio,
			want:      true,
		},
		{
			name:      "sse is valid",
			transport: TransportSSE,
			want:      true,
		},
		{
			name:      "streamable-http is valid",
			transport: TransportStreamableHTTP,
			want:      true,
		},
		{
			name:      "unknown is invalid",
			transport: "websocket",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.transport.IsValid())
		})
	}
}

func TestParsePluginTransport_ExplicitOnly(t *testing.T) {
	t.Parallel()

	stdio, err := ParsePluginTransport("stdio")
	require.NoError(t, err)
	assert.Equal(t, TransportStdio, stdio)

	httpTransport, err := ParsePluginTransport("streamable-http")
	require.NoError(t, err)
	assert.Equal(t, TransportStreamableHTTP, httpTransport)

	sse, err := ParsePluginTransport("sse")
	require.NoError(t, err)
	assert.Equal(t, TransportSSE, sse)

	_, err = ParsePluginTransport("")
	require.ErrorIs(t, err, ErrTransportRequired)

	_, err = ParsePluginTransport("websocket")
	require.ErrorIs(t, err, ErrUnsupportedTransport)
}

func TestTransportType_IsRemote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		transport TransportType
		want      bool
	}{
		{
			name:      "empty is not remote",
			transport: "",
			want:      false,
		},
		{
			name:      "stdio is not remote",
			transport: TransportStdio,
			want:      false,
		},
		{
			name:      "sse is remote",
			transport: TransportSSE,
			want:      true,
		},
		{
			name:      "streamable-http is remote",
			transport: TransportStreamableHTTP,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.transport.IsRemote())
		})
	}
}

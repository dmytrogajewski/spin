package child

// Journey: specs/journeys/JOURNEY-017-local-a2a-server-process.md.

import (
	"context"
	"math"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const (
	cardP99Samples = 100
	cardP99Percent = 99
	cardP99Limit   = 200 * time.Millisecond
	percentScale   = 100
)

func TestServer_CardReceivedP99(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	durations := make([]time.Duration, cardP99Samples)
	for i := range cardP99Samples {
		durations[i] = measureCardReceived(t, spec)
	}

	p99 := percentile(durations, cardP99Percent)
	if p99 >= cardP99Limit {
		t.Fatalf("card-received p99 %s is not < %s", p99, cardP99Limit)
	}
}

func measureCardReceived(t *testing.T, spec *subagent.Spec) time.Duration {
	t.Helper()

	clientConn, serverConn := net.Pipe()

	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	server, err := NewServer(spec)
	require.NoError(t, err)

	go func() {
		_ = server.Serve(ctx, serverConn, serverConn)
	}()

	start := time.Now()
	client, err := a2a.NewClient(clientConn, clientConn)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Equal(t, spec.Name, client.Card().Name)

	return elapsed
}

func percentile(durations []time.Duration, pct int) time.Duration {
	sorted := slices.Clone(durations)
	slices.Sort(sorted)

	idx := max(int(math.Ceil(float64(len(sorted))*float64(pct)/percentScale))-1, 0)
	idx = min(idx, len(sorted)-1)

	return sorted[idx]
}

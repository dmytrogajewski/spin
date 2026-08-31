package child

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	// ListenUnixPrefix is the documented alternate binding scheme.
	ListenUnixPrefix = "unix://"
	networkUnix      = "unix"
)

// ErrUnsupportedListen is returned when --listen is not unix://.
var ErrUnsupportedListen = errors.New("child: listen must be unix://PATH")

// ParseListen accepts unix://PATH as the alternate A2A binding.
func ParseListen(addr string) (network, path string, err error) {
	if !strings.HasPrefix(addr, ListenUnixPrefix) {
		return "", "", fmt.Errorf("%w: %q", ErrUnsupportedListen, addr)
	}

	path = strings.TrimPrefix(addr, ListenUnixPrefix)
	if path == "" {
		return "", "", fmt.Errorf("%w: empty path", ErrUnsupportedListen)
	}

	return networkUnix, path, nil
}

// ListenAndServe binds unix://PATH and serves one A2A connection.
func (server *Server) ListenAndServe(ctx context.Context, addr string) error {
	network, path, err := ParseListen(addr)
	if err != nil {
		return err
	}

	if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("child listen remove: %w", removeErr)
	}

	listenCfg := net.ListenConfig{}

	listener, listenErr := listenCfg.Listen(ctx, network, path)
	if listenErr != nil {
		return fmt.Errorf("child listen: %w", listenErr)
	}

	defer func() { _ = listener.Close() }()

	go func() {
		<-ctx.Done()

		_ = listener.Close()
	}()

	conn, acceptErr := listener.Accept()
	if acceptErr != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("child listen: %w", ctx.Err())
		}

		return fmt.Errorf("child accept: %w", acceptErr)
	}

	defer func() { _ = conn.Close() }()

	return server.Serve(ctx, conn, conn)
}

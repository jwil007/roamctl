package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"
)

func Listen() (net.Listener, func(), error) {
	path := "/run/roamctl/roamctl.sock"
	_ = os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, nil, fmt.Errorf("net.Listen: %w", err)
	}
	cleanup := func() {
		_ = os.Remove(path)
	}
	return listener, cleanup, nil
}

func Serve(
	ctx context.Context,
	listener net.Listener,
	procCh chan ProcessState) {
	var conns []net.Conn
	connCh := handleConnections(ctx, listener)
	for {
		select {
		case <-ctx.Done():
			return
		case conn := <-connCh:
			conns = append(conns, conn)
		case p := <-procCh:
			var healthy []net.Conn
			for _, c := range conns {
				_ = c.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
				err := json.NewEncoder(c).Encode(p)
				if err != nil {
					slog.Error("Error enconding json", "err", err)
				} else {
					healthy = append(healthy, c)
				}
			}
			conns = healthy
		}
	}
}

func handleConnections(ctx context.Context, l net.Listener) <-chan net.Conn {
	connCh := make(chan net.Conn)
	go func() {
		go func() {
			<-ctx.Done()
			_ = l.Close()
		}()
		for {
			conn, err := l.Accept()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				slog.Error("Listener Accept err:", err)
				continue
			}
			connCh <- conn
		}
	}()
	return connCh
}

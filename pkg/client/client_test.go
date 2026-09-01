package client_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/Muxcore-Media/cache-local/internal/cache"
	"github.com/Muxcore-Media/cache-local/internal/server"
	"github.com/Muxcore-Media/cache-local/pkg/client"
	"github.com/Muxcore-Media/core/pkg/contracts"
)

const bufSize = 1 << 20

func startBufconnServer(t *testing.T) *bufconn.Listener {
	t.Helper()
	c := cache.New()
	t.Cleanup(c.Close)
	srv := server.New(c)
	maxRecv := int(server.MaxValueBytes + 1024)
	gs := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxRecv),
		grpc.MaxSendMsgSize(maxRecv),
	)
	srv.RegisterWithGRPC(gs)
	lis := bufconn.Listen(bufSize)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(func() {
		gs.Stop()
		_ = lis.Close()
	})
	return lis
}

func dialClient(t *testing.T, lis *bufconn.Listener) *client.Client {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return client.New(conn)
}

func TestClientGetSetInvalidate(t *testing.T) {
	lis := startBufconnServer(t)
	cl := dialClient(t, lis)
	ctx := context.Background()

	if _, ok := cl.Get(ctx, "k"); ok {
		t.Fatal("expected miss")
	}
	if err := cl.Set(ctx, "k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	got, ok := cl.Get(ctx, "k")
	if !ok || string(got) != "v" {
		t.Fatalf("hit: ok=%v got=%q", ok, got)
	}
	if err := cl.Invalidate(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, ok := cl.Get(ctx, "k"); ok {
		t.Fatal("expected miss after invalidate")
	}
}

var _ contracts.CacheLayer = (*client.Client)(nil)

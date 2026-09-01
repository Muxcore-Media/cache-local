package test

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
)

const bufSize = 1 << 20

// In-process CacheLayer round-trip (no live core required).
func TestCacheLayerInProcess(t *testing.T) {
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

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	cl := client.New(conn)
	ctx := context.Background()

	if err := cl.Set(ctx, "obj:1", []byte("payload")); err != nil {
		t.Fatal(err)
	}
	got, ok := cl.Get(ctx, "obj:1")
	if !ok || string(got) != "payload" {
		t.Fatalf("get: ok=%v got=%q", ok, got)
	}
	if err := cl.Invalidate(ctx, "obj:"); err != nil {
		t.Fatal(err)
	}
	if _, ok := cl.Get(ctx, "obj:1"); ok {
		t.Fatal("expected miss after invalidate")
	}
}

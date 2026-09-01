package server_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/Muxcore-Media/cache-local/internal/cache"
	"github.com/Muxcore-Media/cache-local/internal/server"
	cachev1 "github.com/Muxcore-Media/core/proto/gen/muxcore/cache/v1"
)

func startServer(t *testing.T, c *cache.Cache) (cachev1.CacheLayerServiceClient, func()) {
	t.Helper()
	srv := server.New(c)
	maxRecv := int(server.MaxValueBytes + 1024)
	grpcSrv := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxRecv),
		grpc.MaxSendMsgSize(maxRecv),
	)
	srv.RegisterWithGRPC(grpcSrv)

	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = grpcSrv.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	cleanup := func() {
		_ = conn.Close()
		grpcSrv.GracefulStop()
	}
	return cachev1.NewCacheLayerServiceClient(conn), cleanup
}

func TestCacheLayerService_RoundTrip(t *testing.T) {
	c := cache.New()
	t.Cleanup(c.Close)
	client, cleanup := startServer(t, c)
	t.Cleanup(cleanup)
	ctx := context.Background()

	miss, err := client.Get(ctx, &cachev1.GetCacheLayerRequest{Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if miss.GetFound() {
		t.Fatal("expected miss")
	}

	if _, err := client.Set(ctx, &cachev1.SetCacheLayerRequest{Key: "k", Value: []byte("v")}); err != nil {
		t.Fatal(err)
	}
	hit, err := client.Get(ctx, &cachev1.GetCacheLayerRequest{Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if !hit.GetFound() || string(hit.GetValue()) != "v" {
		t.Fatalf("hit: found=%v value=%q", hit.GetFound(), hit.GetValue())
	}

	if _, err := client.Set(ctx, &cachev1.SetCacheLayerRequest{Key: "k2", Value: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Invalidate(ctx, &cachev1.InvalidateCacheLayerRequest{Prefix: "k"}); err != nil {
		t.Fatal(err)
	}
	after, err := client.Get(ctx, &cachev1.GetCacheLayerRequest{Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if after.GetFound() {
		t.Fatal("expected invalidate")
	}
}

func TestCacheLayerService_EmptyKey(t *testing.T) {
	c := cache.New()
	t.Cleanup(c.Close)
	client, cleanup := startServer(t, c)
	t.Cleanup(cleanup)
	ctx := context.Background()

	_, err := client.Get(ctx, &cachev1.GetCacheLayerRequest{Key: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Get empty key: code=%v err=%v", status.Code(err), err)
	}
	_, err = client.Set(ctx, &cachev1.SetCacheLayerRequest{Key: "", Value: []byte("x")})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Set empty key: code=%v err=%v", status.Code(err), err)
	}
}

func TestCacheLayerService_EmptyPrefixInvalidate(t *testing.T) {
	c := cache.New()
	t.Cleanup(c.Close)
	client, cleanup := startServer(t, c)
	t.Cleanup(cleanup)
	ctx := context.Background()

	if _, err := client.Set(ctx, &cachev1.SetCacheLayerRequest{Key: "keep", Value: []byte("1")}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Invalidate(ctx, &cachev1.InvalidateCacheLayerRequest{Prefix: ""}); err != nil {
		t.Fatal(err)
	}
	hit, err := client.Get(ctx, &cachev1.GetCacheLayerRequest{Key: "keep"})
	if err != nil {
		t.Fatal(err)
	}
	if !hit.GetFound() {
		t.Fatal("empty prefix invalidate should be no-op")
	}
}

func TestCacheLayerService_OversizeValue(t *testing.T) {
	c := cache.New()
	t.Cleanup(c.Close)
	client, cleanup := startServer(t, c)
	t.Cleanup(cleanup)
	ctx := context.Background()

	oversize := make([]byte, server.MaxValueBytes+1)
	_, err := client.Set(ctx, &cachev1.SetCacheLayerRequest{Key: "big", Value: oversize})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("oversize Set: code=%v err=%v", status.Code(err), err)
	}
}

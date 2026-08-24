package server_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Muxcore-Media/cache-local/internal/cache"
	"github.com/Muxcore-Media/cache-local/internal/server"
	cachev1 "github.com/Muxcore-Media/core/proto/gen/muxcore/cache/v1"
)

func TestCacheLayerService_RoundTrip(t *testing.T) {
	c := cache.New()
	t.Cleanup(c.Close)
	srv := server.New(c)

	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	srv.RegisterWithGRPC(grpcSrv)
	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(grpcSrv.GracefulStop)

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := cachev1.NewCacheLayerServiceClient(conn)
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

package client

import (
	"context"

	"google.golang.org/grpc"

	"github.com/Muxcore-Media/core/pkg/contracts"
	cachev1 "github.com/Muxcore-Media/core/proto/gen/muxcore/cache/v1"
)

// Client implements contracts.CacheLayer over CacheLayerService gRPC.
type Client struct {
	rpc cachev1.CacheLayerServiceClient
}

// New returns a CacheLayer client backed by conn.
func New(conn grpc.ClientConnInterface) *Client {
	return &Client{rpc: cachev1.NewCacheLayerServiceClient(conn)}
}

var _ contracts.CacheLayer = (*Client)(nil)

// Get returns cached data for key. The bool indicates a cache hit.
func (c *Client) Get(ctx context.Context, key string) ([]byte, bool) {
	resp, err := c.rpc.Get(ctx, &cachev1.GetCacheLayerRequest{Key: key})
	if err != nil || !resp.GetFound() {
		return nil, false
	}
	return resp.GetValue(), true
}

// Set caches data for key.
func (c *Client) Set(ctx context.Context, key string, data []byte) error {
	_, err := c.rpc.Set(ctx, &cachev1.SetCacheLayerRequest{Key: key, Value: data})
	return err
}

// Invalidate removes entries whose keys have the given prefix.
func (c *Client) Invalidate(ctx context.Context, prefix string) error {
	_, err := c.rpc.Invalidate(ctx, &cachev1.InvalidateCacheLayerRequest{Prefix: prefix})
	return err
}

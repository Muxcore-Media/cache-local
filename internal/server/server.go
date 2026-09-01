package server

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Muxcore-Media/cache-local/internal/cache"
	cachev1 "github.com/Muxcore-Media/core/proto/gen/muxcore/cache/v1"
)

// MaxValueBytes is the largest value accepted by Set (matches cache.DefaultMaxEntryBytes).
const MaxValueBytes = cache.DefaultMaxEntryBytes

type Server struct {
	cachev1.UnimplementedCacheLayerServiceServer
	cache *cache.Cache
}

func New(c *cache.Cache) *Server {
	return &Server{cache: c}
}

func (s *Server) RegisterWithGRPC(srv *grpc.Server) {
	cachev1.RegisterCacheLayerServiceServer(srv, s)
}

func (s *Server) Get(ctx context.Context, req *cachev1.GetCacheLayerRequest) (*cachev1.GetCacheLayerResponse, error) {
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}
	data, ok := s.cache.Get(ctx, req.GetKey())
	if !ok {
		return &cachev1.GetCacheLayerResponse{Found: false}, nil
	}
	return &cachev1.GetCacheLayerResponse{Value: data, Found: true}, nil
}

func (s *Server) Set(ctx context.Context, req *cachev1.SetCacheLayerRequest) (*cachev1.SetCacheLayerResponse, error) {
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}
	if int64(len(req.GetValue())) > MaxValueBytes {
		return nil, status.Errorf(codes.InvalidArgument, "value exceeds maximum size of %d bytes", MaxValueBytes)
	}
	if err := s.cache.Set(ctx, req.GetKey(), req.GetValue()); err != nil {
		switch {
		case errors.Is(err, cache.ErrEntryTooLarge):
			return nil, status.Errorf(codes.InvalidArgument, "value exceeds maximum size of %d bytes", MaxValueBytes)
		case errors.Is(err, cache.ErrCacheFull), errors.Is(err, cache.ErrTooManyKeys):
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		default:
			return nil, status.Error(codes.Internal, "set failed")
		}
	}
	return &cachev1.SetCacheLayerResponse{Status: "ok"}, nil
}

func (s *Server) Invalidate(ctx context.Context, req *cachev1.InvalidateCacheLayerRequest) (*cachev1.InvalidateCacheLayerResponse, error) {
	if err := s.cache.Invalidate(ctx, req.GetPrefix()); err != nil {
		return nil, status.Error(codes.Internal, "invalidate failed")
	}
	return &cachev1.InvalidateCacheLayerResponse{Status: "ok"}, nil
}

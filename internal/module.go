package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/Muxcore-Media/cache-local/internal/cache"
	"github.com/Muxcore-Media/cache-local/internal/server"
	"github.com/Muxcore-Media/core/pkg/contracts"
)

type Module struct {
	cache    *cache.Cache
	srv      *server.Server
	grpcSrv  *grpc.Server
	lis      net.Listener
	id       string
	grpcAddr string
}

type Config struct {
	ID       string
	GRPCAddr string
}

func NewModule(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "cache-local"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":9610"
	}
	if v := os.Getenv("CACHE_LOCAL_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	return &Module{
		id:       cfg.ID,
		grpcAddr: cfg.GRPCAddr,
	}
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID:           m.id,
		Name:         "Cache Local",
		Version:      "0.1.0",
		Roles:        []string{"infrastructure"},
		Description:  "In-memory local read-through cache for the storage orchestrator",
		Author:       "MuxCore",
		Capabilities: []string{contracts.CapabilityCacheLocal, "cache.memory"},
		HTTPAddr:     m.grpcAddr,
	}
}

func (m *Module) Init(ctx context.Context) error {
	m.cache = cache.New()
	m.srv = server.New(m.cache)

	lis, err := net.Listen("tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", m.grpcAddr, err)
	}
	m.lis = lis

	slog.Info("cache-local initialized", "addr", m.grpcAddr)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	m.grpcSrv = grpc.NewServer()
	m.srv.RegisterWithGRPC(m.grpcSrv)

	go func() {
		slog.Info("cache-local gRPC service started", "addr", m.grpcAddr)
		if err := m.grpcSrv.Serve(m.lis); err != nil {
			slog.Error("cache-local gRPC serve error", "error", err)
		}
	}()
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	if m.grpcSrv != nil {
		m.grpcSrv.GracefulStop()
	}
	if m.cache != nil {
		m.cache.Close()
	}
	slog.Info("cache-local stopped")
	return nil
}

func (m *Module) Health(ctx context.Context) error {
	if m.cache == nil {
		return fmt.Errorf("not initialized")
	}
	return nil
}

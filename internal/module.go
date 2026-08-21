package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/Muxcore-Media/cache-local/internal/cache"
	"github.com/Muxcore-Media/cache-local/internal/server"
	"github.com/Muxcore-Media/core/pkg/contracts"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
)

type Module struct {
	cache      *cache.Cache
	srv        *server.Server
	grpcSrv    *grpc.Server
	lis        net.Listener
	id         string
	grpcAddr   string
	cfgMu      sync.RWMutex
	defaultTTL time.Duration
}

type Config struct {
	ID         string
	GRPCAddr   string
	DefaultTTL time.Duration
}

func NewModule(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "cache-local"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":9602"
	}
	if v := os.Getenv("CACHE_LOCAL_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	ttl := cfg.DefaultTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if v := strings.TrimSpace(os.Getenv("CACHE_LOCAL_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			ttl = d
		}
	}
	return &Module{
		id:         cfg.ID,
		grpcAddr:   cfg.GRPCAddr,
		defaultTTL: ttl,
	}
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID:           m.id,
		Name:         "Cache Local",
		Version:      "0.1.1",
		Roles:        []string{"infrastructure"},
		Description:  "In-memory process-local CacheLayer (canonical cache.local; also advertises legacy cache.memory alias)",
		Author:       "MuxCore",
		Capabilities: []string{contracts.CapabilityCacheLocal, "cache.memory", "settings"},
		HTTPAddr:     m.grpcAddr,
	}
}

func (m *Module) Init(ctx context.Context) error {
	m.cfgMu.RLock()
	ttl := m.defaultTTL
	m.cfgMu.RUnlock()
	m.cache = cache.NewWithTTL(ttl)
	m.srv = server.New(m.cache)

	lis, err := net.Listen("tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", m.grpcAddr, err)
	}
	m.lis = lis

	slog.Info("cache-local initialized", "addr", m.grpcAddr, "default_ttl", ttl)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	m.grpcSrv = grpc.NewServer()
	m.srv.RegisterWithGRPC(m.grpcSrv)
	modulesdk.RegisterSettings(m.grpcSrv, m.id, m)

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

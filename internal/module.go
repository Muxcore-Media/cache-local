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
	"google.golang.org/grpc/credentials"

	manifest "github.com/Muxcore-Media/cache-local"
	"github.com/Muxcore-Media/cache-local/internal/cache"
	"github.com/Muxcore-Media/cache-local/internal/grpctls"
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
		cfg.GRPCAddr = "127.0.0.1:9602"
	}
	if v := os.Getenv("CACHE_LOCAL_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	cfg.GRPCAddr = resolveGRPCAddr(cfg.GRPCAddr)
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
		Version:      modulesdk.ManifestVersion(manifest.ManifestJSON),
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
	var grpcOpts []grpc.ServerOption
	tlsCfg, err := grpctls.ServerConfig()
	if err != nil {
		return fmt.Errorf("gRPC TLS: %w", err)
	}
	if tlsCfg != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(tlsCfg)))
		slog.Info("cache-local gRPC TLS enabled", "addr", m.grpcAddr)
	} else {
		slog.Warn("cache-local gRPC listening without TLS (dev only)",
			"addr", m.grpcAddr,
			"hint", "unset MUXCORE_INSECURE_DISABLE_TLS for production",
		)
	}
	m.grpcSrv = grpc.NewServer(grpcOpts...)
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

// resolveGRPCAddr prefers loopback when plaintext is explicitly enabled and the
// bind address would otherwise listen on all interfaces.
func resolveGRPCAddr(addr string) string {
	if !grpctls.InsecureAllowed() {
		return addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			return "127.0.0.1" + addr
		}
		return addr
	}
	if host == "" || host == "0.0.0.0" {
		return "127.0.0.1:" + port
	}
	return addr
}

func (m *Module) Health(ctx context.Context) error {
	if m.cache == nil {
		return fmt.Errorf("not initialized")
	}
	return nil
}

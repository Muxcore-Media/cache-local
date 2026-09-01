package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
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
	maxBytes   int64
	maxEntry   int64
	maxEntries int
	initErr    error
}

type Config struct {
	ID            string
	GRPCAddr      string
	DefaultTTL    time.Duration
	MaxBytes      int64
	MaxEntryBytes int64
	MaxEntries    int
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
	var initErr error
	if v := strings.TrimSpace(os.Getenv("CACHE_LOCAL_TTL")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			initErr = fmt.Errorf("invalid CACHE_LOCAL_TTL %q", v)
		} else {
			ttl = d
		}
	}

	maxBytes := cfg.MaxBytes
	if maxBytes <= 0 {
		maxBytes = envInt64("CACHE_LOCAL_MAX_BYTES", cache.DefaultMaxBytes)
	}
	maxEntry := cfg.MaxEntryBytes
	if maxEntry <= 0 {
		maxEntry = envInt64("CACHE_LOCAL_MAX_ENTRY_BYTES", cache.DefaultMaxEntryBytes)
	}
	maxEntries := cfg.MaxEntries
	if maxEntries <= 0 {
		maxEntries = envInt("CACHE_LOCAL_MAX_ENTRIES", 0)
	}

	return &Module{
		id:         cfg.ID,
		grpcAddr:   cfg.GRPCAddr,
		defaultTTL: ttl,
		maxBytes:   maxBytes,
		maxEntry:   maxEntry,
		maxEntries: maxEntries,
		initErr:    initErr,
	}
}

func envInt64(key string, def int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func (m *Module) Info() contracts.ModuleInfo {
	ver := Version
	return contracts.ModuleInfo{
		ID:           m.id,
		Name:         "Cache Local",
		Version:      ver,
		Roles:        []string{"infrastructure"},
		Description:  "In-memory process-local CacheLayer (canonical cache.local; also advertises legacy cache.memory alias)",
		Author:       "MuxCore",
		Capabilities: []string{contracts.CapabilityCacheLocal, "cache.memory", "settings"},
		HTTPAddr:     m.grpcAddr,
		Contracts: []contracts.ContractDeclaration{
			{
				Repo:      "github.com/Muxcore-Media/core/pkg/contracts",
				Interface: "CacheLayer",
				Version:   "v0.5.8",
			},
		},
		MinCoreVersion: MinCoreVersion,
	}
}

func (m *Module) Init(ctx context.Context) error {
	if m.initErr != nil {
		return m.initErr
	}
	m.cfgMu.RLock()
	ttl := m.defaultTTL
	maxBytes := m.maxBytes
	maxEntry := m.maxEntry
	maxEntries := m.maxEntries
	m.cfgMu.RUnlock()

	m.cache = cache.NewWithConfig(cache.Config{
		TTL:           ttl,
		MaxBytes:      maxBytes,
		MaxEntryBytes: maxEntry,
		MaxEntries:    maxEntries,
	})
	m.srv = server.New(m.cache)

	lis, err := net.Listen("tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", m.grpcAddr, err)
	}
	m.lis = lis

	slog.Info("cache-local initialized",
		"addr", m.grpcAddr,
		"default_ttl", ttl,
		"max_bytes", maxBytes,
		"max_entry_bytes", maxEntry,
		"max_entries", maxEntries,
	)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	maxRecv := int(server.MaxValueBytes + 1024)
	m.grpcSrv = grpc.NewServer(
		grpc.MaxRecvMsgSize(maxRecv),
		grpc.MaxSendMsgSize(maxRecv),
	)
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
	const key = "__healthcheck__"
	if err := m.cache.Set(ctx, key, []byte("ok")); err != nil {
		return fmt.Errorf("health set: %w", err)
	}
	got, ok := m.cache.Get(ctx, key)
	if !ok || string(got) != "ok" {
		return fmt.Errorf("health get failed")
	}
	if err := m.cache.Invalidate(ctx, key); err != nil {
		return fmt.Errorf("health invalidate: %w", err)
	}
	return nil
}

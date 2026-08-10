package internal

import (
	"context"
	"testing"
	"time"
)

func TestModuleInfo(t *testing.T) {
	m := NewModule(Config{})
	info := m.Info()
	if info.ID != "cache-local" {
		t.Errorf("ID = %q", info.ID)
	}
	if info.Version != "0.1.1" {
		t.Errorf("Version = %q", info.Version)
	}
	foundLocal, foundMemory, foundSettings := false, false, false
	for _, c := range info.Capabilities {
		if c == "cache.local" {
			foundLocal = true
		}
		if c == "cache.memory" {
			foundMemory = true
		}
		if c == "settings" {
			foundSettings = true
		}
	}
	if !foundLocal {
		t.Error("expected cache.local capability")
	}
	if !foundMemory {
		t.Error("expected legacy cache.memory alias")
	}
	if !foundSettings {
		t.Error("expected settings capability")
	}
	if info.HTTPAddr != ":9610" {
		t.Errorf("HTTPAddr = %q, want :9610", info.HTTPAddr)
	}
}

func TestSettings_DefaultTTL(t *testing.T) {
	m := NewModule(Config{})
	defs := m.Settings()
	if len(defs) != 1 || defs[0].Key != "default_ttl" {
		t.Fatalf("defs=%+v", defs)
	}
	if err := m.UpdateSetting("default_ttl", "30s"); err != nil {
		t.Fatal(err)
	}
	if m.Settings()[0].Value != "30s" {
		t.Fatalf("value=%q", m.Settings()[0].Value)
	}
	if err := m.UpdateSetting("default_ttl", "bogus"); err == nil {
		t.Fatal("expected parse error")
	}
	ctx := context.Background()
	m2 := NewModule(Config{GRPCAddr: "127.0.0.1:0", DefaultTTL: time.Minute})
	if err := m2.Init(ctx); err != nil {
		t.Fatal(err)
	}
	defer m2.Stop(ctx)
	if err := m2.UpdateSetting("CACHE_LOCAL_TTL", "10s"); err != nil {
		t.Fatal(err)
	}
	if got := m2.cache.DefaultTTL(); got != 10*time.Second {
		t.Fatalf("cache ttl=%v", got)
	}
}

func TestModuleLifecycle(t *testing.T) {
	m := NewModule(Config{GRPCAddr: "127.0.0.1:0"})
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Health(ctx); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

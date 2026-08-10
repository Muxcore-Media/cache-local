package internal

import (
	"context"
	"testing"
)

func TestModuleInfo(t *testing.T) {
	m := NewModule(Config{})
	info := m.Info()
	if info.ID != "cache-local" {
		t.Errorf("ID = %q", info.ID)
	}
	if info.Version == "" {
		t.Error("version must not be empty")
	}
	foundLocal, foundMemory := false, false
	for _, c := range info.Capabilities {
		if c == "cache.local" {
			foundLocal = true
		}
		if c == "cache.memory" {
			foundMemory = true
		}
	}
	if !foundLocal {
		t.Error("expected cache.local capability")
	}
	if !foundMemory {
		t.Error("expected legacy cache.memory alias")
	}
	if info.HTTPAddr != ":9610" {
		t.Errorf("HTTPAddr = %q, want :9610", info.HTTPAddr)
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

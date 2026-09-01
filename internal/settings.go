package internal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Muxcore-Media/cache-local/internal/cache"
	"github.com/Muxcore-Media/core/pkg/contracts"
)

func (m *Module) Settings() []contracts.SettingDef {
	return m.settingsDefs()
}

func (m *Module) UpdateSetting(key, value string) error {
	return m.updateSetting(key, value)
}

func (m *Module) settingsDefs() []contracts.SettingDef {
	m.cfgMu.RLock()
	ttl := m.defaultTTL
	maxBytes := m.maxBytes
	maxEntry := m.maxEntry
	maxEntries := m.maxEntries
	m.cfgMu.RUnlock()
	return []contracts.SettingDef{
		{
			Key:         "default_ttl",
			Label:       "Default TTL",
			Type:        contracts.SettingTypeString,
			Value:       ttl.String(),
			Default:     (5 * time.Minute).String(),
			Description: "TTL for new cache entries (CACHE_LOCAL_TTL), e.g. 5m, 30s, 1h",
			Group:       "Cache",
		},
		{
			Key:         "max_bytes",
			Label:       "Max total bytes",
			Type:        contracts.SettingTypeString,
			Value:       strconv.FormatInt(maxBytes, 10),
			Default:     strconv.FormatInt(cache.DefaultMaxBytes, 10),
			Description: "Total in-memory cache size cap (CACHE_LOCAL_MAX_BYTES)",
			Group:       "Cache",
		},
		{
			Key:         "max_entry_bytes",
			Label:       "Max entry bytes",
			Type:        contracts.SettingTypeString,
			Value:       strconv.FormatInt(maxEntry, 10),
			Default:     strconv.FormatInt(cache.DefaultMaxEntryBytes, 10),
			Description: "Per-entry size cap (CACHE_LOCAL_MAX_ENTRY_BYTES)",
			Group:       "Cache",
		},
		{
			Key:         "max_entries",
			Label:       "Max entries",
			Type:        contracts.SettingTypeString,
			Value:       strconv.Itoa(maxEntries),
			Default:     "0",
			Description: "Optional key count cap; 0 disables (CACHE_LOCAL_MAX_ENTRIES)",
			Group:       "Cache",
		},
	}
}

func (m *Module) updateSetting(key, value string) error {
	value = strings.TrimSpace(value)
	switch key {
	case "default_ttl", "CACHE_LOCAL_TTL":
		if value == "" {
			return fmt.Errorf("default_ttl must not be empty")
		}
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid default_ttl %q: %w", value, err)
		}
		if d <= 0 {
			return fmt.Errorf("default_ttl must be positive")
		}
		return m.setDefaultTTL(d)
	case "max_bytes", "CACHE_LOCAL_MAX_BYTES":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid max_bytes %q", value)
		}
		return m.setMaxBytes(n)
	case "max_entry_bytes", "CACHE_LOCAL_MAX_ENTRY_BYTES":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid max_entry_bytes %q", value)
		}
		return m.setMaxEntryBytes(n)
	case "max_entries", "CACHE_LOCAL_MAX_ENTRIES":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid max_entries %q", value)
		}
		return m.setMaxEntries(n)
	default:
		return fmt.Errorf("unknown setting %q", key)
	}
}

func (m *Module) setDefaultTTL(ttl time.Duration) error {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	m.defaultTTL = ttl
	if m.cache != nil {
		m.cache.SetDefaultTTL(ttl)
	}
	return nil
}

func (m *Module) setMaxBytes(n int64) error {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	m.maxBytes = n
	if m.cache != nil {
		m.cache.SetLimits(n, 0, -1)
	}
	return nil
}

func (m *Module) setMaxEntryBytes(n int64) error {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	m.maxEntry = n
	if m.cache != nil {
		m.cache.SetLimits(0, n, -1)
	}
	return nil
}

func (m *Module) setMaxEntries(n int) error {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	m.maxEntries = n
	if m.cache != nil {
		m.cache.SetLimits(0, 0, n)
	}
	return nil
}

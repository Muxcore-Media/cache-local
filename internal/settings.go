package internal

import (
	"fmt"
	"strings"
	"time"

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

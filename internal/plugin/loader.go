package plugin

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	goplugin "plugin"
	"strings"
)

const pluginSymbol = "NewPlugin"

// LoadFromDirectory scans a directory for .so plugin files and loads them
// into the registry. Each .so must export a NewPlugin() Plugin function.
// A bad plugin logs an error but does not prevent other plugins from loading.
func LoadFromDirectory(dir string, registry *Registry, logger *slog.Logger) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("plugin directory does not exist, skipping", "dir", dir)
			return nil
		}
		return fmt.Errorf("reading plugin directory %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".so") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if err := loadPlugin(path, registry); err != nil {
			logger.Error("failed to load plugin",
				"path", path,
				"error", err,
			)
			continue
		}
		logger.Info("loaded plugin from file", "path", path)
	}

	return nil
}

func loadPlugin(path string, registry *Registry) error {
	p, err := goplugin.Open(path)
	if err != nil {
		return fmt.Errorf("opening plugin %q: %w", path, err)
	}

	sym, err := p.Lookup(pluginSymbol)
	if err != nil {
		return fmt.Errorf("plugin %q missing %s symbol: %w", path, pluginSymbol, err)
	}

	newFn, ok := sym.(func() Plugin)
	if !ok {
		return fmt.Errorf("plugin %q: %s has wrong signature, expected func() Plugin", path, pluginSymbol)
	}

	plug := newFn()
	return registry.Register(plug, nil)
}

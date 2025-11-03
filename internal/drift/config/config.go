package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultHTTPListen      = "0.0.0.0:9090"
	defaultMetricsListen   = "127.0.0.1:9091"
	defaultBridgeName      = "vbr0"
	defaultExternalIF      = "auto" // auto-detect external interface
	defaultStateDir        = "~/.volant/drift"
	defaultBPFObject       = "drift_l4.bpf.o"
)

// Config captures runtime settings for the Drift control daemon.
type Config struct {
	HTTPListen       string
	MetricsListen    string
	BridgeName       string
	ExternalInterface string // External interface for public IP exposure (auto-detect if "auto")
	StateDir         string
	RoutesPath       string
	BPFObjectPath    string
	APIKey           string
}

// FromEnv loads configuration using environment variables with defaults.
func FromEnv() (Config, error) {
	cfg := Config{
		HTTPListen:        getenv("DRIFT_HTTP_LISTEN", defaultHTTPListen),
		MetricsListen:     getenv("DRIFT_METRICS_LISTEN", defaultMetricsListen),
		BridgeName:        getenv("DRIFT_BRIDGE", defaultBridgeName),
		ExternalInterface: getenv("DRIFT_EXTERNAL_IF", defaultExternalIF),
		StateDir:          expandPath(getenv("DRIFT_STATE_DIR", defaultStateDir)),
		RoutesPath:        expandPath(getenv("DRIFT_ROUTES_PATH", "")),
		BPFObjectPath:     expandPath(getenv("DRIFT_BPF_OBJECT", defaultBPFObject)),
		APIKey:            strings.TrimSpace(os.Getenv("DRIFT_API_KEY")),
	}

	if cfg.HTTPListen = strings.TrimSpace(cfg.HTTPListen); cfg.HTTPListen == "" {
		return Config{}, fmt.Errorf("http listen address required")
	}

	if cfg.MetricsListen = strings.TrimSpace(cfg.MetricsListen); cfg.MetricsListen == "" {
		cfg.MetricsListen = cfg.HTTPListen
	}

	if cfg.StateDir == "" {
		return Config{}, fmt.Errorf("state directory required")
	}

	if cfg.RoutesPath == "" {
		cfg.RoutesPath = filepath.Join(cfg.StateDir, "routes.json")
	}

	if cfg.BPFObjectPath == "" {
		return Config{}, fmt.Errorf("bpf object path required")
	}

	cfg.BPFObjectPath = resolveBPFObjectPath(cfg.BPFObjectPath, cfg.StateDir)

	// Auto-detect external interface if set to "auto"
	if strings.ToLower(strings.TrimSpace(cfg.ExternalInterface)) == "auto" {
		if extIf, err := detectExternalInterface(); err == nil && extIf != "" {
			cfg.ExternalInterface = extIf
		} else {
			// If auto-detection fails, disable external interface attachment
			cfg.ExternalInterface = ""
		}
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func expandPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return filepath.Clean(path)
}

func resolveBPFObjectPath(path, stateDir string) string {
	cleaned := filepath.Clean(path)
	if filepath.IsAbs(cleaned) {
		return cleaned
	}

	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidate := filepath.Join(base, cleaned)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
	}

	if stateDir != "" {
		return filepath.Join(stateDir, cleaned)
	}

	return cleaned
}

// detectExternalInterface finds the network interface used for the default route
func detectExternalInterface() (string, error) {
	// Read /proc/net/route to find default route (0.0.0.0)
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Skip header line
	if !scanner.Scan() {
		return "", fmt.Errorf("empty route table")
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		// Check if destination is 00000000 (0.0.0.0 - default route)
		if fields[1] == "00000000" {
			return fields[0], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("no default route found")
}

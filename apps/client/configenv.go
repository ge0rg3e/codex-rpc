package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Direct launches do not inherit Taskfile's dotenv configuration. Read only
// installation settings, without changing the process environment.
func localInstallationEnv() (map[string]string, error) {
	var paths []string
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Dir(executable)
		paths = append(paths, filepath.Join(dir, ".env"))
		if filepath.Base(dir) == "bin" {
			paths = append(paths, filepath.Join(filepath.Dir(dir), ".env"))
		}
	}
	paths = append(paths, ".env", filepath.Join("apps", "client", ".env"))
	return readInstallationEnv(paths)
}

func readInstallationEnv(paths []string) (map[string]string, error) {
	values := make(map[string]string)
	for _, path := range paths {
		file, err := os.Open(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("Read installation configuration: %w", err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			key, value, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
			key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
			if !ok || (key != "DISCORD_CLIENT_ID" && key != "CODEX_RPC_WEBSITE") {
				continue
			}
			value = strings.TrimSpace(value)
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			if _, exists := values[key]; !exists {
				values[key] = value
			}
		}
		err = scanner.Err()
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("Read installation configuration: %w", err)
		}
	}
	return values, nil
}

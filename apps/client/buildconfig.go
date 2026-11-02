package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Release builds embed the public installation configuration through linker flags.
// Never put OAuth client secrets, bot tokens, or other credentials here.
var bundledDiscordClientID string
var bundledLeaderboardURL string

func installationConfig() (string, string, error) {
	local, err := localInstallationEnv()
	if err != nil {
		return "", "", err
	}
	id, website := bundledDiscordClientID, bundledLeaderboardURL
	if id == "" {
		id = os.Getenv("DISCORD_CLIENT_ID")
	}
	if website == "" {
		website = os.Getenv("CODEX_RPC_WEBSITE")
	}
	if id == "" {
		id = local["DISCORD_CLIENT_ID"]
	}
	if website == "" {
		website = local["CODEX_RPC_WEBSITE"]
	}
	id = strings.TrimSpace(id)
	website = strings.TrimRight(strings.TrimSpace(website), "/")
	if website == "" {
		website = "http://localhost:5173"
	}
	if id != "" && !regexp.MustCompile(`^\d{17,22}$`).MatchString(id) {
		return "", "", fmt.Errorf("Invalid Discord application ID in this build")
	}
	if err := validateWebsite(website); err != nil {
		return "", "", fmt.Errorf("Invalid leaderboard origin in this build")
	}
	return id, website, nil
}
func originFingerprint(website string) string {
	sum := sha256.Sum256([]byte(strings.TrimRight(website, "/")))
	return hex.EncodeToString(sum[:])
}

type persistedConfig struct {
	Enabled       bool   `json:"enabled"`
	Token         string `json:"token"`
	TokenOrigin   string `json:"tokenOrigin"`
	LegacyWebsite string `json:"website,omitempty"`
}

func loadPreferences(data []byte, cfg *Config) error {
	var saved persistedConfig
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}
	cfg.Enabled = saved.Enabled
	origin := saved.TokenOrigin
	if origin == "" && saved.LegacyWebsite != "" {
		origin = originFingerprint(saved.LegacyWebsite)
	}
	if origin == originFingerprint(cfg.Website) {
		cfg.Token = saved.Token
	} else {
		cfg.Token = ""
	}
	return nil
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadInstallationEnv(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first.env"), filepath.Join(dir, "second.env")
	if err := os.WriteFile(first, []byte("# local settings\nDISCORD_CLIENT_ID='123456789012345678'\nUNRELATED=value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("DISCORD_CLIENT_ID=987654321098765432\nexport CODEX_RPC_WEBSITE=\"https://example.com\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	values, err := readInstallationEnv([]string{filepath.Join(dir, "missing.env"), first, second})
	if err != nil {
		t.Fatal(err)
	}
	if values["DISCORD_CLIENT_ID"] != "123456789012345678" || values["CODEX_RPC_WEBSITE"] != "https://example.com" || len(values) != 2 {
		t.Fatalf("incorrect local configuration: %v", values)
	}
}

func TestInstallationConfigEnvironmentWins(t *testing.T) {
	t.Setenv("DISCORD_CLIENT_ID", "123456789012345678")
	t.Setenv("CODEX_RPC_WEBSITE", "https://example.com")
	id, site, err := installationConfig()
	if err != nil || id != "123456789012345678" || site != "https://example.com" {
		t.Fatalf("environment configuration not respected: %s %s %v", id, site, err)
	}
}

// Opt-in check against an installed Discord desktop client; no account data is logged.
func TestDiscordLiveHandshake(t *testing.T) {
	if os.Getenv("CODEX_RPC_TEST_DISCORD") != "1" {
		t.Skip("set CODEX_RPC_TEST_DISCORD=1 to check the local Discord connection")
	}
	id, _, err := installationConfig()
	if err != nil || id == "" {
		t.Fatalf("Discord application ID unavailable: %v", err)
	}
	conn, err := dialDiscord()
	if err != nil {
		t.Fatal(err)
	}
	d := Discord{conn: conn}
	defer d.close()
	if err := d.write(0, map[string]any{"v": 1, "client_id": id}); err != nil {
		t.Fatal(err)
	}
	reply, err := d.read()
	if err != nil || reply["evt"] != "READY" {
		t.Fatalf("Discord did not accept the configured application: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	var collector Collector
	sessions, err := collector.Scan(filepath.Join(home, ".codex"))
	if err != nil {
		t.Fatal(err)
	}
	var latest Session
	for _, session := range sessions {
		if !session.Internal && session.Last.After(latest.Last) {
			latest = session
		}
	}
	if latest.ID == "" {
		t.Fatal("No Codex session detected")
	}
	latest.LifetimeSessions = len(sessions)
	for _, session := range sessions {
		latest.LifetimeTokens += session.Tokens
		latest.LifetimePrompts += session.Prompts
		latest.LifetimeTools += session.Tools
	}
	if err := d.command(discordActivity(latest, "", "")); err != nil {
		t.Fatalf("Discord did not accept the Codex activity: %v", err)
	}
}

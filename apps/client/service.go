package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	Enabled   bool   `json:"enabled"`
	DiscordID string `json:"-"`
	Website   string `json:"-"`
	Token     string `json:"token"`
}
type Snapshot struct {
	Enabled        bool              `json:"enabled"`
	Tokens         int64             `json:"tokens"`
	Sessions       int               `json:"sessions"`
	Active         bool              `json:"active"`
	Activity       string            `json:"activity"`
	ActivityLabel  string            `json:"activityLabel"`
	Source         string            `json:"source"`
	Login          string            `json:"login"`
	Rank           *int              `json:"rank"`
	Public         bool              `json:"public"`
	Presence       string            `json:"presence"`
	Sync           string            `json:"sync"`
	ScanError      string            `json:"scanError"`
	PairCode       string            `json:"pairCode"`
	ResetStatus    *CodexResetStatus `json:"resetStatus"`
	ResetError     string            `json:"resetError"`
	ResetCheckedAt *time.Time        `json:"resetCheckedAt"`
}
type PresenceService struct {
	mu             sync.Mutex
	work           sync.Mutex
	config         Config
	state          Snapshot
	path, home     string
	collector      Collector
	discord        Discord
	client         *http.Client
	wake           chan struct{}
	stop           chan struct{}
	done           chan struct{}
	pending        string
	pairExpires    time.Time
	lastSync       time.Time
	activityToken  string
	activitySynced map[int64]bool
	openURL        func(string) error
	quit           func()
}

func NewPresenceService() (*PresenceService, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	s := &PresenceService{path: filepath.Join(root, "codex-rpc", "config.json"), home: filepath.Join(home, ".codex"), client: &http.Client{Timeout: 8 * time.Second}, wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	if custom := os.Getenv("CODEX_HOME"); custom != "" {
		s.home = custom
	}
	id, site, err := installationConfig()
	if err != nil {
		return nil, err
	}
	s.config.DiscordID = id
	s.config.Website = site
	if b, err := os.ReadFile(s.path); err == nil {
		if err = loadPreferences(b, &s.config); err != nil {
			return nil, fmt.Errorf("invalid settings: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}
func validateWebsite(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("Enter the website origin, without a path")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")) {
		return fmt.Errorf("Use HTTPS, or HTTP on localhost for development")
	}
	return nil
}
func (s *PresenceService) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(persistedConfig{Enabled: s.config.Enabled, Token: s.config.Token, TokenOrigin: originFingerprint(s.config.Website)}, "", "  ")
	if err != nil {
		return err
	}
	temporary := s.path + ".tmp"
	if err = os.WriteFile(temporary, b, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, s.path)
}
func (s *PresenceService) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.state
	v.Enabled = s.config.Enabled
	return v
}
func (s *PresenceService) notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func (s *PresenceService) SetEnabled(enabled bool) error {
	s.mu.Lock()
	before := s.config
	s.config.Enabled = enabled
	err := s.saveLocked()
	if err != nil {
		s.config = before
	}
	s.mu.Unlock()
	s.notify()
	return err
}
func (s *PresenceService) SetPublic(public bool) error {
	s.work.Lock()
	defer s.work.Unlock()
	s.mu.Lock()
	connected := s.config.Token != ""
	s.mu.Unlock()
	if !connected {
		return fmt.Errorf("Connect GitHub before changing leaderboard visibility")
	}
	if err := s.request("POST", "profile", map[string]bool{"public": public}, nil); err != nil {
		return err
	}
	s.mu.Lock()
	s.state.Public = public
	if !public {
		s.state.Rank = nil
	}
	s.mu.Unlock()
	s.lastSync = time.Time{}
	s.notify()
	return nil
}
func (s *PresenceService) request(method, path string, body any, result any) error {
	s.mu.Lock()
	config := s.config
	s.mu.Unlock()
	var buffer bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buffer).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, config.Website+"/api/"+path, &buffer)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+config.Token)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Leaderboard unavailable. Try again later")
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return fmt.Errorf("Website returned %d; check setup or reconnect GitHub", response.StatusCode)
	}
	if result != nil {
		return json.NewDecoder(io.LimitReader(response.Body, 2*1024*1024)).Decode(result)
	}
	return nil
}
func (s *PresenceService) ConnectGitHub() (string, error) {
	s.work.Lock()
	defer s.work.Unlock()
	var result struct{ Token, Code, URL string }
	if err := s.request("POST", "device/start", map[string]string{}, &result); err != nil {
		return "", err
	}
	s.pending = result.Token
	s.pairExpires = time.Now().Add(10 * time.Minute)
	s.mu.Lock()
	s.state.PairCode = result.Code
	s.mu.Unlock()
	// Construct the browser URL locally; never trust a response URL to launch arbitrary protocols.
	s.mu.Lock()
	website := s.config.Website
	s.mu.Unlock()
	if err := s.openURL(website + "/?device=" + url.QueryEscape(result.Code)); err != nil {
		return result.Code, err
	}
	return result.Code, nil
}
func (s *PresenceService) Disconnect() error {
	s.work.Lock()
	defer s.work.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	before := s.config
	s.config.Token = ""
	if err := s.saveLocked(); err != nil {
		s.config = before
		return err
	}
	s.pending = ""
	s.state.Login = ""
	s.state.Public = false
	s.state.PairCode = ""
	s.state.Rank = nil
	s.state.Sync = "Disconnected"
	return nil
}
func (s *PresenceService) OpenLeaderboard() error {
	s.mu.Lock()
	site := s.config.Website
	s.mu.Unlock()
	return s.openURL(site + "/#leaderboard")
}
func (s *PresenceService) OpenResetSource() error {
	return s.openURL("https://codex-resets.com/")
}
func (s *PresenceService) Quit() { s.quit() }
func (s *PresenceService) run() {
	go func() {
		defer close(s.done)
		defer s.discord.close()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		resetTicker := time.NewTicker(15 * time.Minute)
		defer resetTicker.Stop()
		go s.refreshResetStatus()
		s.refresh()
		for {
			select {
			case <-s.stop:
				_ = s.discord.Update("", false, Session{}, "")
				return
			case <-ticker.C:
				s.refresh()
			case <-resetTicker.C:
				go s.refreshResetStatus()
			case <-s.wake:
				s.refresh()
			}
		}
	}()
}

func (s *PresenceService) refreshResetStatus() {
	status, err := fetchCodexResetStatus(s.client, codexResetStatusURL)
	checkedAt := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ResetCheckedAt = &checkedAt
	if err != nil {
		s.state.ResetError = "Reset status is temporarily unavailable"
		return
	}
	s.state.ResetStatus = status
	s.state.ResetError = ""
}

func (s *PresenceService) refresh() {
	s.work.Lock()
	defer s.work.Unlock()
	sessions, scanErr := s.collector.Scan(s.home)
	var total int64
	var latest Session
	for _, session := range sessions {
		total += session.Tokens
		if !session.Internal && session.Last.After(latest.Last) {
			latest = session
		}
	}
	latest.LifetimeTokens = total
	latest.LifetimeSessions = len(sessions)
	for _, session := range sessions {
		latest.LifetimeTools += session.Tools
		latest.LifetimePrompts += session.Prompts
	}
	activity, visible := activityAt(latest, time.Now())
	latest.Activity = activity
	active := visible && activity != "idle" && activity != "success" && activity != "error" && activity != "stopped"
	s.mu.Lock()
	s.state.Tokens = total
	s.state.Sessions = len(sessions)
	s.state.Active = active
	s.state.Activity = activity
	s.state.ActivityLabel = activityLabel(activity)
	s.state.Source = latest.Source
	s.state.ScanError = ""
	if scanErr != nil {
		s.state.ScanError = "Some Codex logs could not be read"
	}
	cfg := s.config
	s.mu.Unlock()
	if s.pending != "" {
		if time.Now().After(s.pairExpires) {
			s.pending = ""
			s.mu.Lock()
			s.state.PairCode = ""
			s.state.Sync = "Pairing expired. Connect GitHub again."
			s.mu.Unlock()
		} else {
			var result struct{ Pending bool }
			if err := s.request("POST", "device/poll", map[string]string{"token": s.pending}, &result); err == nil && !result.Pending {
				s.mu.Lock()
				before := s.config
				s.config.Token = s.pending
				if err := s.saveLocked(); err != nil {
					s.config = before
					s.state.Sync = "Could not save the account connection"
				} else {
					s.pending = ""
					s.state.PairCode = ""
					s.lastSync = time.Time{}
				}
				cfg = s.config
				s.mu.Unlock()
			}
		}
	}
	if cfg.Token != "" && time.Since(s.lastSync) >= time.Minute {
		syncErr := error(nil)
		for start := 0; start < len(sessions); start += 500 {
			end := min(start+500, len(sessions))
			if syncErr = s.request("POST", "usage", map[string]any{"sessions": sessions[start:end]}, nil); syncErr != nil {
				break
			}
		}
		var me struct {
			Login  string
			Rank   *int
			Public bool
		}
		if syncErr == nil {
			syncErr = s.syncActivity(sessions, cfg.Token)
		}
		if syncErr == nil {
			syncErr = s.request("GET", "me", nil, &me)
		}
		s.mu.Lock()
		if syncErr != nil {
			s.state.Sync = syncErr.Error()
		} else {
			s.state.Sync = "Synced just now"
			s.state.Login = me.Login
			s.state.Rank = me.Rank
			s.state.Public = me.Public
		}
		s.mu.Unlock()
		s.lastSync = time.Now()
	}
	s.mu.Lock()
	login := s.state.Login
	s.mu.Unlock()
	status := "Presence paused"
	if cfg.Enabled {
		if cfg.DiscordID == "" {
			status = "Discord presence is unavailable in this build"
			_ = s.discord.Update("", false, Session{}, "")
		} else if err := s.discord.Update(cfg.DiscordID, visible, latest, login, cfg.Website); err != nil {
			status = err.Error()
		} else if visible {
			status = "Discord · " + activityLabel(activity)
		} else {
			status = "Waiting for a Codex session"
		}
	} else {
		_ = s.discord.Update("", false, Session{}, "")
	}
	s.mu.Lock()
	s.state.Presence = status
	s.mu.Unlock()
}

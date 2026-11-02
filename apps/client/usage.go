package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Session struct {
	ID               string    `json:"id"`
	Tokens           int64     `json:"tokens"`
	Project          string    `json:"-"`
	Source           string    `json:"-"`
	Last             time.Time `json:"-"`
	Activity         string    `json:"-"`
	Started          time.Time `json:"-"`
	Model            string    `json:"-"`
	Effort           string    `json:"-"`
	InputTokens      int64     `json:"-"`
	CachedTokens     int64     `json:"-"`
	OutputTokens     int64     `json:"-"`
	ReasoningTokens  int64     `json:"-"`
	ContextTokens    int64     `json:"-"`
	ContextWindow    int64     `json:"-"`
	Tools            int       `json:"-"`
	Prompts          int       `json:"-"`
	CurrentTool      string    `json:"-"`
	Internal         bool      `json:"-"`
	LifetimeTokens   int64     `json:"-"`
	LifetimeSessions int       `json:"-"`
	LifetimeTools    int       `json:"-"`
	LifetimePrompts  int       `json:"-"`
	ActivityMinutes  []int64   `json:"-"`
}

type tokenUsage struct {
	Tokens    int64 `json:"total_tokens"`
	Input     int64 `json:"input_tokens"`
	Cached    int64 `json:"cached_input_tokens"`
	Output    int64 `json:"output_tokens"`
	Reasoning int64 `json:"reasoning_output_tokens"`
}
type cachedSession struct {
	size     int64
	modified time.Time
	session  Session
}
type Collector struct{ cache map[string]cachedSession }

// Only decode counters and metadata. Prompt bodies and code are never retained.
func parseSession(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 65536), 32*1024*1024)
	var session Session
	pending := map[string]string{}
	seenTools := map[string]bool{}
	minutes := map[int64]bool{}
	var previousEvent time.Time
	var eventPrompts, responsePrompts int
	countTool := func(id, name string) {
		if id != "" && !seenTools[id] {
			seenTools[id] = true
			session.Tools++
		}
		if name != "" {
			session.CurrentTool = name
		}
	}
	setActivity := func(state string, at time.Time) {
		if at.IsZero() || at.Before(session.Last) {
			return
		}
		session.Activity = state
		session.Last = at
	}
	for scanner.Scan() {
		var row struct {
			Type      string    `json:"type"`
			Timestamp time.Time `json:"timestamp"`
			Payload   struct {
				Timestamp    time.Time `json:"timestamp"`
				Model        string    `json:"model"`
				Effort       string    `json:"effort"`
				ThreadSource string    `json:"thread_source"`
				ID           string    `json:"id"`
				Type         string    `json:"type"`
				Cwd          string    `json:"cwd"`
				Originator   string    `json:"originator"`
				Name         string    `json:"name"`
				CallID       string    `json:"call_id"`
				Role         string    `json:"role"`
				Phase        string    `json:"phase"`
				Status       string    `json:"status"`
				Success      *bool     `json:"success"`
				ExitCode     *int      `json:"exit_code"`
				Info         *struct {
					Total         tokenUsage `json:"total_token_usage"`
					Last          tokenUsage `json:"last_token_usage"`
					ContextWindow int64      `json:"model_context_window"`
				} `json:"info"`
			} `json:"payload"`
		}
		if json.Unmarshal(scanner.Bytes(), &row) != nil {
			continue
		} // A live file may end with an incomplete record.
		if (row.Type == "event_msg" || row.Type == "response_item") && !row.Timestamp.IsZero() {
			minute := row.Timestamp.Unix() / 60
			minutes[minute] = true
			gap := row.Timestamp.Sub(previousEvent)
			working := session.Activity != "" && session.Activity != "success" && session.Activity != "error" && session.Activity != "stopped" && session.Activity != "idle"
			if working && !previousEvent.IsZero() && gap > 0 && gap <= 2*time.Minute {
				for cursor := previousEvent.Unix() / 60; cursor <= minute; cursor++ {
					minutes[cursor] = true
				}
			}
			if row.Timestamp.After(previousEvent) {
				previousEvent = row.Timestamp
			}
		}
		if row.Type == "session_meta" {
			session.Started = row.Payload.Timestamp
			if session.Started.IsZero() {
				session.Started = row.Timestamp
			}
			session.Internal = row.Payload.ThreadSource == "guardian_review"
			session.ID = row.Payload.ID
			session.Project = filepath.Base(row.Payload.Cwd)
			session.Source = "Codex CLI"
			if strings.Contains(strings.ToLower(row.Payload.Originator), "desktop") || strings.Contains(strings.ToLower(row.Payload.Originator), "app") {
				session.Source = "Codex desktop"
			}
		}
		if row.Type == "turn_context" {
			if row.Payload.Model != "" {
				session.Model = row.Payload.Model
			}
			session.Effort = row.Payload.Effort
			if row.Payload.Cwd != "" {
				session.Project = filepath.Base(row.Payload.Cwd)
			}
		}
		if row.Type == "event_msg" && row.Payload.Type == "token_count" && row.Payload.Info != nil {
			// These are cumulative totals, not deltas. Cached input and reasoning are already included.
			if n := row.Payload.Info.Total.Tokens; n > session.Tokens {
				session.Tokens = n
			}
			usage := row.Payload.Info.Total
			session.InputTokens = max(session.InputTokens, usage.Input)
			session.CachedTokens = max(session.CachedTokens, usage.Cached)
			session.OutputTokens = max(session.OutputTokens, usage.Output)
			session.ReasoningTokens = max(session.ReasoningTokens, usage.Reasoning)
			if row.Payload.Info.ContextWindow > 0 {
				session.ContextWindow = row.Payload.Info.ContextWindow
				session.ContextTokens = row.Payload.Info.Last.Tokens
			}
			if row.Timestamp.After(session.Last) {
				session.Last = row.Timestamp
			}
			if session.Activity == "" {
				session.Activity = "thinking"
			}
		}
		if row.Type == "event_msg" {
			if row.Payload.Type == "user_message" {
				eventPrompts++
			}
			switch row.Payload.Type {
			case "task_started", "turn_started", "user_message":
				clear(pending)
				setActivity("thinking", row.Timestamp)
			case "agent_reasoning", "agent_reasoning_raw_content", "agent_reasoning_section_break":
				setActivity(pendingActivity(pending), row.Timestamp)
			case "agent_message", "agent_message_delta":
				setActivity("coding", row.Timestamp)
			case "exec_command_begin", "mcp_tool_call_begin", "web_search_begin":
				countTool(row.Payload.CallID, row.Payload.Name)
				pending[row.Payload.CallID] = "running"
				setActivity("running", row.Timestamp)
			case "patch_apply_begin", "patch_apply_updated":
				countTool(row.Payload.CallID, "apply_patch")
				pending[row.Payload.CallID] = "editing"
				setActivity("editing", row.Timestamp)
			case "exec_approval_request", "apply_patch_approval_request", "request_permissions", "request_user_input", "elicitation_request":
				setActivity("waiting", row.Timestamp)
			case "exec_command_end", "patch_apply_end", "mcp_tool_call_end", "web_search_end":
				delete(pending, row.Payload.CallID)
				if (row.Payload.ExitCode != nil && *row.Payload.ExitCode != 0) || (row.Payload.Success != nil && !*row.Payload.Success) || row.Payload.Status == "failed" {
					setActivity("error", row.Timestamp)
				} else {
					setActivity(pendingActivity(pending), row.Timestamp)
				}
			case "task_complete", "turn_complete", "turn_completed":
				clear(pending)
				setActivity("success", row.Timestamp)
			case "error", "turn_failed":
				clear(pending)
				setActivity("error", row.Timestamp)
			case "turn_aborted", "shutdown_complete":
				clear(pending)
				setActivity("stopped", row.Timestamp)
			}
		}
		if row.Type == "response_item" {
			switch row.Payload.Type {
			case "reasoning":
				setActivity(pendingActivity(pending), row.Timestamp)
			case "function_call", "custom_tool_call":
				countTool(row.Payload.CallID, row.Payload.Name)
				state := toolActivity(row.Payload.Name)
				pending[row.Payload.CallID] = state
				setActivity(state, row.Timestamp)
			case "function_call_output", "custom_tool_call_output":
				delete(pending, row.Payload.CallID)
				setActivity(pendingActivity(pending), row.Timestamp)
			case "message":
				if row.Payload.Role == "user" {
					responsePrompts++
					clear(pending)
					setActivity("thinking", row.Timestamp)
				} else if row.Payload.Role == "assistant" {
					if row.Payload.Phase == "final_answer" {
						clear(pending)
						setActivity("success", row.Timestamp)
					} else {
						setActivity("coding", row.Timestamp)
					}
				}
			}
		}
	}
	session.Prompts = max(eventPrompts, responsePrompts)
	for minute := range minutes {
		session.ActivityMinutes = append(session.ActivityMinutes, minute)
	}
	return session, scanner.Err()
}

func (c *Collector) Scan(home string) ([]Session, error) {
	if c.cache == nil {
		c.cache = map[string]cachedSession{}
	}
	sessions := map[string]Session{}
	seen := map[string]bool{}
	var failures []error
	for _, folder := range []string{"sessions", "archived_sessions"} {
		root := filepath.Join(home, folder)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".jsonl") {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			seen[path] = true
			cached, ok := c.cache[path]
			if !ok || cached.size != info.Size() || !cached.modified.Equal(info.ModTime()) {
				s, err := parseSession(path)
				if err != nil {
					failures = append(failures, err)
					return nil
				}
				cached = cachedSession{info.Size(), info.ModTime(), s}
				c.cache[path] = cached
			}
			s := cached.session
			if s.ID != "" {
				old, exists := sessions[s.ID]
				if !exists {
					sessions[s.ID] = s
				} else {
					newest := old
					if s.Last.After(old.Last) {
						newest = s
					}
					newest.Tokens = max(old.Tokens, s.Tokens)
					sessions[s.ID] = newest
				}
			}
			return nil
		})
		if err != nil {
			failures = append(failures, err)
		}
	}
	for path := range c.cache {
		if !seen[path] {
			delete(c.cache, path)
		}
	}
	result := make([]Session, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, s)
	}
	return result, errors.Join(failures...)
}

func pendingActivity(pending map[string]string) string {
	for _, state := range []string{"waiting", "editing", "running"} {
		for _, value := range pending {
			if value == state {
				return state
			}
		}
	}
	return "thinking"
}

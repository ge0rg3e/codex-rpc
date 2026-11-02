package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type Discord struct {
	conn          net.Conn
	applicationID string
	sequence      int
	lastActivity  string
	lastSent      time.Time
}

func (d *Discord) close() {
	if d.conn != nil {
		d.conn.Close()
		d.conn = nil
	}
	d.lastActivity = ""
	d.lastSent = time.Time{}
}
func (d *Discord) write(op uint32, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header, op)
	binary.LittleEndian.PutUint32(header[4:], uint32(len(body)))
	d.conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, err = io.Copy(d.conn, io.MultiReader(bytes.NewReader(header), bytes.NewReader(body)))
	return err
}

func (d *Discord) read() (map[string]any, error) {
	for i := 0; i < 10; i++ {
		header := make([]byte, 8)
		if _, err := io.ReadFull(d.conn, header); err != nil {
			return nil, err
		}
		op, n := binary.LittleEndian.Uint32(header), binary.LittleEndian.Uint32(header[4:])
		if n > 1024*1024 {
			return nil, fmt.Errorf("Discord frame too large")
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(d.conn, body); err != nil {
			return nil, err
		}
		if op == 3 {
			if err := d.write(4, json.RawMessage(body)); err != nil {
				return nil, err
			}
			continue
		}
		if op == 2 {
			return nil, fmt.Errorf("Discord disconnected")
		}
		var message map[string]any
		if err := json.Unmarshal(body, &message); err != nil {
			return nil, err
		}
		return message, nil
	}
	return nil, fmt.Errorf("Discord did not reply")
}
func (d *Discord) Update(id string, active bool, s Session, login string, website ...string) error {
	if id != d.applicationID {
		d.close()
	}
	if !active {
		if d.conn != nil {
			_ = d.command(nil)
			d.close()
		}
		return nil
	}
	if d.conn == nil {
		conn, err := dialDiscord()
		if err != nil {
			return fmt.Errorf("Open Discord desktop to share your presence")
		}
		d.conn = conn
		d.applicationID = id
		if err = d.write(0, map[string]any{"v": 1, "client_id": id}); err != nil {
			d.close()
			return err
		}
		ready, err := d.read()
		if err != nil || ready["evt"] != "READY" {
			d.close()
			return fmt.Errorf("Discord rejected the application ID")
		}
	}
	site := ""
	if len(website) > 0 {
		site = website[0]
	}
	activity := discordActivity(s, login, site)
	encoded, _ := json.Marshal(activity)
	// Avoid repeated writes and rapid sprite cycling. Always send a reconnect's first state.
	if d.lastActivity == string(encoded) || (!d.lastSent.IsZero() && time.Since(d.lastSent) < 15*time.Second) {
		return nil
	}
	if err := d.command(activity); err != nil {
		d.close()
		return err
	}
	d.lastActivity = string(encoded)
	d.lastSent = time.Now()
	return nil
}

func discordActivity(s Session, login, website string) map[string]any {
	return discordActivityAt(s, login, website, time.Now())
}

func discordActivityAt(s Session, login, website string, now time.Time) map[string]any {
	state := s.Activity
	if state == "" {
		state = "thinking"
	}
	label := activityLabel(state)
	image := "codex-" + state
	details := label
	if s.Project != "" {
		details += " in " + s.Project
	}
	model := s.Model
	if s.Effort != "" && model != "" {
		model += " · " + s.Effort + " reasoning"
	}
	frames := []string{}
	if model != "" {
		frames = append(frames, model+" · "+compact(s.Tokens)+" tokens")
	}
	frames = append(frames, fmt.Sprintf("%s · %s session tokens", s.Source, compact(s.Tokens)))
	if s.InputTokens > 0 || s.OutputTokens > 0 {
		frames = append(frames, fmt.Sprintf("Tokens · %s input · %s output · %s cached", compact(s.InputTokens), compact(s.OutputTokens), compact(s.CachedTokens)))
	}
	if s.ReasoningTokens > 0 {
		frames = append(frames, fmt.Sprintf("Reasoning · %s tokens", compact(s.ReasoningTokens)))
	}
	if s.ContextWindow > 0 && s.ContextTokens > 0 {
		percent := min(100, float64(s.ContextTokens)*100/float64(s.ContextWindow))
		frames = append(frames, fmt.Sprintf("Context · %.0f%% · %s / %s tokens", percent, compact(s.ContextTokens), compact(s.ContextWindow)))
	}
	if s.Tools > 0 || s.Prompts > 0 {
		frames = append(frames, fmt.Sprintf("Session · %d prompts · %d tool calls", s.Prompts, s.Tools))
	}
	if s.CurrentTool != "" && (state == "running" || state == "editing" || state == "waiting") {
		frames = append(frames, "Tool · "+s.CurrentTool)
	}
	if s.LifetimeSessions > 0 {
		frames = append(frames, fmt.Sprintf("Lifetime · %s tokens · %d sessions", compact(s.LifetimeTokens), s.LifetimeSessions))
		frames = append(frames, fmt.Sprintf("Lifetime · %d prompts · %d tool calls", s.LifetimePrompts, s.LifetimeTools))
	}
	frame := int(now.Unix()/20) % len(frames)
	tooltip := "Codex · " + label
	if s.Model != "" {
		tooltip += " · " + s.Model
	}
	if s.LifetimeSessions > 0 {
		tooltip += " · " + compact(s.LifetimeTokens) + " lifetime tokens"
	}
	activity := map[string]any{"details": presenceText(details), "state": presenceText(frames[frame]), "assets": map[string]string{"large_image": image, "large_text": presenceText(tooltip)}}
	if !s.Started.IsZero() && !s.Started.After(now) {
		activity["timestamps"] = map[string]int64{"start": s.Started.Unix()}
	}
	githubURL := "https://github.com/ge0rg3e/codex-rpc"
	if login != "" {
		githubURL = "https://github.com/" + login
	}
	activity["buttons"] = []any{map[string]string{"label": "View on GitHub", "url": githubURL}}
	return activity
}

// Discord caps text fields at 128 bytes. Keep UTF-8 intact and strip line breaks.
func presenceText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= 128 {
		return value
	}
	result := ""
	for _, char := range value {
		if len(result)+len(string(char)) > 125 {
			break
		}
		result += string(char)
	}
	return result + "..."
}
func (d *Discord) command(activity any) error {
	d.sequence++
	nonce := fmt.Sprint(d.sequence)
	if err := d.write(1, map[string]any{"cmd": "SET_ACTIVITY", "args": map[string]any{"pid": os.Getpid(), "activity": activity}, "nonce": nonce}); err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		reply, err := d.read()
		if err != nil {
			return err
		}
		if reply["evt"] == "ERROR" {
			return fmt.Errorf("Discord rejected the presence; check your application ID")
		}
		if reply["nonce"] == nonce {
			return nil
		}
	}
	return fmt.Errorf("Discord did not acknowledge the presence")
}
func compact(n int64) string {
	if n >= 1e9 {
		return fmt.Sprintf("%.1fB", float64(n)/1e9)
	}
	if n >= 1e6 {
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	}
	if n >= 1e3 {
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

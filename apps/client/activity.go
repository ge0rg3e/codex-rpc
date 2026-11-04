package main

import (
	"strings"
	"time"
)

func toolActivity(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "request_user_input"), strings.Contains(name, "approval"):
		return "waiting"
	case strings.Contains(name, "apply_patch"), strings.Contains(name, "write_file"), strings.Contains(name, "edit_file"):
		return "editing"
	default:
		return "running"
	}
}

func activityLabel(state string) string {
	switch state {
	case "thinking":
		return "Thinking"
	case "coding":
		return "Writing a response"
	case "editing":
		return "Editing files"
	case "running":
		return "Using tools"
	case "waiting":
		return "Waiting for your input"
	case "success":
		return "Task complete"
	case "error":
		return "Encountered an error"
	case "stopped":
		return "Session paused"
	default:
		return "Idle"
	}
}
func activityAt(s Session, now time.Time) (string, bool) {
	age := now.Sub(s.Last)
	if s.Last.IsZero() || age < -time.Minute || age > 10*time.Minute {
		return "idle", false
	}
	state := s.Activity
	if state == "" {
		state = "thinking"
	}
	switch state {
	case "success", "error", "stopped":
		if age > time.Minute {
			return "idle", true
		}
	}
	return state, true
}

func liveCodexPresence(sessions []Session, now, cli, desktop, backend time.Time, windowOpen bool) (Session, string, bool) {
	var latest Session
	activity := "idle"
	for _, session := range sessions {
		if session.Internal || session.Archived || !session.Last.After(latest.Last) {
			continue
		}
		state, visible := activityAt(session, now)
		if session.Source == "Codex desktop" {
			running := !backend.IsZero() && !session.TurnStarted.IsZero() && !session.Last.Before(backend)
			if running {
				state, visible = session.Activity, true
			} else {
				visible = visible && windowOpen && !session.Last.Before(desktop)
			}
		} else {
			visible = visible && !cli.IsZero() && !session.Last.Before(cli)
		}
		if visible {
			latest, activity = session, state
		}
	}
	return latest, activity, latest.ID != ""
}

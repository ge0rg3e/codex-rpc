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

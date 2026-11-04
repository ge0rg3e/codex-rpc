package main

import (
	"testing"
	"time"
)

func TestLiveCodexPresence(t *testing.T) {
	now := time.Now()
	start := now.Add(-time.Hour)
	cli := Session{ID: "cli", Source: "Codex CLI", Last: now, Activity: "thinking"}
	finished := Session{ID: "finished", Source: "Codex desktop", Last: now, Activity: "success"}
	running := Session{
		ID: "running", Source: "Codex desktop", Last: now.Add(-20 * time.Minute),
		Activity: "running", TurnStarted: now.Add(-30 * time.Minute),
	}
	sessions := []Session{cli, finished, running}
	for _, test := range []struct {
		name                  string
		cli, desktop, backend time.Time
		windowOpen            bool
		want                  string
	}{
		{name: "CLI open", cli: start, want: "cli"},
		{name: "CLI closed"},
		{name: "desktop open", desktop: start, windowOpen: true, want: "finished"},
		{name: "desktop hidden with task", desktop: start, backend: start, want: "running"},
		{name: "desktop quit with task", backend: start, want: "running"},
		{name: "desktop quit and backend stopped"},
		{name: "backend restarted", backend: now.Add(-time.Minute)},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected, _, visible := liveCodexPresence(sessions, now, test.cli, test.desktop, test.backend, test.windowOpen)
			if selected.ID != test.want || visible != (test.want != "") {
				t.Fatalf("selected %q, visible %v, want %q", selected.ID, visible, test.want)
			}
		})
	}
	running.TurnStarted = time.Time{}
	if _, _, visible := liveCodexPresence([]Session{finished, running}, now, time.Time{}, start, start, false); visible {
		t.Fatal("a hidden desktop kept presence after the task completed")
	}
}

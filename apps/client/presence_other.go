//go:build !windows

package main

import "time"

func codexPresence(sessions []Session, now time.Time) (Session, string, bool) {
	var latest Session
	for _, session := range sessions {
		if !session.Internal && session.Last.After(latest.Last) {
			latest = session
		}
	}
	activity, visible := activityAt(latest, now)
	return latest, activity, visible
}

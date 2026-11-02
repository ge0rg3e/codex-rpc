package main

import (
	"sort"
	"time"
)

// Minute buckets make overlaps between parallel sessions and retried uploads
// idempotent. No prompts, paths, or model information are sent.
func (s *PresenceService) syncActivity(sessions []Session, token string) error {
	if s.activityToken != token {
		s.activityToken = token
		s.activitySynced = map[int64]bool{}
	}
	pending := map[int64]bool{}
	now := time.Now().Unix() / 60
	for _, session := range sessions {
		if session.Internal {
			continue
		}
		for _, minute := range session.ActivityMinutes {
			if minute >= 26297280 && minute <= now && !s.activitySynced[minute] {
				pending[minute] = true
			}
		}
	}
	minutes := make([]int64, 0, len(pending))
	for minute := range pending {
		minutes = append(minutes, minute)
	}
	sort.Slice(minutes, func(i, j int) bool { return minutes[i] < minutes[j] })
	for start := 0; start < len(minutes); start += 5000 {
		batch := minutes[start:min(start+5000, len(minutes))]
		if err := s.request("POST", "activity", map[string]any{"minutes": batch}, nil); err != nil {
			return err
		}
		for _, minute := range batch {
			s.activitySynced[minute] = true
		}
	}
	return nil
}

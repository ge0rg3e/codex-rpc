package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const codexResetStatusURL = "https://codex-resets.com/api/v1/status"

type ResetSource struct {
	Type   string `json:"type"`
	Author string `json:"author,omitempty"`
	URL    string `json:"url,omitempty"`
}

type ResetAnnouncement struct {
	ID          string      `json:"id"`
	ResetType   string      `json:"reset_type"`
	AnnouncedAt time.Time   `json:"announced_at"`
	Text        string      `json:"text"`
	Source      ResetSource `json:"source"`
}

type ScheduledReset struct {
	ID           string      `json:"id"`
	Status       string      `json:"status"`
	ResetType    string      `json:"reset_type"`
	AnnouncedAt  time.Time   `json:"announced_at"`
	ScheduledFor *time.Time  `json:"scheduled_for"`
	Text         string      `json:"text"`
	Source       ResetSource `json:"source"`
}

type ResetWatch struct {
	Level              string      `json:"level"`
	ResetChancePercent *int        `json:"reset_chance_percent"`
	ForecastWindow     string      `json:"forecast_window"`
	ObservedAt         time.Time   `json:"observed_at"`
	ExpiresAt          time.Time   `json:"expires_at"`
	Text               string      `json:"text"`
	Source             ResetSource `json:"source"`
}

type ResetStats struct {
	Total           int        `json:"total"`
	LastResetAt     *time.Time `json:"last_reset_at"`
	DaysSinceLast   *float64   `json:"days_since_last"`
	AvgIntervalDays *float64   `json:"avg_interval_days"`
}

type CodexResetStatus struct {
	LatestReset    *ResetAnnouncement `json:"latest_reset"`
	ScheduledReset *ScheduledReset    `json:"scheduled_reset"`
	ActiveWatch    *ResetWatch        `json:"active_watch"`
	Stats          ResetStats         `json:"stats"`
}

func fetchCodexResetStatus(client *http.Client, endpoint string) (*CodexResetStatus, error) {
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request reset status: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reset API returned status %d", response.StatusCode)
	}
	var payload struct {
		Data CodexResetStatus `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode reset status: %w", err)
	}
	return &payload.Data, nil
}

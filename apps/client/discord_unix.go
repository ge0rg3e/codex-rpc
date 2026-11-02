//go:build !windows

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

func dialDiscord() (net.Conn, error) {
	for _, root := range []string{os.Getenv("XDG_RUNTIME_DIR"), os.Getenv("TMPDIR"), "/tmp"} {
		if root == "" {
			continue
		}
		for i := 0; i < 10; i++ {
			conn, err := net.DialTimeout("unix", filepath.Join(root, fmt.Sprintf("discord-ipc-%d", i)), 100*time.Millisecond)
			if err == nil {
				return conn, nil
			}
		}
	}
	return nil, fmt.Errorf("Discord is not running")
}

//go:build windows

package main

import (
	"fmt"
	"gopkg.in/natefinch/npipe.v2"
	"net"
	"time"
)

func dialDiscord() (net.Conn, error) {
	for i := 0; i < 10; i++ {
		conn, err := npipe.DialTimeout(fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i), 100*time.Millisecond)
		if err == nil {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("Discord is not running")
}

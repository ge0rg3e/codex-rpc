//go:build !linux || android || server

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func configureTrayPosition(*application.App, *application.SystemTray, *application.WebviewWindow) {}

package main

import (
	"slices"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var windowTitleLength = windows.NewLazySystemDLL("user32.dll").NewProc("GetWindowTextLengthW")

type codexProcess struct {
	name   string
	parent uint32
}

func codexPresence(sessions []Session, now time.Time) (Session, string, bool) {
	processes := codexProcesses()
	var cli, desktop, backend time.Time
	for pid, process := range processes {
		switch process.name {
		case "chatgpt.exe":
			desktop = earlierStart(desktop, pid, nil)
		case "codex.exe", "codex-x86_64-pc-windows-msvc.exe", "codex-aarch64-pc-windows-msvc.exe":
			if hasDesktopParent(processes, process.parent) {
				backend = earlierStart(backend, pid, nil)
			} else {
				var appServer bool
				start := earlierStart(time.Time{}, pid, &appServer)
				if appServer {
					backend = firstStart(backend, start)
				} else {
					cli = firstStart(cli, start)
				}
			}
		}
	}
	return liveCodexPresence(sessions, now, cli, desktop, backend, !desktop.IsZero() && desktopWindowOpen(processes))
}

func codexProcesses() map[uint32]codexProcess {
	processes := make(map[uint32]codexProcess)
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return processes
	}
	defer windows.CloseHandle(snapshot)
	var currentSession uint32
	if windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &currentSession) != nil {
		return processes
	}
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		var session uint32
		if windows.ProcessIdToSessionId(entry.ProcessID, &session) == nil && session == currentSession {
			processes[entry.ProcessID] = codexProcess{
				name:   strings.ToLower(windows.UTF16ToString(entry.ExeFile[:])),
				parent: entry.ParentProcessID,
			}
		}
	}
	return processes
}

func hasDesktopParent(processes map[uint32]codexProcess, pid uint32) bool {
	// A snapshot can contain a reused parent PID.
	for range len(processes) {
		process, ok := processes[pid]
		if !ok {
			break
		}
		if process.name == "chatgpt.exe" {
			return true
		}
		pid = process.parent
	}
	return false
}

func earlierStart(current time.Time, pid uint32, appServer *bool) time.Time {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return current
	}
	defer windows.CloseHandle(handle)
	var created, exited, kernel, user windows.Filetime
	if windows.GetProcessTimes(handle, &created, &exited, &kernel, &user) != nil || exited != (windows.Filetime{}) {
		return current
	}
	if appServer != nil {
		*appServer = isAppServer(handle)
	}
	return firstStart(current, time.Unix(0, created.Nanoseconds()))
}

func isAppServer(handle windows.Handle) bool {
	var size uint32
	_ = windows.NtQueryInformationProcess(handle, windows.ProcessCommandLineInformation, nil, 0, &size)
	if size < uint32(unsafe.Sizeof(windows.NTUnicodeString{})) || size > 128*1024 {
		return false
	}
	buffer := make([]byte, size)
	if windows.NtQueryInformationProcess(handle, windows.ProcessCommandLineInformation, unsafe.Pointer(&buffer[0]), size, nil) != nil {
		return false
	}
	commandLine := (*windows.NTUnicodeString)(unsafe.Pointer(&buffer[0]))
	args, err := windows.DecomposeCommandLine(windows.UTF16ToString(unsafe.Slice(commandLine.Buffer, int(commandLine.Length)/2)))
	return err == nil && len(args) > 1 && slices.Contains(args[1:], "app-server")
}

func firstStart(current, candidate time.Time) time.Time {
	if !candidate.IsZero() && (current.IsZero() || candidate.Before(current)) {
		return candidate
	}
	return current
}

type desktopWindowScan struct {
	processes map[uint32]codexProcess
	open      bool
}

var scanDesktopWindow = windows.NewCallback(func(hwnd windows.HWND, param unsafe.Pointer) uintptr {
	scan := (*desktopWindowScan)(param)
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err != nil || scan.processes[pid].name != "chatgpt.exe" || !windows.IsWindowVisible(hwnd) {
		return 1
	}
	titleLength, _, _ := windowTitleLength.Call(uintptr(hwnd))
	var cloaked uint32
	const dwmwaCloaked = 14
	_ = windows.DwmGetWindowAttribute(hwnd, dwmwaCloaked, unsafe.Pointer(&cloaked), uint32(unsafe.Sizeof(cloaked)))
	if titleLength > 0 && cloaked == 0 {
		scan.open = true
		return 0
	}
	return 1
})

func desktopWindowOpen(processes map[uint32]codexProcess) bool {
	scan := desktopWindowScan{processes: processes}
	_ = windows.EnumWindows(scanDesktopWindow, unsafe.Pointer(&scan))
	return scan.open
}

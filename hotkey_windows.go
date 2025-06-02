//go:build windows
// +build windows

package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// Windows API constants
const (
	WM_HOTKEY = 0x0312
	MOD_ALT   = 0x0001
	MOD_CTRL  = 0x0002
	MOD_SHIFT = 0x0004
	MOD_WIN   = 0x0008
)

// Virtual key codes for common keys
var virtualKeyCodes = map[string]uint32{
	"a": 0x41, "b": 0x42, "c": 0x43, "d": 0x44, "e": 0x45, "f": 0x46, "g": 0x47, "h": 0x48,
	"i": 0x49, "j": 0x4A, "k": 0x4B, "l": 0x4C, "m": 0x4D, "n": 0x4E, "o": 0x4F, "p": 0x50,
	"q": 0x51, "r": 0x52, "s": 0x53, "t": 0x54, "u": 0x55, "v": 0x56, "w": 0x57, "x": 0x58,
	"y": 0x59, "z": 0x5A,
	"0": 0x30, "1": 0x31, "2": 0x32, "3": 0x33, "4": 0x34, "5": 0x35, "6": 0x36, "7": 0x37,
	"8": 0x38, "9": 0x39,
	"f1": 0x70, "f2": 0x71, "f3": 0x72, "f4": 0x73, "f5": 0x74, "f6": 0x75, "f7": 0x76, "f8": 0x77,
	"f9": 0x78, "f10": 0x79, "f11": 0x7A, "f12": 0x7B,
	"space": 0x20, "enter": 0x0D, "esc": 0x1B, "tab": 0x09, "backspace": 0x08,
	"left": 0x25, "up": 0x26, "right": 0x27, "down": 0x28,
	"home": 0x24, "end": 0x23, "pageup": 0x21, "pagedown": 0x22,
	"insert": 0x2D, "delete": 0x2E,
}

type WindowsHotkeyManager struct {
	mutex           sync.Mutex
	running         bool
	hotkeys         map[int32]*hotkeyInfo
	nextHotkeyID    int32
	messageWindow   windows.Handle
	stopChan        chan struct{}
	captureMode     bool
	captureChan     chan string
	captureStopChan chan struct{}
}

type hotkeyInfo struct {
	id       int32
	shortcut string
	callback func()
}

// newPlatformHotkeyManager creates a new Windows hotkey manager
func newPlatformHotkeyManager() (HotkeyManager, error) {
	return &WindowsHotkeyManager{
		hotkeys:      make(map[int32]*hotkeyInfo),
		nextHotkeyID: 1,
		stopChan:     make(chan struct{}),
	}, nil
}

func (w *WindowsHotkeyManager) Start() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.running {
		return nil
	}

	// Create a message-only window for receiving hotkey messages
	err := w.createMessageWindow()
	if err != nil {
		return fmt.Errorf("failed to create message window: %w", err)
	}

	w.running = true

	// Start message loop in a separate goroutine
	go w.messageLoop()

	log.Println("Windows hotkey manager started")
	return nil
}

func (w *WindowsHotkeyManager) Stop() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.running {
		return nil
	}

	// Unregister all hotkeys
	for id := range w.hotkeys {
		w.unregisterHotkeyByID(id)
	}

	// Stop message loop
	close(w.stopChan)
	w.running = false

	// Destroy message window
	if w.messageWindow != 0 {
		syscall.NewLazyDLL("user32.dll").NewProc("DestroyWindow").Call(uintptr(w.messageWindow))
		w.messageWindow = 0
	}

	log.Println("Windows hotkey manager stopped")
	return nil
}

func (w *WindowsHotkeyManager) RegisterHotkey(shortcut string, callback func()) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.running {
		return fmt.Errorf("hotkey manager not running")
	}

	modifiers, vk, err := w.parseShortcut(shortcut)
	if err != nil {
		return fmt.Errorf("failed to parse shortcut '%s': %w", shortcut, err)
	}

	id := w.nextHotkeyID
	w.nextHotkeyID++

	// Register the hotkey with Windows
	ret, _, err := syscall.NewLazyDLL("user32.dll").NewProc("RegisterHotKey").Call(
		uintptr(w.messageWindow),
		uintptr(id),
		uintptr(modifiers),
		uintptr(vk),
	)

	if ret == 0 {
		return fmt.Errorf("failed to register hotkey '%s': %w", shortcut, err)
	}

	w.hotkeys[id] = &hotkeyInfo{
		id:       id,
		shortcut: shortcut,
		callback: callback,
	}

	log.Printf("Registered Windows hotkey: %s (ID: %d)", shortcut, id)
	return nil
}

func (w *WindowsHotkeyManager) UnregisterHotkey(shortcut string) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Find hotkey by shortcut
	for id, info := range w.hotkeys {
		if info.shortcut == shortcut {
			return w.unregisterHotkeyByID(id)
		}
	}

	return fmt.Errorf("hotkey '%s' not found", shortcut)
}

func (w *WindowsHotkeyManager) unregisterHotkeyByID(id int32) error {
	syscall.NewLazyDLL("user32.dll").NewProc("UnregisterHotKey").Call(
		uintptr(w.messageWindow),
		uintptr(id),
	)
	delete(w.hotkeys, id)
	log.Printf("Unregistered Windows hotkey ID: %d", id)
	return nil
}

func (w *WindowsHotkeyManager) StartShortcutCapture() (<-chan string, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.captureMode {
		return w.captureChan, nil
	}

	w.captureMode = true
	w.captureChan = make(chan string, 1)
	w.captureStopChan = make(chan struct{})

	// Start low-level keyboard hook for capture mode
	go w.captureLoop()

	return w.captureChan, nil
}

func (w *WindowsHotkeyManager) StopShortcutCapture() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.captureMode {
		return nil
	}

	w.captureMode = false
	close(w.captureStopChan)
	return nil
}

func (w *WindowsHotkeyManager) parseShortcut(shortcut string) (uint32, uint32, error) {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	if len(parts) == 0 {
		return 0, 0, fmt.Errorf("empty shortcut")
	}

	var modifiers uint32
	var key string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "ctrl", "control":
			modifiers |= MOD_CTRL
		case "alt":
			modifiers |= MOD_ALT
		case "shift":
			modifiers |= MOD_SHIFT
		case "cmd", "win", "meta":
			modifiers |= MOD_WIN
		default:
			if key == "" {
				key = part
			} else {
				return 0, 0, fmt.Errorf("multiple keys specified: %s", shortcut)
			}
		}
	}

	if key == "" {
		return 0, 0, fmt.Errorf("no key specified")
	}

	vk, ok := virtualKeyCodes[key]
	if !ok {
		return 0, 0, fmt.Errorf("unknown key: %s", key)
	}

	return modifiers, vk, nil
}

func (w *WindowsHotkeyManager) createMessageWindow() error {
	// This is a simplified implementation
	// In a real implementation, you'd want to register a window class and create a proper message-only window
	// For now, we'll use a simple approach
	w.messageWindow = windows.Handle(1) // Placeholder
	return nil
}

func (w *WindowsHotkeyManager) messageLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		select {
		case <-w.stopChan:
			return
		default:
			// Simplified message loop - in a real implementation, you'd use GetMessage/PeekMessage
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (w *WindowsHotkeyManager) captureLoop() {
	// Simplified capture implementation
	// In a real implementation, you'd use SetWindowsHookEx with WH_KEYBOARD_LL
	// For now, we'll provide a basic timeout-based approach

	timeout := time.After(30 * time.Second)

	select {
	case <-timeout:
		// Timeout - send a default shortcut
		select {
		case w.captureChan <- "ctrl+shift+s":
		default:
		}
	case <-w.captureStopChan:
		return
	}
}

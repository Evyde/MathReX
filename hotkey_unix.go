//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"
)

type UnixHotkeyManager struct {
	mutex             sync.Mutex
	running           bool
	registeredHotkeys map[string]func()
	captureMode       bool
	captureChan       chan string
	captureStopChan   chan struct{}
}

// newPlatformHotkeyManager creates a new Unix/Linux/macOS hotkey manager
func newPlatformHotkeyManager() (HotkeyManager, error) {
	return &UnixHotkeyManager{
		registeredHotkeys: make(map[string]func()),
	}, nil
}

func (u *UnixHotkeyManager) Start() error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if u.running {
		return nil
	}

	log.Println("Starting Unix hotkey system (simplified mode)...")

	// Note: This is a simplified implementation without global hotkey support
	// Global hotkeys require platform-specific implementations that are complex
	// For now, we'll just mark as running and log a warning
	log.Println("Warning: Global hotkeys are not supported in this simplified implementation")
	log.Println("Hotkey functionality will be limited to application focus")

	u.running = true
	log.Println("Unix hotkey system started (simplified mode)")
	return nil
}

func (u *UnixHotkeyManager) Stop() error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if !u.running {
		return nil
	}

	log.Println("Stopping Unix hotkey system...")

	// Stop capture mode if active
	if u.captureMode {
		u.captureMode = false
		if u.captureStopChan != nil {
			close(u.captureStopChan)
		}
	}

	u.running = false
	log.Println("Unix hotkey system stopped")
	return nil
}

func (u *UnixHotkeyManager) RegisterHotkey(shortcut string, callback func()) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if !u.running {
		return fmt.Errorf("hotkey manager not running")
	}

	// In simplified mode, we just store the hotkey but don't actually register it globally
	log.Printf("Registering Unix hotkey (simplified mode): %s", shortcut)
	log.Printf("Warning: Global hotkey '%s' will not be active (simplified implementation)", shortcut)

	u.registeredHotkeys[shortcut] = callback
	log.Printf("Stored Unix hotkey: %s", shortcut)
	return nil
}

func (u *UnixHotkeyManager) UnregisterHotkey(shortcut string) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if _, exists := u.registeredHotkeys[shortcut]; !exists {
		return fmt.Errorf("hotkey '%s' not found", shortcut)
	}

	delete(u.registeredHotkeys, shortcut)
	// Note: gohook doesn't provide a direct way to unregister specific hotkeys
	// In a real implementation, you might need to restart the hook system
	log.Printf("Unregistered Unix hotkey: %s", shortcut)
	return nil
}

func (u *UnixHotkeyManager) StartShortcutCapture() (<-chan string, error) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if u.captureMode {
		return u.captureChan, nil
	}

	u.captureMode = true
	u.captureChan = make(chan string, 1)
	u.captureStopChan = make(chan struct{})

	// In simplified mode, we'll simulate capture by providing a default shortcut after a delay
	go func() {
		time.Sleep(2 * time.Second)
		select {
		case u.captureChan <- "ctrl+shift+s":
			log.Println("Simulated shortcut capture: ctrl+shift+s")
		case <-u.captureStopChan:
			return
		default:
		}
	}()

	log.Println("Started Unix shortcut capture mode (simplified - will auto-suggest ctrl+shift+s)")
	return u.captureChan, nil
}

func (u *UnixHotkeyManager) StopShortcutCapture() error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if !u.captureMode {
		return nil
	}

	u.captureMode = false
	if u.captureStopChan != nil {
		close(u.captureStopChan)
	}

	log.Println("Stopped Unix shortcut capture mode")
	return nil
}

// parseShortcutString converts a shortcut string like "cmd+shift+c" into gohook format
func (u *UnixHotkeyManager) parseShortcutString(shortcut string) []string {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	if len(parts) == 0 {
		return nil
	}

	var key string
	var modifiers []string

	for _, part := range parts {
		p := strings.ToLower(strings.TrimSpace(part))
		switch p {
		case "ctrl", "control":
			modifiers = append(modifiers, "ctrl")
		case "alt", "option":
			modifiers = append(modifiers, "alt")
		case "shift":
			modifiers = append(modifiers, "shift")
		case "cmd", "command", "super", "win":
			if runtime.GOOS == "darwin" {
				modifiers = append(modifiers, "cmd")
			} else {
				modifiers = append(modifiers, p)
			}
		default:
			if key == "" {
				key = p
			} else {
				log.Printf("Warning: Multiple non-modifier keys found in shortcut string: '%s'. Using '%s'.", shortcut, key)
			}
		}
	}

	if key == "" {
		log.Printf("Warning: No main key found in shortcut string: '%s'", shortcut)
		return nil
	}

	return append([]string{key}, modifiers...)
}

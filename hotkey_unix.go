//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	gohook "github.com/robotn/gohook"
)

type UnixHotkeyManager struct {
	mutex             sync.Mutex
	running           bool
	hookEventChannel  chan gohook.Event
	hookProcessDone   chan struct{}
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

	log.Println("Starting Unix hotkey system...")
	u.hookEventChannel = gohook.Start()
	u.hookProcessDone = make(chan struct{})

	// Start the gohook process goroutine
	go func(currentHookEventChannel chan gohook.Event, currentHookProcessDone chan struct{}) {
		log.Println("gohook.Process starting...")
		<-gohook.Process(currentHookEventChannel)
		close(currentHookProcessDone)
		log.Println("gohook.Process finished.")
	}(u.hookEventChannel, u.hookProcessDone)

	// Register generic event listener for shortcut capture
	gohook.Register(gohook.KeyDown, []string{}, func(e gohook.Event) {
		u.mutex.Lock()
		if u.captureMode && u.captureChan != nil {
			shortcutStr := u.eventToShortcutString(e)
			if shortcutStr != "" {
				select {
				case u.captureChan <- shortcutStr:
					log.Printf("Captured shortcut: %s", shortcutStr)
				default:
				}
			}
		}
		u.mutex.Unlock()
	})

	u.running = true
	log.Println("Unix hotkey system started")
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

	// Stop gohook
	if u.hookEventChannel != nil {
		gohook.End()

		// Wait for the gohook.Process goroutine to finish
		if u.hookProcessDone != nil {
			select {
			case <-u.hookProcessDone:
				log.Println("gohook.Process finished.")
			case <-time.After(2 * time.Second):
				log.Println("Timeout waiting for gohook.Process to finish.")
			}
		}
		u.hookEventChannel = nil
		u.hookProcessDone = nil
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

	shortcutParts := u.parseShortcutString(shortcut)
	if shortcutParts == nil || len(shortcutParts) < 1 {
		return fmt.Errorf("failed to parse shortcut string: '%s'", shortcut)
	}

	log.Printf("Registering Unix hotkey: %s -> %v", shortcut, shortcutParts)

	gohook.Register(gohook.KeyDown, shortcutParts, func(e gohook.Event) {
		u.mutex.Lock()
		if !u.captureMode {
			u.mutex.Unlock()
			log.Printf("Unix hotkey triggered: %s", shortcut)
			go callback()
		} else {
			u.mutex.Unlock()
			log.Printf("Unix hotkey ignored (capture mode): %s", shortcut)
		}
	})

	u.registeredHotkeys[shortcut] = callback
	log.Printf("Registered Unix hotkey: %s", shortcut)
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

	log.Println("Started Unix shortcut capture mode")
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

// eventToShortcutString converts a gohook.Event to a shortcut string
func (u *UnixHotkeyManager) eventToShortcutString(ev gohook.Event) string {
	// Hardcoded raw code to string map for common keys
	rawCodeToNamedKey := map[uint16]string{
		27: "esc", 32: "space", 13: "enter", 9: "tab", 8: "backspace",
		46: "delete", 36: "home", 35: "end", 33: "pageup", 34: "pagedown",
		37: "left", 38: "up", 39: "right", 40: "down",
		112: "f1", 113: "f2", 114: "f3", 115: "f4", 116: "f5", 117: "f6",
		118: "f7", 119: "f8", 120: "f9", 121: "f10", 122: "f11", 123: "f12",
	}

	// Modifier constants
	const (
		ModShift uint16 = 1
		ModCtrl  uint16 = 2
		ModAlt   uint16 = 4
		ModCmd   uint16 = 8
	)

	var parts []string
	keyStr := ""

	// Try to get a friendly name for the key
	if name, ok := rawCodeToNamedKey[ev.Rawcode]; ok {
		keyStr = name
	} else if ev.Keychar != 0 && ev.Keychar != 65535 {
		keyStr = strings.ToLower(string(ev.Keychar))
	} else {
		keyStr = fmt.Sprintf("raw%d", ev.Rawcode)
	}

	if keyStr == "" {
		log.Printf("Warning: Could not determine key string for event: %+v", ev)
		return ""
	}

	var modifiers []string

	// Handle modifiers based on platform
	if runtime.GOOS == "darwin" {
		if ev.Mask&ModAlt > 0 {
			modifiers = append(modifiers, "cmd")
		}
		if ev.Mask&ModCmd > 0 {
			modifiers = append(modifiers, "alt")
		}
	} else {
		if ev.Mask&ModAlt > 0 {
			modifiers = append(modifiers, "alt")
		}
		if ev.Mask&ModCmd > 0 {
			modifiers = append(modifiers, "meta")
		}
	}

	if ev.Mask&ModCtrl > 0 {
		modifiers = append(modifiers, "ctrl")
	}
	if ev.Mask&ModShift > 0 {
		modifiers = append(modifiers, "shift")
	}

	// Sort modifiers for consistency
	modOrder := map[string]int{"ctrl": 1, "alt": 2, "shift": 3, "cmd": 4, "meta": 5}
	sort.SliceStable(modifiers, func(i, j int) bool {
		return modOrder[modifiers[i]] < modOrder[modifiers[j]]
	})

	parts = append(parts, keyStr)
	parts = append(parts, modifiers...)
	return strings.Join(parts, "+")
}

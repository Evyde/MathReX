package main

import (
	"log"
	"sync"
)

// HotkeyManager provides a platform-agnostic interface for hotkey management
type HotkeyManager interface {
	// Start initializes the hotkey system
	Start() error
	
	// Stop shuts down the hotkey system
	Stop() error
	
	// RegisterHotkey registers a global hotkey with the given shortcut string
	RegisterHotkey(shortcut string, callback func()) error
	
	// UnregisterHotkey removes a previously registered hotkey
	UnregisterHotkey(shortcut string) error
	
	// StartShortcutCapture starts capturing key events for shortcut setting
	StartShortcutCapture() (<-chan string, error)
	
	// StopShortcutCapture stops capturing key events for shortcut setting
	StopShortcutCapture() error
}

// Global hotkey manager instance
var globalHotkeyManager HotkeyManager
var hotkeyManagerMutex sync.Mutex

// InitializeHotkeyManager creates and initializes the platform-specific hotkey manager
func InitializeHotkeyManager() error {
	hotkeyManagerMutex.Lock()
	defer hotkeyManagerMutex.Unlock()
	
	if globalHotkeyManager != nil {
		return nil // Already initialized
	}
	
	var err error
	globalHotkeyManager, err = newPlatformHotkeyManager()
	if err != nil {
		return err
	}
	
	return globalHotkeyManager.Start()
}

// GetHotkeyManager returns the global hotkey manager instance
func GetHotkeyManager() HotkeyManager {
	hotkeyManagerMutex.Lock()
	defer hotkeyManagerMutex.Unlock()
	return globalHotkeyManager
}

// ShutdownHotkeyManager stops and cleans up the hotkey manager
func ShutdownHotkeyManager() error {
	hotkeyManagerMutex.Lock()
	defer hotkeyManagerMutex.Unlock()
	
	if globalHotkeyManager == nil {
		return nil
	}
	
	err := globalHotkeyManager.Stop()
	globalHotkeyManager = nil
	return err
}

// RegisterGlobalHotkey is a convenience function to register a hotkey
func RegisterGlobalHotkey(shortcut string, callback func()) error {
	manager := GetHotkeyManager()
	if manager == nil {
		log.Println("Warning: Hotkey manager not initialized")
		return nil
	}
	return manager.RegisterHotkey(shortcut, callback)
}

// UnregisterGlobalHotkey is a convenience function to unregister a hotkey
func UnregisterGlobalHotkey(shortcut string) error {
	manager := GetHotkeyManager()
	if manager == nil {
		return nil
	}
	return manager.UnregisterHotkey(shortcut)
}

// StartGlobalShortcutCapture starts capturing key events for shortcut setting
func StartGlobalShortcutCapture() (<-chan string, error) {
	manager := GetHotkeyManager()
	if manager == nil {
		log.Println("Warning: Hotkey manager not initialized")
		return nil, nil
	}
	return manager.StartShortcutCapture()
}

// StopGlobalShortcutCapture stops capturing key events for shortcut setting
func StopGlobalShortcutCapture() error {
	manager := GetHotkeyManager()
	if manager == nil {
		return nil
	}
	return manager.StopShortcutCapture()
}

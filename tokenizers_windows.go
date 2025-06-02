//go:build windows
// +build windows

package main

import (
	"fmt"
	"log"
)

// WindowsTokenizerWrapper provides a Windows-compatible tokenizer interface
type WindowsTokenizerWrapper struct {
	tokenizer *FixedTokenizer
}

// NewWindowsTokenizer creates a new Windows-compatible tokenizer
func NewWindowsTokenizer(configPath string) (*WindowsTokenizerWrapper, error) {
	log.Printf("Initializing Windows tokenizer from: %s", configPath)

	// Try to use the fixed tokenizers implementation
	tok, err := NewFixedTokenizerFromFile(configPath)
	if err != nil {
		log.Printf("Failed to create fixed tokenizer: %v", err)
		log.Println("Note: Make sure libtokenizers.a is available for Windows")
		return nil, fmt.Errorf("failed to create Windows tokenizer: %w", err)
	}

	return &WindowsTokenizerWrapper{
		tokenizer: tok,
	}, nil
}

// Encode tokenizes the input text using the fixed tokenizers implementation
func (w *WindowsTokenizerWrapper) Encode(text string, addSpecialTokens bool) ([]int, error) {
	if w.tokenizer == nil {
		return nil, fmt.Errorf("tokenizer not initialized")
	}

	// Use the fixed tokenizer
	tokens, err := w.tokenizer.Encode(text, addSpecialTokens)
	if err != nil {
		return nil, fmt.Errorf("encoding failed: %w", err)
	}

	// Convert []uint32 to []int
	result := make([]int, len(tokens))
	for i, token := range tokens {
		result[i] = int(token)
	}

	return result, nil
}

// Decode converts token IDs back to text using the fixed tokenizers implementation
func (w *WindowsTokenizerWrapper) Decode(tokens []int, skipSpecialTokens bool) (string, error) {
	if w.tokenizer == nil {
		return "", fmt.Errorf("tokenizer not initialized")
	}

	// Convert []int to []uint32
	uint32Tokens := make([]uint32, len(tokens))
	for i, token := range tokens {
		uint32Tokens[i] = uint32(token)
	}

	// Use the fixed tokenizer
	result, err := w.tokenizer.Decode(uint32Tokens, skipSpecialTokens)
	if err != nil {
		return "", fmt.Errorf("decoding failed: %w", err)
	}

	return result, nil
}

// Close cleans up the tokenizer
func (w *WindowsTokenizerWrapper) Close() error {
	if w.tokenizer != nil {
		err := w.tokenizer.Close()
		w.tokenizer = nil
		return err
	}
	return nil
}

// GetVocabSize returns the vocabulary size using the fixed tokenizers implementation
func (w *WindowsTokenizerWrapper) GetVocabSize() int {
	if w.tokenizer == nil {
		return 0
	}
	return int(w.tokenizer.GetVocabSize(false))
}

// IsInitialized checks if the tokenizer is ready to use
func (w *WindowsTokenizerWrapper) IsInitialized() bool {
	return w.tokenizer != nil && w.tokenizer.IsValid()
}

// Additional helper functions for Windows compatibility

// InitWindowsTokenizer initializes the Windows tokenizer system
func InitWindowsTokenizer() error {
	log.Println("Initializing Windows tokenizer compatibility layer...")

	// In a real implementation, this might:
	// 1. Check for Windows-compatible tokenizers library
	// 2. Set up alternative tokenization methods
	// 3. Initialize any required Windows-specific resources

	log.Println("Windows tokenizer compatibility layer initialized (stub)")
	return nil
}

// CleanupWindowsTokenizer cleans up Windows tokenizer resources
func CleanupWindowsTokenizer() error {
	log.Println("Cleaning up Windows tokenizer compatibility layer...")
	return nil
}

// GetWindowsTokenizerInfo returns information about the Windows tokenizer implementation
func GetWindowsTokenizerInfo() map[string]interface{} {
	return map[string]interface{}{
		"implementation": "fixed_tokenizers",
		"platform":       "windows",
		"cgo_enabled":    true,
		"ldl_required":   false,
		"status":         "full_functionality",
		"note":           "Windows-compatible tokenizers implementation without -ldl dependency",
	}
}

// Platform-specific implementation functions

// createPlatformSpecificTokenizer creates a Windows-specific tokenizer
func createPlatformSpecificTokenizer(configPath string) (TokenizerInterface, error) {
	return NewWindowsTokenizer(configPath)
}

// getPlatformSpecificTokenizerInfo returns Windows-specific tokenizer information
func getPlatformSpecificTokenizerInfo() map[string]interface{} {
	return GetWindowsTokenizerInfo()
}

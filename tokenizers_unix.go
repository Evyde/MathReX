//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"log"

	"github.com/daulet/tokenizers"
)

// UnixTokenizerWrapper wraps the original tokenizers library for Unix/Linux/macOS
type UnixTokenizerWrapper struct {
	tokenizer *tokenizers.Tokenizer
}

// NewUnixTokenizer creates a new Unix-compatible tokenizer using the original library
func NewUnixTokenizer(configPath string) (*UnixTokenizerWrapper, error) {
	log.Printf("Initializing Unix tokenizer from: %s", configPath)

	tok, err := tokenizers.FromFile(configPath)
	if err != nil {
		return nil, err
	}

	return &UnixTokenizerWrapper{
		tokenizer: tok,
	}, nil
}

// Encode tokenizes the input text using the original tokenizers library
func (u *UnixTokenizerWrapper) Encode(text string, addSpecialTokens bool) ([]int, error) {
	if u.tokenizer == nil {
		return nil, fmt.Errorf("tokenizer not initialized")
	}

	encoding := u.tokenizer.EncodeWithOptions(text, addSpecialTokens)

	// Convert []uint32 to []int
	result := make([]int, len(encoding.IDs))
	for i, id := range encoding.IDs {
		result[i] = int(id)
	}

	return result, nil
}

// Decode converts token IDs back to text using the original tokenizers library
func (u *UnixTokenizerWrapper) Decode(tokens []int, skipSpecialTokens bool) (string, error) {
	if u.tokenizer == nil {
		return "", fmt.Errorf("tokenizer not initialized")
	}

	// Convert []int to []uint32
	uint32Tokens := make([]uint32, len(tokens))
	for i, token := range tokens {
		uint32Tokens[i] = uint32(token)
	}

	result := u.tokenizer.Decode(uint32Tokens, skipSpecialTokens)
	return result, nil
}

// Close cleans up the tokenizer
func (u *UnixTokenizerWrapper) Close() error {
	if u.tokenizer != nil {
		u.tokenizer.Close()
		u.tokenizer = nil
	}
	return nil
}

// GetVocabSize returns the vocabulary size
func (u *UnixTokenizerWrapper) GetVocabSize() int {
	if u.tokenizer == nil {
		return 0
	}
	return int(u.tokenizer.VocabSize())
}

// IsInitialized checks if the tokenizer is ready to use
func (u *UnixTokenizerWrapper) IsInitialized() bool {
	return u.tokenizer != nil
}

// Additional helper functions for Unix compatibility

// InitUnixTokenizer initializes the Unix tokenizer system
func InitUnixTokenizer() error {
	log.Println("Initializing Unix tokenizer system...")
	// The original tokenizers library handles initialization
	log.Println("Unix tokenizer system initialized")
	return nil
}

// CleanupUnixTokenizer cleans up Unix tokenizer resources
func CleanupUnixTokenizer() error {
	log.Println("Cleaning up Unix tokenizer system...")
	// The original tokenizers library handles cleanup
	return nil
}

// GetUnixTokenizerInfo returns information about the Unix tokenizer implementation
func GetUnixTokenizerInfo() map[string]interface{} {
	return map[string]interface{}{
		"implementation": "daulet/tokenizers",
		"platform":       "unix",
		"cgo_enabled":    true,
		"ldl_required":   true,
		"status":         "full_functionality",
		"note":           "Using original tokenizers library with full functionality",
	}
}

// Platform-specific implementation functions

// createPlatformSpecificTokenizer creates a Unix-specific tokenizer
func createPlatformSpecificTokenizer(configPath string) (TokenizerInterface, error) {
	return NewUnixTokenizer(configPath)
}

// getPlatformSpecificTokenizerInfo returns Unix-specific tokenizer information
func getPlatformSpecificTokenizerInfo() map[string]interface{} {
	return GetUnixTokenizerInfo()
}

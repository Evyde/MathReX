package main

import (
	"fmt"
	"log"
)

// TokenizerInterface provides a platform-agnostic interface for tokenization
type TokenizerInterface interface {
	// Encode tokenizes the input text and returns token IDs
	Encode(text string, addSpecialTokens bool) ([]int, error)

	// Decode converts token IDs back to text
	Decode(tokens []int, skipSpecialTokens bool) (string, error)

	// Close cleans up the tokenizer resources
	Close() error

	// GetVocabSize returns the vocabulary size
	GetVocabSize() int

	// IsInitialized checks if the tokenizer is ready to use
	IsInitialized() bool
}

// Global tokenizer instance
var globalTokenizer TokenizerInterface

// InitializeTokenizer creates and initializes the platform-specific tokenizer
func InitializeTokenizer(configPath string) error {
	log.Printf("Initializing platform-specific tokenizer from: %s", configPath)

	var err error
	globalTokenizer, err = newPlatformTokenizer(configPath)
	if err != nil {
		return fmt.Errorf("failed to create platform tokenizer: %w", err)
	}

	if !globalTokenizer.IsInitialized() {
		return fmt.Errorf("tokenizer failed to initialize properly")
	}

	log.Printf("Platform-specific tokenizer initialized successfully")
	log.Printf("Vocabulary size: %d", globalTokenizer.GetVocabSize())

	return nil
}

// GetTokenizer returns the global tokenizer instance
func GetTokenizer() TokenizerInterface {
	return globalTokenizer
}

// CleanupTokenizer shuts down and cleans up the tokenizer
func CleanupTokenizer() error {
	if globalTokenizer == nil {
		return nil
	}

	err := globalTokenizer.Close()
	globalTokenizer = nil

	log.Println("Tokenizer cleaned up")
	return err
}

// Convenience functions for tokenization

// EncodeText tokenizes text using the global tokenizer
func EncodeText(text string, addSpecialTokens bool) ([]int, error) {
	if globalTokenizer == nil {
		return nil, fmt.Errorf("tokenizer not initialized")
	}
	return globalTokenizer.Encode(text, addSpecialTokens)
}

// DecodeTokens converts token IDs back to text using the global tokenizer
func DecodeTokens(tokens []int, skipSpecialTokens bool) (string, error) {
	if globalTokenizer == nil {
		return "", fmt.Errorf("tokenizer not initialized")
	}
	return globalTokenizer.Decode(tokens, skipSpecialTokens)
}

// GetTokenizerVocabSize returns the vocabulary size of the global tokenizer
func GetTokenizerVocabSize() int {
	if globalTokenizer == nil {
		return 0
	}
	return globalTokenizer.GetVocabSize()
}

// IsTokenizerReady checks if the global tokenizer is ready to use
func IsTokenizerReady() bool {
	return globalTokenizer != nil && globalTokenizer.IsInitialized()
}

// GetTokenizerInfo returns information about the current tokenizer implementation
func GetTokenizerInfo() map[string]interface{} {
	info := map[string]interface{}{
		"initialized": IsTokenizerReady(),
		"vocab_size":  GetTokenizerVocabSize(),
	}

	// Add platform-specific information
	if IsTokenizerReady() {
		// This will be implemented in platform-specific files
		platformInfo := getPlatformTokenizerInfo()
		for k, v := range platformInfo {
			info[k] = v
		}
	}

	return info
}

// Platform-specific function declarations
// These will be implemented in tokenizers_windows.go and tokenizers_unix.go

// newPlatformTokenizer creates a new platform-specific tokenizer
// This function is implemented in platform-specific files using build tags
func newPlatformTokenizer(configPath string) (TokenizerInterface, error) {
	return createPlatformSpecificTokenizer(configPath)
}

// getPlatformTokenizerInfo returns platform-specific tokenizer information
// This function is implemented in platform-specific files using build tags
func getPlatformTokenizerInfo() map[string]interface{} {
	return getPlatformSpecificTokenizerInfo()
}

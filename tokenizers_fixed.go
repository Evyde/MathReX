package main

/*
Windows-compatible tokenizers implementation
This file provides a wrapper around the tokenizers library with Windows-compatible CGO flags
*/

/*
#cgo windows LDFLAGS: -ltokenizers -lm -lstdc++
#cgo !windows LDFLAGS: -ltokenizers -ldl -lm -lstdc++
#include <stdlib.h>
#include <stdbool.h>
#include <stdint.h>

// Tokenizers C interface definitions
// These should match the actual tokenizers.h interface

struct EncodeOptions {
  bool add_special_token;
  bool return_type_ids;
  bool return_tokens;
  bool return_special_tokens_mask;
  bool return_attention_mask;
  bool return_offsets;
};

struct TokenizerOptions {
  bool encode_special_tokens;
};

struct Buffer {
  uint32_t *ids;
  uint32_t *type_ids;
  uint32_t *special_tokens_mask;
  uint32_t *attention_mask;
  char **tokens;
  uint32_t len;
};

struct Offsets {
  uint32_t *start;
  uint32_t *end;
  uint32_t len;
};

struct Encoding {
  struct Buffer *buffer;
  struct Offsets *offsets;
};

// Forward declarations for tokenizers functions
typedef struct tokenizer tokenizer;

// Core tokenizer functions
tokenizer* tokenizer_from_file(const char* config);
tokenizer* tokenizer_from_bytes(const char* config, uint32_t len);
void tokenizer_free(tokenizer* tok);

// Encoding functions
struct Encoding* tokenizer_encode(tokenizer* tok, const char* text, struct EncodeOptions options);
char* tokenizer_decode(tokenizer* tok, uint32_t* ids, uint32_t len, bool skip_special_tokens);
uint32_t tokenizer_get_vocab_size(tokenizer* tok, bool with_added_tokens);

// Memory management
void encoding_free(struct Encoding* encoding);
void buffer_free(struct Buffer* buffer);
void offsets_free(struct Offsets* offsets);
void string_free(char* str);
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// FixedTokenizer provides a Windows-compatible tokenizer implementation
type FixedTokenizer struct {
	tokenizer *C.tokenizer
}

// NewFixedTokenizerFromFile creates a new tokenizer from a file
func NewFixedTokenizerFromFile(configPath string) (*FixedTokenizer, error) {
	cPath := C.CString(configPath)
	defer C.free(unsafe.Pointer(cPath))
	
	tok := C.tokenizer_from_file(cPath)
	if tok == nil {
		return nil, fmt.Errorf("failed to create tokenizer from file: %s", configPath)
	}
	
	result := &FixedTokenizer{
		tokenizer: tok,
	}
	
	// Set finalizer to ensure cleanup
	runtime.SetFinalizer(result, (*FixedTokenizer).finalize)
	
	return result, nil
}

// NewFixedTokenizerFromBytes creates a new tokenizer from byte data
func NewFixedTokenizerFromBytes(data []byte) (*FixedTokenizer, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty tokenizer data")
	}
	
	cData := C.CString(string(data))
	defer C.free(unsafe.Pointer(cData))
	
	tok := C.tokenizer_from_bytes(cData, C.uint32_t(len(data)))
	if tok == nil {
		return nil, fmt.Errorf("failed to create tokenizer from bytes")
	}
	
	result := &FixedTokenizer{
		tokenizer: tok,
	}
	
	runtime.SetFinalizer(result, (*FixedTokenizer).finalize)
	
	return result, nil
}

// Encode tokenizes the input text
func (t *FixedTokenizer) Encode(text string, addSpecialTokens bool) ([]uint32, error) {
	if t.tokenizer == nil {
		return nil, fmt.Errorf("tokenizer is nil")
	}
	
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	
	options := C.struct_EncodeOptions{
		add_special_token: C.bool(addSpecialTokens),
		return_type_ids:   C.bool(false),
		return_tokens:     C.bool(false),
		return_special_tokens_mask: C.bool(false),
		return_attention_mask: C.bool(false),
		return_offsets:    C.bool(false),
	}
	
	encoding := C.tokenizer_encode(t.tokenizer, cText, options)
	if encoding == nil {
		return nil, fmt.Errorf("encoding failed")
	}
	defer C.encoding_free(encoding)
	
	if encoding.buffer == nil {
		return nil, fmt.Errorf("encoding buffer is nil")
	}
	
	// Convert C array to Go slice
	length := int(encoding.buffer.len)
	if length == 0 {
		return []uint32{}, nil
	}
	
	// Create Go slice from C array
	ids := (*[1 << 30]C.uint32_t)(unsafe.Pointer(encoding.buffer.ids))[:length:length]
	result := make([]uint32, length)
	for i, id := range ids {
		result[i] = uint32(id)
	}
	
	return result, nil
}

// Decode converts token IDs back to text
func (t *FixedTokenizer) Decode(tokens []uint32, skipSpecialTokens bool) (string, error) {
	if t.tokenizer == nil {
		return "", fmt.Errorf("tokenizer is nil")
	}
	
	if len(tokens) == 0 {
		return "", nil
	}
	
	// Convert Go slice to C array
	cTokens := (*C.uint32_t)(C.malloc(C.size_t(len(tokens)) * C.size_t(unsafe.Sizeof(C.uint32_t(0)))))
	defer C.free(unsafe.Pointer(cTokens))
	
	// Copy tokens to C array
	cTokensSlice := (*[1 << 30]C.uint32_t)(unsafe.Pointer(cTokens))[:len(tokens):len(tokens)]
	for i, token := range tokens {
		cTokensSlice[i] = C.uint32_t(token)
	}
	
	cResult := C.tokenizer_decode(t.tokenizer, cTokens, C.uint32_t(len(tokens)), C.bool(skipSpecialTokens))
	if cResult == nil {
		return "", fmt.Errorf("decode failed")
	}
	defer C.string_free(cResult)
	
	return C.GoString(cResult), nil
}

// GetVocabSize returns the vocabulary size
func (t *FixedTokenizer) GetVocabSize(withAddedTokens bool) uint32 {
	if t.tokenizer == nil {
		return 0
	}
	
	return uint32(C.tokenizer_get_vocab_size(t.tokenizer, C.bool(withAddedTokens)))
}

// Close frees the tokenizer resources
func (t *FixedTokenizer) Close() error {
	if t.tokenizer != nil {
		C.tokenizer_free(t.tokenizer)
		t.tokenizer = nil
		runtime.SetFinalizer(t, nil)
	}
	return nil
}

// finalize is called by the garbage collector
func (t *FixedTokenizer) finalize() {
	t.Close()
}

// IsValid checks if the tokenizer is valid
func (t *FixedTokenizer) IsValid() bool {
	return t.tokenizer != nil
}

// GetInfo returns information about the tokenizer
func (t *FixedTokenizer) GetInfo() map[string]interface{} {
	return map[string]interface{}{
		"implementation": "fixed_tokenizers",
		"valid":          t.IsValid(),
		"vocab_size":     t.GetVocabSize(false),
		"ldl_required":   false,
		"platform":       runtime.GOOS,
		"note":           "Windows-compatible tokenizers implementation without -ldl dependency",
	}
}

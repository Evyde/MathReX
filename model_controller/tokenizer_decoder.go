package model_controller

import (
	"github.com/daulet/tokenizers"
)

type Tokenizer struct {
	tok *tokenizers.Tokenizer
}

func NewTokenizer(path string) (*Tokenizer, error) {
	tk, err := tokenizers.FromFile(path)
	if err != nil {
		return nil, err
	}
	return &Tokenizer{tok: tk}, nil
}

func (t *Tokenizer) Close() {
	t.tok.Close()
}

func (t *Tokenizer) Decode(tokens []uint32) string {
	return t.tok.Decode(tokens, true)
}

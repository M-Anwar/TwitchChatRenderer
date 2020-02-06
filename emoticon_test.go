package main

import (
	"testing"
)

func TestStart(t *testing.T) {

	comments, err := LoadCommentsFromCSV("data/sample_comments3.csv", 0, 300)
	if err != nil {
		t.Error(err)
	}
	cache, err := NewEmoticonCache(comments)
	if err != nil {
		t.Error(err)
	}

	cache.SaveSpriteMap("sample.png")
}

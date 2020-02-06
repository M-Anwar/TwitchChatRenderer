package main

import (
	"encoding/json"
	"os"

	"github.com/gocarina/gocsv"
)

// Message represents a twitch chat comment to render
type Message struct {
	MessageBody          string  `csv:"message_body"`
	UserColor            string  `csv:"message_user_color"`
	ContentOffsetSeconds float64 `csv:"content_offset_seconds"`
	CommenterDisplayName string  `csv:"commenter_display_name"`
	MessageFragmentsRaw  string  `csv:"message_fragments"`
	MessageFragments     []MessagePart
}

type MessagePart struct {
	Text     string   `json:"text"`
	Emoticon Emoticon `json:"emoticon"`
}
type Emoticon struct {
	EmoticonSetID string `json:"emoticon_set_id"`
	EmoticonID    string `json:"emoticon_id"`
}

// LoadCommentsFromCSV loads a csv containing chat messages and returns an
// array of messages
func LoadCommentsFromCSV(csvPath string, start float64, end float64) ([]*Message, error) {
	csvFile, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()

	chatMessages := []*Message{}

	if err := gocsv.UnmarshalFile(csvFile, &chatMessages); err != nil {
		return nil, err
	}

	filteredMessages := []*Message{}
	for _, message := range chatMessages {
		if message.ContentOffsetSeconds >= start && message.ContentOffsetSeconds <= end {
			err := json.Unmarshal([]byte(message.MessageFragmentsRaw), &message.MessageFragments)
			if err != nil {
				return nil, err
			}
			filteredMessages = append(filteredMessages, message)
		}
	}
	return filteredMessages, nil
}

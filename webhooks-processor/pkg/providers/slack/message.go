// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package slack

import (
	"encoding/json"
)

type SlackMessage struct {
	Blocks []*SlackMessageBlock `json:"blocks,omitempty"`
}

type SlackMessageBlock struct {
	Type   string               `json:"type,omitempty"`
	Text   *SlackMessageField   `json:"text,omitempty"`
	Fields []*SlackMessageField `json:"fields,omitempty"`
}

type SlackMessageField struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

func NewMessage() *SlackMessage {
	return new(SlackMessage)
}

func (m *SlackMessage) AddBlocks(blocks ...*SlackMessageBlock) *SlackMessage {
	m.Blocks = append(m.Blocks, blocks...)
	return m
}

func (m *SlackMessage) JSON() ([]byte, error) {
	return json.Marshal(m)
}

func NewBlock(kind string, text *SlackMessageField, fields ...*SlackMessageField) *SlackMessageBlock {
	return &SlackMessageBlock{
		Type:   kind,
		Text:   text,
		Fields: fields,
	}
}

func NewField(kind string, text string) *SlackMessageField {
	return &SlackMessageField{
		Type: kind,
		Text: text,
	}
}

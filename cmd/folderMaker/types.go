package main

import (
	"encoding/json"
	"fmt"
)

type Config struct {
	Shared   map[string]SharedEntry `json:"shared"`
	BaseDir  BaseDirConfig          `json:"base_dir"`
	Create   []string               `json:"create,omitempty"`
	Sections []ChoiceQuestion       `json:"sections"`
}

type SharedEntry struct {
	dirs     []string
	question *ChoiceQuestion
}

func (e *SharedEntry) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		e.dirs = arr
		return nil
	}
	var q ChoiceQuestion
	if err := json.Unmarshal(data, &q); err != nil {
		return err
	}
	e.question = &q
	return nil
}

type BaseDirConfig struct {
	Questions []BaseQuestion `json:"questions"`
	Format    string         `json:"format"`
}

// BaseQuestion covers both text and choice questions in one struct.
// IsChoice() returns true when Choices is non-empty (inferred from JSON shape).
type BaseQuestion struct {
	ID       string   `json:"id"`
	Prompt   string   `json:"prompt"`
	Header   []string `json:"header"`
	Choices  []string `json:"choices"`
	Values   []string `json:"values"`
	Validate string   `json:"validate"`
	Flag     string   `json:"flag"`
}

func (q *BaseQuestion) IsChoice() bool { return len(q.Choices) > 0 }

type ChoiceQuestion struct {
	Header  []string  `json:"header"`
	Prompt  string    `json:"prompt"`
	Choices []string  `json:"choices"`
	Multi   bool      `json:"multi"`
	Flag    string    `json:"flag"`
	On      []OnEntry `json:"on"`
}

// OnEntry is polymorphic: null → skip, string → one dir, []string → many dirs,
type OnEntry struct {
	skip   bool
	single string
	multi  []string
	full   *Actions
}

func (e *OnEntry) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		e.skip = true
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		e.single = s
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		e.multi = arr
		return nil
	}
	var a Actions
	if err := json.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("invalid on entry: %w", err)
	}
	e.full = &a
	return nil
}

type Actions struct {
	Create []string `json:"create"`
	CD     string   `json:"cd"`
	Steps  []Step   `json:"steps"`
}

// Step is polymorphic: []string → create dirs, { "ref": "..." } → shared lookup,
// object with prompt+on → inline question, object with create/cd → simple op.
type Step struct {
	create   []string
	ref      string
	question *ChoiceQuestion
	op       *OpStep
}

func (s *Step) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		s.create = arr
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if refRaw, ok := raw["ref"]; ok {
		return json.Unmarshal(refRaw, &s.ref)
	}
	_, hasPrompt := raw["prompt"]
	_, hasOn := raw["on"]
	if hasPrompt && hasOn {
		var q ChoiceQuestion
		if err := json.Unmarshal(data, &q); err != nil {
			return err
		}
		s.question = &q
		return nil
	}
	var op OpStep
	if err := json.Unmarshal(data, &op); err != nil {
		return err
	}
	s.op = &op
	return nil
}

type OpStep struct {
	Create []string `json:"create"`
	CD     string   `json:"cd"`
}

type Ctx struct {
	flags  map[string]string
	shared map[string]SharedEntry
}

func (c *Ctx) flagVal(name string) (string, bool) {
	if name == "" {
		return "", false
	}
	v, ok := c.flags[name]
	return v, ok
}

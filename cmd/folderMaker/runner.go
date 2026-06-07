package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func expand(template string, vars map[string]string) string {
	for k, v := range vars {
		template = strings.ReplaceAll(template, "{"+k+"}", v)
	}
	return template
}

func makeDirs(base string, dirs []string) error {
	displayTree(dirs)
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(base, d), 0755); err != nil {
			return err
		}
	}
	return nil
}

func (c *Ctx) runQuestion(q *ChoiceQuestion, path string) (string, error) {
	fv, hasFlag := c.flagVal(q.Flag)
	indices := askChoice(q.Prompt, q.Header, q.Choices, q.Multi, fv, hasFlag)
	cur := path
	for _, idx := range indices {
		if idx >= len(q.On) {
			continue
		}
		newPath, err := c.runOnEntry(&q.On[idx], path)
		if err != nil {
			return cur, err
		}
		if len(indices) == 1 {
			cur = newPath
		}
	}
	return cur, nil
}

func (c *Ctx) runOnEntry(e *OnEntry, path string) (string, error) {
	switch {
	case e.skip:
		return path, nil
	case e.single != "":
		return path, makeDirs(path, []string{e.single})
	case e.multi != nil:
		return path, makeDirs(path, e.multi)
	case e.full != nil:
		return c.runActions(e.full, path)
	}
	return path, nil
}

func (c *Ctx) runActions(a *Actions, path string) (string, error) {
	cur := path
	if len(a.Create) > 0 {
		if err := makeDirs(cur, a.Create); err != nil {
			return "", err
		}
	}
	if a.CD != "" {
		cur = filepath.Join(cur, a.CD)
		if err := os.MkdirAll(cur, 0755); err != nil {
			return "", err
		}
	}
	return c.runSteps(a.Steps, cur)
}

func (c *Ctx) runSteps(steps []Step, path string) (string, error) {
	cur := path
	for _, step := range steps {
		var err error
		switch {
		case step.create != nil:
			err = makeDirs(cur, step.create)
		case step.ref != "":
			entry, ok := c.shared[step.ref]
			if !ok {
				return "", fmt.Errorf("unknown shared ref: %q", step.ref)
			}
			if entry.dirs != nil {
				err = makeDirs(cur, entry.dirs)
			} else {
				cur, err = c.runQuestion(entry.question, cur)
			}
		case step.question != nil:
			cur, err = c.runQuestion(step.question, cur)
		case step.op != nil:
			if len(step.op.Create) > 0 {
				err = makeDirs(cur, step.op.Create)
			}
			if err == nil && step.op.CD != "" {
				cur = filepath.Join(cur, step.op.CD)
				err = os.MkdirAll(cur, 0755)
			}
		}
		if err != nil {
			return "", err
		}
	}
	return cur, nil
}

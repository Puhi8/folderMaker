package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readTree.go provides two subcommands:
//
//   compile <template-dir> [out.json]
//     Walks a template directory and emits a config JSON.
//     A directory containing a config file (main.txt / config.txt / main / config)
//     becomes a section whose subdirectories are the choices.
//     A directory without a config file is always created (goes into "create").
//
//   expand <config.json> <out-dir>
//     Inverse: reads a config JSON and recreates the template directory structure
//     so that running compile on it would reproduce the config.
//
// Config file format (any of the names above):
//   prompt: Which type of items?
//   multi:  true
//   flag:   items
//   header: ======= Title =======
//   header: Second header line

var cfgFileNames = []string{"main.txt", "config.txt", "main", "config"}

type outConfig struct {
	Shared   map[string]interface{} `json:"shared,omitempty"`
	BaseDir  outBaseDir             `json:"base_dir"`
	Create   []string               `json:"create,omitempty"`
	Sections []outQuestion          `json:"sections,omitempty"`
}

type outBaseDir struct {
	Questions []interface{} `json:"questions"`
	Format    string        `json:"format"`
}

type outQuestion struct {
	Prompt  string        `json:"prompt"`
	Header  []string      `json:"header,omitempty"`
	Choices []string      `json:"choices"`
	Multi   bool          `json:"multi,omitempty"`
	Flag    string        `json:"flag,omitempty"`
	On      []interface{} `json:"on"`
}

type outActions struct {
	CD    string        `json:"cd,omitempty"`
	Steps []interface{} `json:"steps,omitempty"`
}

type dirCfg struct {
	prompt  string
	multi   bool
	flag    string
	headers []string
}

func findCfgFile(dir string) string {
	for _, name := range cfgFileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return filepath.Join(dir, name)
		}
	}
	return ""
}

func parseDirCfg(path string) (dirCfg, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return dirCfg{}, err
	}
	var dc dirCfg
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(key)) {
		case "prompt":
			dc.prompt = strings.TrimSpace(val)
		case "multi":
			dc.multi = strings.TrimSpace(val) == "true"
		case "flag":
			dc.flag = strings.TrimSpace(val)
		case "header":
			dc.headers = append(dc.headers, strings.TrimSpace(val))
		}
	}
	return dc, nil
}

func listDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func buildOnEntry(dir, name string) (interface{}, error) {
	// This choice dir has its own config → its contents are a sub-question.
	if findCfgFile(dir) != "" {
		q, err := buildQuestion(dir, name)
		if err != nil {
			return nil, err
		}
		return outActions{CD: name, Steps: []interface{}{q}}, nil
	}

	subdirs, err := listDirs(dir)
	if err != nil {
		return nil, err
	}

	// Leaf choice: just create the directory.
	if len(subdirs) == 0 {
		return name, nil
	}

	// Split subdirs: those with a config become nested question steps;
	// those without are always created (emitted as a []string step so they
	// land inside the cd'd path, not alongside it).
	var always []string
	var questions []interface{}
	for _, sub := range subdirs {
		subPath := filepath.Join(dir, sub)
		if findCfgFile(subPath) != "" {
			q, err := buildQuestion(subPath, sub)
			if err != nil {
				return nil, err
			}
			questions = append(questions, q)
		} else {
			always = append(always, sub)
		}
	}

	var steps []interface{}
	if len(always) > 0 {
		steps = append(steps, always)
	}
	steps = append(steps, questions...)

	return outActions{CD: name, Steps: steps}, nil
}

// buildQuestion reads the config file inside dir and builds an outQuestion
// whose choices are the subdirectories of dir.
func buildQuestion(dir, name string) (outQuestion, error) {
	var dc dirCfg
	if p := findCfgFile(dir); p != "" {
		var err error
		dc, err = parseDirCfg(p)
		if err != nil {
			return outQuestion{}, fmt.Errorf("config in %q: %w", dir, err)
		}
	}
	if dc.prompt == "" {
		dc.prompt = fmt.Sprintf("Choose %s:", name)
	}

	subdirs, err := listDirs(dir)
	if err != nil {
		return outQuestion{}, err
	}

	q := outQuestion{
		Prompt:  dc.prompt,
		Header:  dc.headers,
		Choices: subdirs,
		Multi:   dc.multi,
		Flag:    dc.flag,
		On:      make([]interface{}, len(subdirs)),
	}
	for i, sub := range subdirs {
		q.On[i], err = buildOnEntry(filepath.Join(dir, sub), sub)
		if err != nil {
			return outQuestion{}, err
		}
	}
	return q, nil
}

// compileTemplate walks templateDir and produces an outConfig.
func compileTemplate(templateDir string) (outConfig, error) {
	subdirs, err := listDirs(templateDir)
	if err != nil {
		return outConfig{}, fmt.Errorf("reading %q: %w", templateDir, err)
	}

	cfg := outConfig{
		BaseDir: outBaseDir{Questions: []interface{}{}, Format: ""},
	}
	for _, sub := range subdirs {
		subPath := filepath.Join(templateDir, sub)
		if findCfgFile(subPath) != "" {
			q, err := buildQuestion(subPath, sub)
			if err != nil {
				return outConfig{}, err
			}
			cfg.Sections = append(cfg.Sections, q)
		} else {
			cfg.Create = append(cfg.Create, sub)
		}
	}
	return cfg, nil
}

func runCompile(args []string) {
	if len(args) < 1 {
		showHelp("compile")
		os.Exit(1)
	}

	cfg, err := compileTemplate(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if len(args) >= 2 {
		if err := os.WriteFile(args[1], data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Written to %s\n", args[1])
	} else {
		os.Stdout.Write(data)
	}
}

// ── expand: JSON config → template dir ───────────────────────────────────────

// sanitizeName turns a choice label into a safe directory name.
func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// writeCfgFile writes a main.txt for a question inside dir.
func writeCfgFile(dir string, q ChoiceQuestion) error {
	var sb strings.Builder
	for _, h := range q.Header {
		fmt.Fprintf(&sb, "header: %s\n", h)
	}
	if q.Prompt != "" {
		fmt.Fprintf(&sb, "prompt: %s\n", q.Prompt)
	}
	if q.Multi {
		fmt.Fprintf(&sb, "multi: true\n")
	}
	if q.Flag != "" {
		fmt.Fprintf(&sb, "flag: %s\n", q.Flag)
	}
	return os.WriteFile(filepath.Join(dir, "main.txt"), []byte(sb.String()), 0644)
}

// expandQuestion creates the template directory for a ChoiceQuestion inside parent.
// The question dir is named after its flag (or sectionN as fallback).
func expandQuestion(parent string, q ChoiceQuestion, name string) error {
	qDir := filepath.Join(parent, name)
	if err := os.MkdirAll(qDir, 0755); err != nil {
		return err
	}
	if err := writeCfgFile(qDir, q); err != nil {
		return err
	}

	for i, choice := range q.Choices {
		choiceDir := filepath.Join(qDir, sanitizeName(choice))
		if err := os.MkdirAll(choiceDir, 0755); err != nil {
			return err
		}
		if i < len(q.On) {
			if err := expandOnEntry(choiceDir, &q.On[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// expandOnEntry creates template subdirectory content for an OnEntry.
func expandOnEntry(dir string, e *OnEntry) error {
	switch {
	case e.skip:
		// null → empty choice dir (selecting it does nothing)
	case e.single != "":
		// single string → the choice IS that string, dir already named correctly
	case e.multi != nil:
		// multiple dirs created → create them as plain subdirs
		for _, d := range e.multi {
			if err := os.MkdirAll(filepath.Join(dir, sanitizeName(d)), 0755); err != nil {
				return err
			}
		}
	case e.full != nil:
		if err := expandActions(dir, e.full); err != nil {
			return err
		}
	}
	return nil
}

// expandActions recurses into an Actions block.
func expandActions(dir string, a *Actions) error {
	// Plain dirs to create at this level
	for _, d := range a.Create {
		if err := os.MkdirAll(filepath.Join(dir, sanitizeName(d)), 0755); err != nil {
			return err
		}
	}

	// CD means we step into a sub-dir; process its steps there
	cur := dir
	if a.CD != "" {
		cur = filepath.Join(dir, sanitizeName(a.CD))
		if err := os.MkdirAll(cur, 0755); err != nil {
			return err
		}
	}

	return expandSteps(cur, a.Steps)
}

// expandSteps handles the polymorphic Steps slice.
func expandSteps(dir string, steps []Step) error {
	subQuestionIdx := 0
	for i := range steps {
		s := &steps[i]
		switch {
		case s.create != nil:
			for _, d := range s.create {
				if err := os.MkdirAll(filepath.Join(dir, sanitizeName(d)), 0755); err != nil {
					return err
				}
			}
		case s.question != nil:
			name := s.question.Flag
			if name == "" {
				name = fmt.Sprintf("question_%d", subQuestionIdx)
			}
			subQuestionIdx++
			if err := expandQuestion(dir, *s.question, name); err != nil {
				return err
			}
		case s.op != nil:
			cur := dir
			if s.op.CD != "" {
				cur = filepath.Join(dir, sanitizeName(s.op.CD))
				if err := os.MkdirAll(cur, 0755); err != nil {
					return err
				}
			}
			for _, d := range s.op.Create {
				if err := os.MkdirAll(filepath.Join(cur, sanitizeName(d)), 0755); err != nil {
					return err
				}
			}
		case s.ref != "":
			// shared refs can't be inlined without the full config; skip
		}
	}
	return nil
}

func runExpand(args []string) {
	if len(args) < 2 {
		showHelp("expand")
		os.Exit(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "expand: read config: %v\n", err)
		os.Exit(1)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "expand: bad JSON: %v\n", err)
		os.Exit(1)
	}

	outDir := args[1]
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "expand: mkdir: %v\n", err)
		os.Exit(1)
	}

	// Unconditional dirs
	for _, d := range cfg.Create {
		if err := os.MkdirAll(filepath.Join(outDir, sanitizeName(d)), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "expand: %v\n", err)
			os.Exit(1)
		}
	}

	for i, section := range cfg.Sections {
		name := section.Flag
		if name == "" {
			name = fmt.Sprintf("section_%d", i)
		}
		if err := expandQuestion(outDir, section, name); err != nil {
			fmt.Fprintf(os.Stderr, "expand: section %d: %v\n", i, err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "Template written to %s\n", outDir)
}

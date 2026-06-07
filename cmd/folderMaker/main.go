package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "compile":
			runCompile(args[1:])
			return
		case "expand":
			runExpand(args[1:])
			return
		case "help":
			runHelp(args[1:])
			return
		case "-h", "--help":
			showHelp("")
			return
		}
	}

	flags := map[string]string{}
	var configPath string

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			key := strings.TrimPrefix(args[i], "-")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[key] = "true"
			}
		} else if configPath == "" {
			configPath = args[i]
		}
	}

	if configPath == "" {
		showHelp("")
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read config %q: %v\n", configPath, err)
		os.Exit(1)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Bad config JSON: %v\n", err)
		os.Exit(1)
	}
	if config.Shared == nil {
		config.Shared = map[string]SharedEntry{}
	}

	ctx := &Ctx{flags: flags, shared: config.Shared}

	// Cleanup on Ctrl+C
	var cleanupMu sync.Mutex
	var cleanupDir string
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cleanupMu.Lock()
		dir := cleanupDir
		cleanupMu.Unlock()
		fmt.Fprintln(os.Stderr, "\nInterrupted.")
		if dir != "" {
			fmt.Fprintf(os.Stderr, "Removing %s\n", dir)
			os.RemoveAll(dir)
		}
		os.Exit(1)
	}()

	// base-dir
	vars := map[string]string{}
	for i, q := range config.BaseDir.Questions {
		id := q.ID
		if id == "" {
			id = q.Flag
		}
		if id == "" {
			id = fmt.Sprintf("field%d", i)
		}
		fv, hasFlag := ctx.flagVal(q.Flag)
		if q.IsChoice() {
			idx := askChoice(q.Prompt, q.Header, q.Choices, false, fv, hasFlag)[0]
			if idx < len(q.Values) {
				vars[id] = q.Values[idx]
			} else {
				vars[id] = q.Choices[idx]
			}
		} else {
			vars[id] = askText(q.Prompt, q.Validate, fv, hasFlag)
		}
	}

	baseName := expand(config.BaseDir.Format, vars)
	fmt.Printf("\nCreating: %s\n", baseName)

	if _, err := os.Stat(baseName); err == nil {
		fmt.Fprintf(os.Stderr, "Error: %q already exists.\n", baseName)
		os.Exit(1)
	}
	if err := os.Mkdir(baseName, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}
	cleanupMu.Lock()
	cleanupDir = baseName
	cleanupMu.Unlock()

	if len(config.Create) > 0 {
		if err := makeDirs(baseName, config.Create); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// sections
	for i := range config.Sections {
		if _, err := ctx.runQuestion(&config.Sections[i], baseName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("\nDone! Folders created successfully.")
}

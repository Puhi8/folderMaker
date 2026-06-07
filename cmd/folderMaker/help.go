package main

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed help/main.txt
var helpMain string

//go:embed help/compile.txt
var helpCompile string

//go:embed help/expand.txt
var helpExpand string

var helpTopics = map[string]string{
	"":        helpMain,
	"main":    helpMain,
	"compile": helpCompile,
	"expand":  helpExpand,
}

func showHelp(topic string) {
	text, ok := helpTopics[topic]
	if !ok {
		fmt.Fprintf(os.Stderr, "No help for %q. Topics: main, compile, expand\n", topic)
		os.Exit(1)
	}
	fmt.Print(text)
}

func runHelp(args []string) {
	topic := ""
	if len(args) > 0 {
		topic = args[0]
	}
	showHelp(topic)
}

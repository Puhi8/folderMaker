package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var scanner = bufio.NewScanner(os.Stdin)

func readLine(promptText string) string {
	fmt.Print(promptText)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func displayChoices(choices []string) {
	for i, c := range choices {
		fmt.Printf("  %d. %s\n", i, c)
	}
}

func displayTree(items []string) {
	last := len(items) - 1
	for i, item := range items {
		branch := "├"
		if i == last {
			branch = "└"
		}
		fmt.Printf("  %s──%s\n", branch, item)
	}
}

func validateText(val, rule string) bool {
	if rule == "date" {
		return len(val) == 10
	}
	return len(val) > 0
}

func askText(promptStr, validate, flagVal string, hasFlag bool) string {
	if hasFlag {
		if validateText(flagVal, validate) {
			fmt.Printf("(flag) %s\n", flagVal)
			return flagVal
		}
		fmt.Printf("Flag value %q is invalid, falling back to prompt.\n", flagVal)
	}
	for {
		answer := readLine(promptStr)
		if validateText(answer, validate) {
			return answer
		}
		fmt.Println("Invalid input. Please try again.")
	}
}

func askChoice(promptStr string, header, choices []string, multi bool, flagVal string, hasFlag bool) []int {
	max := len(choices) - 1
	if hasFlag {
		if n, err := strconv.Atoi(flagVal); err == nil && n >= 0 && n <= max {
			fmt.Printf("(flag) %d. %s\n", n, choices[n])
			return []int{n}
		}
		fmt.Printf("Flag value %q is invalid, falling back to prompt.\n", flagVal)
	}
	if len(header) > 0 {
		fmt.Println()
		for _, line := range header {
			fmt.Println(line)
		}
	}
	displayChoices(choices)
	for {
		answer := readLine(promptStr)
		if n, err := strconv.Atoi(answer); err == nil && n >= 0 && n <= max {
			return []int{n}
		}
		if multi && len(answer) > 1 {
			var indices []int
			valid := true
			for _, ch := range answer {
				if ch < '0' || ch > '9' {
					valid = false
					break
				}
				n := int(ch - '0')
				if n > max {
					valid = false
					break
				}
				indices = append(indices, n)
			}
			if valid {
				return indices
			}
		}
		fmt.Printf("Enter a number from 0 to %d.\n", max)
	}
}

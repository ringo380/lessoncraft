package lesson

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"
	"time"
)

// SimpleParser is an implementation of the Parser interface that uses regular expressions
// to parse markdown content into lessons. It's designed to be a lightweight alternative
// to more complex markdown parsers that rely on external libraries.
//
// This parser recognizes the following markdown elements:
// - Title: A line starting with a single # followed by text
// - Description: Text following the title until the first code block or heading
// - Docker blocks: Code blocks with the docker language identifier (```docker)
// - Expect blocks: Code blocks with the expect language identifier (```expect)
// - Question blocks: Code blocks with the question language identifier (```question)
type SimpleParser struct{}

// NewSimpleParser creates a new SimpleParser instance.
// This function is used by NewParser() to create the default parser implementation.
func NewSimpleParser() *SimpleParser {
	return &SimpleParser{}
}

var (
	// simpleTitleRegex matches a markdown heading level 1 (# Title)
	simpleTitleRegex = regexp.MustCompile(`^#\s+(.+)$`)

	// simpleDockerBlockRegex matches a docker code block (```docker\n...\n```)
	simpleDockerBlockRegex = regexp.MustCompile("(?s)```docker\n(.*?)\n```")

	// simpleExpectBlockRegex matches an expect code block (```expect\n...\n```)
	simpleExpectBlockRegex = regexp.MustCompile("(?s)```expect\n(.*?)\n```")

	// simpleQuestionRegex matches a question code block (```question\n...\n```)
	simpleQuestionRegex = regexp.MustCompile("(?s)```question\n(.*?)\n```")

	// simpleBlockRegex matches any of the above code blocks and captures the type and content
	simpleBlockRegex = regexp.MustCompile("(?s)```(docker|expect|question)\n(.*?)\n```")
)

// Parse implements the Parser interface by reading markdown content from the provided reader
// and converting it into a Lesson object. It extracts the title, description, and steps
// from the markdown content using regular expressions.
//
// The parsing process follows these steps:
// 1. Read the entire content into memory
// 2. Extract the title from the first heading
// 3. Extract the description from the text following the title
// 4. Find all code blocks (docker, expect, question) in order
// 5. Process each block to build the lesson steps
//
// Each docker block creates a new step or adds commands to an existing step.
// Expect and question blocks add metadata to the current step.
//
// Returns a pointer to a Lesson object and any error encountered during parsing.
func (p *SimpleParser) Parse(r io.Reader) (*Lesson, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}

	content := buf.String()
	lesson := &Lesson{
		Steps: []LessonStep{},
	}

	// Extract title and description
	scanner := bufio.NewScanner(strings.NewReader(content))
	var titleFound bool
	var descLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if !titleFound {
			if match := simpleTitleRegex.FindStringSubmatch(line); len(match) > 1 {
				lesson.Title = match[1]
				titleFound = true
				continue
			}
		} else if len(descLines) == 0 && strings.TrimSpace(line) != "" {
			// First non-empty line after title is the description
			descLines = append(descLines, line)
		} else if len(descLines) > 0 && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "```") {
			// Continue adding to description until we hit a heading or code block
			descLines = append(descLines, line)
		} else if len(descLines) > 0 {
			// We've reached the end of the description
			break
		}
	}

	if len(descLines) > 0 {
		lesson.Description = strings.Join(descLines, " ")
	}

	// Find all blocks (docker, expect, question) in order
	blockMatches := simpleBlockRegex.FindAllStringSubmatch(content, -1)

	var currentStep *LessonStep

	// Process each block in order
	for _, match := range blockMatches {
		if len(match) < 3 {
			continue
		}

		blockType := match[1]
		blockContent := match[2]

		switch blockType {
		case "docker":
			commands := parseCommands(blockContent)

			// If there's already a step and it doesn't have an expected output or question,
			// add these commands to that step instead of creating a new one
			if currentStep != nil &&
				currentStep.Expected == "" &&
				currentStep.Question == "" {
				currentStep.Commands = append(currentStep.Commands, commands...)
			} else {
				// Create a new step
				currentStep = &LessonStep{
					ID:       generateStepID(len(lesson.Steps)),
					Commands: commands,
					Timeout:  5 * time.Minute,
				}

				lesson.Steps = append(lesson.Steps, *currentStep)
				// Update the pointer to point to the step in the slice
				currentStep = &lesson.Steps[len(lesson.Steps)-1]
			}

		case "expect":
			if currentStep != nil {
				currentStep.Expected = strings.TrimSpace(blockContent)
				// After setting expected output, we're done with this step
				currentStep = nil
			}

		case "question":
			if currentStep != nil {
				currentStep.Question = strings.TrimSpace(blockContent)
				// After setting question, we're done with this step
				currentStep = nil
			}
		}
	}

	return lesson, nil
}

// parseCommands extracts individual commands from a docker code block.
// It splits the content by newlines, trims whitespace, and filters out empty lines.
//
// Parameters:
//   - content: The content of a docker code block
//
// Returns:
//   - A slice of strings, each representing a command to be executed
func parseCommands(content string) []string {
	var commands []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		cmd := strings.TrimSpace(scanner.Text())
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// generateStepID creates a unique identifier for a lesson step based on its index.
// The ID follows the pattern "step-a", "step-b", etc.
//
// Parameters:
//   - index: The zero-based index of the step in the lesson
//
// Returns:
//   - A string ID for the step
func generateStepID(index int) string {
	return "step-" + string(rune('a'+index))
}

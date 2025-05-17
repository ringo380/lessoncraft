package lesson

import (
	"io"
)

// Parser is an interface for parsing markdown content into lessons.
// Implementations of this interface should be able to parse markdown content
// with specialized code blocks for docker commands, expected outputs, and questions.
type Parser interface {
	// Parse reads markdown content from the provided reader and converts it into a Lesson object.
	// The markdown content should follow the LessonCraft format, which includes:
	// - A title (# Title)
	// - A description (text following the title)
	// - Code blocks for docker commands (```docker)
	// - Code blocks for expected outputs (```expect)
	// - Code blocks for questions (```question)
	//
	// Returns a pointer to a Lesson object and any error encountered during parsing.
	Parse(r io.Reader) (*Lesson, error)
}

// NewParser creates a new parser for LessonCraft markdown content.
// It returns an implementation of the Parser interface that can parse
// markdown content into Lesson objects.
//
// Currently, it returns a SimpleParser implementation, which uses regular expressions
// to parse the markdown content.
func NewParser() Parser {
	return NewSimpleParser()
}

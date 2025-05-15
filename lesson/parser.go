package lesson

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type Parser struct {
	md goldmark.Markdown
}

func NewParser() *Parser {
	md := goldmark.New(
		goldmark.WithExtensions(
			&lessonExtension{},
		),
	)
	return &Parser{md: md}
}

type lessonExtension struct{}

func (e *lessonExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			parser.NewBlockParserFunc(
				func(b []byte) (ast.Node, bool) {
					return parseCodeBlock(b)
				},
			),
		),
	)
}

var (
	dockerBlockRegex = regexp.MustCompile("^```docker\\s*(.*?)\\s*$")
	expectBlockRegex = regexp.MustCompile("^```expect\\s*(.*?)\\s*$")
	questionRegex    = regexp.MustCompile("^```question\\s*(.*?)\\s*$")
)

func parseCodeBlock(b []byte) (ast.Node, bool) {
	lines := bytes.Split(b, []byte{'\n'})
	if len(lines) == 0 {
		return nil, false
	}

	firstLine := string(lines[0])
	switch {
	case dockerBlockRegex.MatchString(firstLine):
		return parseDockerBlock(lines[1:])
	case expectBlockRegex.MatchString(firstLine):
		return parseExpectBlock(lines[1:])
	case questionRegex.MatchString(firstLine):
		return parseQuestionBlock(lines[1:])
	}
	return nil, false
}

func parseDockerBlock(lines [][]byte) (ast.Node, bool) {
	var commands []string
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("```")) {
			break
		}
		cmd := strings.TrimSpace(string(line))
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}
	return &DockerBlock{Commands: commands}, true
}

func parseExpectBlock(lines [][]byte) (ast.Node, bool) {
	var buf bytes.Buffer
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("```")) {
			break
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return &ExpectBlock{Expected: strings.TrimSpace(buf.String())}, true
}

func parseQuestionBlock(lines [][]byte) (ast.Node, bool) {
	var buf bytes.Buffer
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("```")) {
			break
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return &QuestionBlock{Question: strings.TrimSpace(buf.String())}, true
}

type DockerBlock struct {
	Commands []string
}

func (b *DockerBlock) Kind() ast.NodeKind {
	return ast.KindCodeBlock
}

func (b *DockerBlock) Dump(source []byte, level int) {
	// Implement dump for debugging
}

type ExpectBlock struct {
	Expected string
}

func (b *ExpectBlock) Kind() ast.NodeKind {
	return ast.KindCodeBlock
}

func (b *ExpectBlock) Dump(source []byte, level int) {
	// Implement dump for debugging
}

type QuestionBlock struct {
	Question string
}

func (b *QuestionBlock) Kind() ast.NodeKind {
	return ast.KindCodeBlock
}

func (b *QuestionBlock) Dump(source []byte, level int) {
	// Implement dump for debugging
}

func (p *Parser) Parse(r io.Reader) (*Lesson, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}

	doc := p.md.Parser().Parse(text.NewReader(buf.Bytes()))
	lesson := &Lesson{
		Steps: []LessonStep{},
	}

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch v := n.(type) {
		case *ast.Heading:
			if v.Level == 1 {
				lesson.Title = string(n.Text(buf.Bytes()))
			}
		case *ast.Paragraph:
			if lesson.Description == "" {
				lesson.Description = string(n.Text(buf.Bytes()))
			}
		case *DockerBlock:
			step := LessonStep{
				Commands: v.Commands,
				Timeout:  5 * time.Minute,
			}
			if len(lesson.Steps) > 0 {
				lastStep := &lesson.Steps[len(lesson.Steps)-1]
				if lastStep.Expected == "" && lastStep.Question == "" {
					lastStep.Commands = append(lastStep.Commands, v.Commands...)
					return ast.WalkContinue, nil
				}
			}
			lesson.Steps = append(lesson.Steps, step)
		case *ExpectBlock:
			if len(lesson.Steps) > 0 {
				lesson.Steps[len(lesson.Steps)-1].Expected = v.Expected
			}
		case *QuestionBlock:
			if len(lesson.Steps) > 0 {
				lesson.Steps[len(lesson.Steps)-1].Question = v.Question
			}
		}
		return ast.WalkContinue, nil
	})

	return lesson, err
}

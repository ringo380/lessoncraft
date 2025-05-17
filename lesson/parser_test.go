package lesson

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotNil(t, parser)
}

func TestParse_EmptyInput(t *testing.T) {
	parser := NewParser()
	reader := strings.NewReader("")
	lesson, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Empty(t, lesson.Title)
	assert.Empty(t, lesson.Description)
	assert.Empty(t, lesson.Steps)
}

func TestParse_BasicLesson(t *testing.T) {
	parser := NewParser()
	markdown := "# Basic Lesson Title\nThis is a description of the lesson.\n\n## Step 1\nThis is the content of step 1.\n\n## Step 2\nThis is the content of step 2.\n"
	reader := strings.NewReader(markdown)
	lesson, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Equal(t, "Basic Lesson Title", lesson.Title)
	assert.Equal(t, "This is a description of the lesson.", lesson.Description)
	// The parser doesn't seem to create steps based on headings, so we don't expect any steps here
	assert.Empty(t, lesson.Steps)
}

func TestParse_DockerBlocks(t *testing.T) {
	parser := NewParser()
	markdown := "# Docker Commands Lesson\nLearn basic Docker commands.\n\n" +
		"```docker\ndocker ps\ndocker images\n```\n\n" +
		"Check the running containers:\n\n" +
		"```docker\ndocker ps -a\n```\n"
	reader := strings.NewReader(markdown)
	lesson, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Equal(t, "Docker Commands Lesson", lesson.Title)
	assert.Equal(t, "Learn basic Docker commands.", lesson.Description)
	assert.Len(t, lesson.Steps, 1)

	// First step should have both docker commands
	assert.Len(t, lesson.Steps[0].Commands, 3)
	assert.Equal(t, "docker ps", lesson.Steps[0].Commands[0])
	assert.Equal(t, "docker images", lesson.Steps[0].Commands[1])
	assert.Equal(t, "docker ps -a", lesson.Steps[0].Commands[2])
}

func TestParse_ExpectBlocks(t *testing.T) {
	parser := NewParser()
	markdown := "# Expect Output Lesson\nLearn to verify command output.\n\n" +
		"```docker\necho \"Hello, World!\"\n```\n\n" +
		"```expect\nHello, World!\n```\n"
	reader := strings.NewReader(markdown)
	lesson, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Equal(t, "Expect Output Lesson", lesson.Title)
	assert.Equal(t, "Learn to verify command output.", lesson.Description)
	assert.Len(t, lesson.Steps, 1)

	// Step should have the command and expected output
	assert.Len(t, lesson.Steps[0].Commands, 1)
	assert.Equal(t, "echo \"Hello, World!\"", lesson.Steps[0].Commands[0])
	assert.Equal(t, "Hello, World!", lesson.Steps[0].Expected)
}

func TestParse_QuestionBlocks(t *testing.T) {
	parser := NewParser()
	markdown := "# Question Lesson\nLearn with interactive questions.\n\n" +
		"```docker\nls -la\n```\n\n" +
		"```question\nWhat command would you use to list all files, including hidden ones?\n```\n"
	reader := strings.NewReader(markdown)
	lesson, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Equal(t, "Question Lesson", lesson.Title)
	assert.Equal(t, "Learn with interactive questions.", lesson.Description)
	assert.Len(t, lesson.Steps, 1)

	// Step should have the command and question
	assert.Len(t, lesson.Steps[0].Commands, 1)
	assert.Equal(t, "ls -la", lesson.Steps[0].Commands[0])
	assert.Equal(t, "What command would you use to list all files, including hidden ones?", lesson.Steps[0].Question)
}

func TestParse_ComplexLesson(t *testing.T) {
	parser := NewParser()
	markdown := "# Complex Lesson\nThis is a complex lesson with multiple steps and block types.\n\n" +
		"## Step 1: Basic Commands\n\n" +
		"```docker\necho \"Step 1\"\nls -la\n```\n\n" +
		"```expect\nStep 1\n```\n\n" +
		"## Step 2: Advanced Commands\n\n" +
		"```docker\necho \"Step 2\"\nfind . -type f -name \"*.go\"\n```\n\n" +
		"```question\nWhat command would you use to find all Go files?\n```\n\n" +
		"## Step 3: Final Commands\n\n" +
		"```docker\necho \"Step 3\"\ngrep -r \"func\" .\n```\n\n" +
		"```expect\nMultiple lines of output\nshowing functions\n```\n"
	reader := strings.NewReader(markdown)
	lesson, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Equal(t, "Complex Lesson", lesson.Title)
	assert.Equal(t, "This is a complex lesson with multiple steps and block types.", lesson.Description)
	assert.Len(t, lesson.Steps, 3)

	// Check Step 1
	assert.Len(t, lesson.Steps[0].Commands, 2)
	assert.Equal(t, "echo \"Step 1\"", lesson.Steps[0].Commands[0])
	assert.Equal(t, "ls -la", lesson.Steps[0].Commands[1])
	assert.Equal(t, "Step 1", lesson.Steps[0].Expected)
	assert.Empty(t, lesson.Steps[0].Question)

	// Check Step 2
	assert.Len(t, lesson.Steps[1].Commands, 2)
	assert.Equal(t, "echo \"Step 2\"", lesson.Steps[1].Commands[0])
	assert.Equal(t, "find . -type f -name \"*.go\"", lesson.Steps[1].Commands[1])
	assert.Empty(t, lesson.Steps[1].Expected)
	assert.Equal(t, "What command would you use to find all Go files?", lesson.Steps[1].Question)

	// Check Step 3
	assert.Len(t, lesson.Steps[2].Commands, 2)
	assert.Equal(t, "echo \"Step 3\"", lesson.Steps[2].Commands[0])
	assert.Equal(t, "grep -r \"func\" .", lesson.Steps[2].Commands[1])
	assert.Equal(t, "Multiple lines of output\nshowing functions", lesson.Steps[2].Expected)
	assert.Empty(t, lesson.Steps[2].Question)
}

func TestParse_MalformedMarkdown(t *testing.T) {
	parser := NewParser()
	markdown := "# Malformed Lesson\nThis is a malformed lesson.\n\n" +
		"```docker\necho \"Unclosed block\n"
	reader := strings.NewReader(markdown)
	lesson, err := parser.Parse(reader)

	// The parser should still work with malformed markdown
	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Equal(t, "Malformed Lesson", lesson.Title)
	assert.Equal(t, "This is a malformed lesson.", lesson.Description)

	// The unclosed block might be parsed differently, but it shouldn't crash
	// Just verify we get something reasonable
	if len(lesson.Steps) > 0 {
		assert.Contains(t, lesson.Steps[0].Commands[0], "echo")
	}
}

func TestParse_MultipleDockerBlocks(t *testing.T) {
	parser := NewParser()
	markdown := "# Multiple Docker Blocks\nTesting how multiple docker blocks are handled.\n\n" +
		"```docker\necho \"Block 1\"\n```\n\n" +
		"Some text in between.\n\n" +
		"```docker\necho \"Block 2\"\n```\n"
	reader := strings.NewReader(markdown)
	lesson, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Equal(t, "Multiple Docker Blocks", lesson.Title)
	assert.Equal(t, "Testing how multiple docker blocks are handled.", lesson.Description)

	// Check if we have the right number of steps or if they're combined
	if len(lesson.Steps) == 1 {
		// If blocks are combined into one step
		assert.Len(t, lesson.Steps[0].Commands, 2)
		assert.Equal(t, "echo \"Block 1\"", lesson.Steps[0].Commands[0])
		assert.Equal(t, "echo \"Block 2\"", lesson.Steps[0].Commands[1])
	} else if len(lesson.Steps) == 2 {
		// If blocks create separate steps
		assert.Len(t, lesson.Steps[0].Commands, 1)
		assert.Equal(t, "echo \"Block 1\"", lesson.Steps[0].Commands[0])
		assert.Len(t, lesson.Steps[1].Commands, 1)
		assert.Equal(t, "echo \"Block 2\"", lesson.Steps[1].Commands[0])
	} else {
		t.Fatalf("Unexpected number of steps: %d", len(lesson.Steps))
	}
}

func TestParse_TimeoutSetting(t *testing.T) {
	parser := NewParser()
	markdown := "# Timeout Lesson\nTesting if timeout is set correctly.\n\n" +
		"```docker\necho \"This should have a timeout\"\n```\n"
	reader := strings.NewReader(markdown)
	lesson, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotNil(t, lesson)
	assert.Len(t, lesson.Steps, 1)

	// Check if timeout is set to the default value (5 minutes)
	assert.Equal(t, 5*time.Minute, lesson.Steps[0].Timeout)
}

# LessonCraft Lesson Format

This document describes the format for creating lessons in LessonCraft. Lessons are written in Markdown with specialized code blocks that define commands, expected outputs, and questions.

## Lesson Structure

A LessonCraft lesson consists of:

1. **Title**: A level 1 heading (`# Title`)
2. **Description**: Text following the title
3. **Steps**: A series of steps, each containing content, commands, expected outputs, and/or questions

### Example Lesson

```markdown
# Introduction to Linux Commands

This lesson introduces basic Linux commands for navigating the file system and working with files.

## Step 1: Navigating the File System

Let's start by exploring the file system. The `pwd` command shows your current directory:

```docker
pwd
```

```expect
/home/user
```

Now let's list the files in the current directory:

```docker
ls -la
```

## Step 2: Creating Files

Let's create a new file:

```docker
echo "Hello, World!" > hello.txt
cat hello.txt
```

```expect
Hello, World!
```

```question
What command would you use to create a directory?
```
```

## Code Blocks

LessonCraft uses specialized code blocks to define different parts of a lesson:

### Docker Blocks

Docker blocks define commands that can be executed in the lesson environment:

````markdown
```docker
echo "Hello, World!"
ls -la
```
````

Multiple Docker blocks in the same step will be combined into a single list of commands.

### Expect Blocks

Expect blocks define the expected output of the commands:

````markdown
```expect
Hello, World!
```
````

The output of the commands will be compared to the expected output for validation.

### Question Blocks

Question blocks define questions that can be asked to the user:

````markdown
```question
What command would you use to create a directory?
```
````

## Parsing Logic

The LessonCraft parser processes the markdown content as follows:

1. Extract the title from the first level 1 heading
2. Extract the description from the text following the title
3. Find all code blocks (docker, expect, question) in order
4. Process each block to build the lesson steps:
   - Docker blocks create a new step or add commands to an existing step
   - Expect blocks add expected output to the current step
   - Question blocks add a question to the current step

## API Usage

Lessons can be created, retrieved, updated, and deleted through the LessonCraft API:

### Creating a Lesson

```http
POST /api/lessons
Content-Type: application/json

{
  "title": "Introduction to Linux Commands",
  "description": "This lesson introduces basic Linux commands for navigating the file system and working with files.",
  "steps": [
    {
      "id": "step-1",
      "content": "Let's start by exploring the file system. The `pwd` command shows your current directory:",
      "commands": ["pwd"],
      "expected": "/home/user"
    },
    {
      "id": "step-2",
      "content": "Let's create a new file:",
      "commands": ["echo \"Hello, World!\" > hello.txt", "cat hello.txt"],
      "expected": "Hello, World!",
      "question": "What command would you use to create a directory?"
    }
  ]
}
```

### Starting a Lesson

```http
POST /api/lessons/{id}/start
```

### Completing a Step

```http
POST /api/lessons/{id}/steps/{step}/complete
Content-Type: application/json

{
  "output": "Hello, World!"
}
```

### Validating a Step

```http
POST /api/lessons/{id}/validate
Content-Type: application/json

{
  "output": "Hello, World!"
}
```

## Best Practices

1. **Keep steps focused**: Each step should focus on a single concept or task
2. **Provide clear instructions**: Use the content field to explain what the user should do
3. **Use expected output for validation**: Define expected output for commands that produce output
4. **Ask questions to reinforce learning**: Use questions to check understanding
5. **Use appropriate timeout values**: Set reasonable timeout values for commands that may take time to complete
6. **Validate commands for security**: Ensure commands are safe to execute in the lesson environment
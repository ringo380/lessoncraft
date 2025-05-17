# Contributing to LessonCraft

Thank you for your interest in contributing to LessonCraft! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please read and follow our [Code of Conduct](https://github.com/ringo380/lessoncraft/blob/master/.github/CODE_OF_CONDUCT.md) to foster an inclusive and respectful community.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Set up your development environment as described in the [README.md](README.md)
4. Create a new branch for your changes
5. Make your changes
6. Run tests to ensure your changes don't break existing functionality
7. Submit a pull request

## Development Environment

See the [README.md](README.md) for instructions on setting up your development environment.

## Testing

Before submitting a pull request, make sure to run the tests:

```bash
go test ./...
```

For more comprehensive testing, including race condition detection and coverage reporting:

```bash
go test -race -coverprofile=coverage.txt -covermode=atomic ./...
```

## Pull Request Process

1. Ensure your code follows the project's coding style and conventions
2. Update the documentation as necessary
3. Include tests for new functionality
4. Ensure all tests pass
5. Update the README.md with details of changes to the interface, if applicable
6. The pull request will be merged once it has been reviewed and approved by a maintainer

## Coding Style

- Follow standard Go coding conventions
- Use meaningful variable and function names
- Write comments for exported functions and complex logic
- Use interfaces for dependency injection to facilitate testing

## Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests liberally after the first line

## Documentation

- Update documentation when changing code
- Use godoc-compatible comments for exported functions and types
- Include examples where appropriate

## License

By contributing to LessonCraft, you agree that your contributions will be licensed under the project's [LICENSE](LICENSE).

## Questions?

If you have any questions or need help, please open an issue or contact the maintainers.
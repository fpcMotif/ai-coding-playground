# Contributing to FFmpeg-Go-Relay

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Assume good intentions
- Help others learn and grow

## Getting Started

### Prerequisites
- Go 1.21 or later
- Git
- Docker (for testing)
- Basic knowledge of RTMP protocol (helpful but not required)

### Local Development Setup

```bash
# Clone the repository
git clone https://github.com/your-org/ffmpeg-go-relay.git
cd ffmpeg-go-relay

# Install dependencies
go mod download

# Verify setup
go test ./... -v

# Run the relay locally
go run ./cmd/relay -listen :1935 -upstream rtmp://upstream.example.com:1935/app
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-you-are-fixing
```

### 2. Make Your Changes

Follow the coding standards below.

### 3. Test Your Changes

```bash
# Run unit tests
go test -v ./...

# Run tests with race detector
go test -race ./...

# Check coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run integration tests
go test -v ./test -timeout 60s

# Run benchmarks
go test -bench=. -benchmem ./test
```

### 4. Format and Lint

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Fix common issues
go mod tidy
```

### 5. Commit and Push

```bash
# Commit with descriptive message
git commit -m "feature: add new feature name

Detailed description of changes, why they were made, and any
considerations or trade-offs."

# Push to your fork
git push origin feature/your-feature-name
```

### 6. Create a Pull Request

- Title: Clear, concise description
- Description: Explain what, why, and how
- Link related issues
- Include before/after examples if applicable

## Coding Standards

### Go Style Guide
We follow the [Effective Go](https://golang.org/doc/effective_go) guidelines:

- Use `camelCase` for variables and functions
- Use `PascalCase` for exported types and functions
- Write self-documenting code with clear variable names
- Add comments for exported functions and complex logic
- Keep functions small and focused

### Code Organization

```
project/
├── cmd/
│   └── relay/           # Command-line entry point
├── internal/
│   ├── auth/            # Authentication
│   ├── circuit/         # Circuit breaker
│   ├── config/          # Configuration
│   ├── httpserver/      # HTTP endpoints
│   ├── logger/          # Logging
│   ├── metrics/         # Prometheus metrics
│   ├── middleware/      # Rate limiting, connection limits
│   ├── pool/            # Buffer pooling
│   ├── relay/           # Core relay logic
│   ├── retry/           # Retry logic
│   └── validator/       # URL validation
├── test/                # Integration and benchmark tests
└── README.md            # Documentation
```

### Error Handling

Prefer explicit error handling:

```go
// Good
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Avoid
if err != nil {
    log.Fatal(err)
}

// Avoid
_ = someFunc() // Don't ignore errors silently
```

### Testing

Write table-driven tests:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    interface{}
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        {
            name:    "invalid input",
            input:   invalidInput,
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Comments

Document exported functions and complex logic:

```go
// Authenticate validates a token against the configured token list.
// Returns nil if the token is valid, otherwise returns an error.
func (ta *TokenAuthenticator) Authenticate(token string) error {
    // ...
}
```

## Contribution Areas

### High Priority
- Bug fixes
- Security improvements
- Performance optimizations
- Documentation improvements
- Test coverage increases

### Medium Priority
- Feature enhancements
- Code refactoring
- Logging improvements
- Error message clarity

### Lower Priority (Discuss First)
- New major features
- Breaking changes
- Architecture changes
- Dependency updates

## Pull Request Checklist

Before submitting a PR:

- [ ] Tests added/updated
- [ ] Code formatted (`go fmt`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] All tests pass (`go test ./...`)
- [ ] Coverage maintained or improved
- [ ] Commit messages are clear
- [ ] Documentation updated (README, comments)
- [ ] Security implications reviewed
- [ ] Performance implications reviewed
- [ ] No new dependencies added without justification

## Review Process

### Reviewer Responsibilities
1. Check code quality and style
2. Verify tests are adequate
3. Consider security implications
4. Review performance impact
5. Suggest improvements constructively

### Author Responsibilities
1. Respond to feedback promptly
2. Make requested changes or explain disagreement
3. Keep PR focused on single concern
4. Ensure tests pass before requesting review

### Approval Criteria
- At least 1 approval required
- All conversations resolved
- CI pipeline passes
- Code coverage maintained (70%+)

## Development Tips

### Debugging
```bash
# Run with debug logging
./relay config.json

# Use pprof for profiling
go test -cpuprofile=cpu.prof ./test
go tool pprof cpu.prof

# Check memory allocations
go test -benchmem ./test
```

### Common Tasks

#### Adding a New Feature
1. Create issue for discussion
2. Design in comments/PR description
3. Implement with tests
4. Add config option if needed
5. Update documentation

#### Fixing a Bug
1. Create minimal reproduction test
2. Verify test fails
3. Fix the bug
4. Verify test passes
5. Check for similar issues elsewhere

#### Improving Performance
1. Add benchmark test
2. Get baseline metrics
3. Make change
4. Compare metrics
5. Document performance impact

## Issue Guidelines

### When Creating an Issue

**Bugs:**
1. Clear title describing the issue
2. Steps to reproduce
3. Expected vs actual behavior
4. Environment info (OS, Go version)
5. Relevant logs or error messages

**Features:**
1. Clear description of desired behavior
2. Why you need this feature
3. Alternative solutions considered
4. Proposed implementation (optional)

**Questions:**
1. Clear description of what you're trying to do
2. What you've already tried
3. Relevant code snippets

### Triaging

Issues are triaged and labeled:
- `bug`: Something is broken
- `feature`: New functionality request
- `enhancement`: Improvement to existing feature
- `docs`: Documentation improvements
- `good-first-issue`: Good for new contributors
- `help-wanted`: Need community input

## Performance Considerations

### Benchmarking
```bash
# Run benchmarks
go test -bench=. -benchmem ./test > new.txt
go test -bench=. -benchmem ./test > old.txt
benchstat old.txt new.txt
```

### Profile Changes
- Check memory allocations
- Verify GC pressure impact
- Measure CPU impact
- Test with concurrent connections

### Optimization Guidelines
1. Measure first (don't optimize guesses)
2. Profile to find bottlenecks
3. Make minimal changes
4. Verify improvement with benchmarks
5. Document trade-offs

## Documentation

### Code Documentation
- All exported functions must have comments
- Comments should be clear and concise
- Use proper English with punctuation
- Explain why, not just what

### Project Documentation
- Update README.md for user-facing changes
- Update SECURITY.md for security implications
- Add examples for new features
- Keep documentation DRY

### Example
```go
// Server relays RTMP streams from clients to upstream server.
//
// It handles authentication, rate limiting, connection limiting,
// circuit breaking, and retry logic. All configuration options are
// optional except Upstream.
type Server struct {
    ListenAddr string
    Upstream   string
    // ... more fields ...
}
```

## Release Process

Releases follow semantic versioning (MAJOR.MINOR.PATCH):

1. Update version in code
2. Update CHANGELOG.md
3. Create git tag
4. Push to GitHub
5. Create GitHub release with notes
6. Docker image built and pushed automatically

## Questions?

- Open an issue for feature discussion
- Check existing issues/discussions first
- Ask in PR comments for clarification
- Email maintainers for sensitive topics

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

## Thank You!

We appreciate your contributions to making FFmpeg-Go-Relay better for everyone! 🎉

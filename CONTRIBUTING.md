# Contributing to GoDash

Thank you for your interest in contributing to GoDash! This document provides guidelines and information for contributors.

## 🚀 Getting Started

### Prerequisites

- Go 1.19 or higher
- Git
- Make (optional but recommended)

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/your-eyzaun/godash.git
   cd godash
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   # or using make
   make deps
   ```

3. **Build and Test**
   ```bash
   make build
   make test
   ```

4. **Run the Application**
   ```bash
   make run
   # or for CLI version
   make run-cli
   ```

## 🔧 Development Workflow

### Branch Naming

- `feature/description` - New features
- `bugfix/description` - Bug fixes
- `hotfix/description` - Critical fixes
- `chore/description` - Maintenance tasks
- `docs/description` - Documentation updates

### Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): description

body (optional)

footer (optional)
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `style` - Code formatting
- `refactor` - Code refactoring
- `test` - Adding tests
- `chore` - Maintenance tasks

**Examples:**
```
feat(collector): add network interface monitoring
fix(api): resolve memory leak in metrics collection
docs(readme): update installation instructions
test(collector): add unit tests for CPU metrics
```

### Code Standards

1. **Formatting**
   ```bash
   make fmt
   ```

2. **Linting**
   ```bash
   make lint
   ```

3. **Testing**
   ```bash
   make test
   make coverage
   ```

## 📝 Code Guidelines

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting
- Write clear, self-documenting code
- Add comments for exported functions and types
- Use meaningful variable and function names

### Package Structure

```
internal/
├── collector/     # System metrics collection
├── models/        # Data structures
├── utils/         # Utility functions
├── api/           # HTTP API handlers (future)
├── database/      # Database operations (future)
└── services/      # Business logic services (future)
```

### Error Handling

- Always handle errors explicitly
- Use wrapped errors with context: `fmt.Errorf("operation failed: %w", err)`
- Return errors from functions that can fail
- Log errors appropriately

### Testing

- Write unit tests for all public functions
- Use table-driven tests when appropriate
- Include benchmark tests for performance-critical code
- Aim for >80% test coverage
- Test both success and failure cases

**Example Test:**
```go
func TestFormatBytes(t *testing.T) {
    tests := []struct {
        input    uint64
        expected string
    }{
        {1024, "1.0 KB"},
        {1048576, "1.0 MB"},
    }

    for _, test := range tests {
        result := FormatBytes(test.input)
        if result != test.expected {
            t.Errorf("FormatBytes(%d) = %s; expected %s", 
                test.input, result, test.expected)
        }
    }
}
```

## 🐛 Bug Reports

When reporting bugs, please include:

1. **Environment Information**
   - Operating System
   - Go version
   - GoDash version

2. **Steps to Reproduce**
   - Clear, step-by-step instructions
   - Expected behavior
   - Actual behavior

3. **Additional Context**
   - Error messages
   - Log output
   - Screenshots (if applicable)

## 💡 Feature Requests

When suggesting features:

1. **Describe the Problem**
   - What problem does this solve?
   - Who would benefit from this feature?

2. **Proposed Solution**
   - How should it work?
   - Any implementation ideas?

3. **Alternatives Considered**
   - Other solutions you've considered
   - Why this approach is preferred

## 🔄 Pull Request Process

1. **Before Starting**
   - Check existing issues and PRs
   - Open an issue to discuss major changes
   - Fork the repository

2. **Development**
   - Create a feature branch
   - Write code following our guidelines
   - Add/update tests
   - Update documentation

3. **Before Submitting**
   ```bash
   make check  # Run all checks
   make test   # Ensure tests pass
   ```

4. **PR Description**
   - Clear title and description
   - Link related issues
   - Describe changes made
   - Include screenshots for UI changes

5. **Review Process**
   - Address reviewer feedback
   - Keep PR up to date with main branch
   - Squash commits if requested

## 🏗️ Project Structure

### Current (Week 1)
- ✅ System metrics collection
- ✅ CLI interface
- ✅ Cross-platform support

### Upcoming Weeks
- 🔄 Week 2: Web API + Database
- 🔄 Week 3: Real-time Dashboard
- 🔄 Week 4: Alerts + Production

## 🧪 Testing Guidelines

### Unit Tests
- Test individual functions/methods
- Mock external dependencies
- Focus on business logic

### Integration Tests
- Test component interactions
- Use real dependencies when appropriate
- Test error scenarios

### Performance Tests
- Benchmark critical paths
- Monitor memory usage
- Test with realistic data sizes

## 📚 Documentation

- Update README.md for user-facing changes
- Add code comments for complex logic
- Update API documentation
- Include examples in documentation

## 🎯 Areas for Contribution

### High Priority
- [ ] CPU temperature monitoring
- [ ] Process tree visualization
- [ ] Custom metric definitions
- [ ] Configuration file support

### Medium Priority
- [ ] Plugin system
- [ ] Custom dashboards
- [ ] Historical data analysis
- [ ] Performance optimizations

### Good First Issues
- [ ] Add more unit tests
- [ ] Improve error messages
- [ ] Add command-line flags
- [ ] Documentation improvements

## 🤝 Community

- Be respectful and inclusive
- Help newcomers
- Share knowledge and experience
- Follow our Code of Conduct

## 📞 Getting Help

- 💬 Open a [Discussion](https://github.com/eyzaun/godash/discussions)
- 🐛 Report bugs via [Issues](https://github.com/eyzaun/godash/issues)
- 📧 Contact maintainers directly

Thank you for contributing to GoDash! 🎉
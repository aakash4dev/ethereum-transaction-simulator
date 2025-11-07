# Contributing to Ethereum Transaction Simulator

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to this project.

## Code of Conduct

- Be respectful and considerate of others
- Welcome newcomers and help them learn
- Focus on constructive feedback
- Respect different viewpoints and experiences

## How to Contribute

### Reporting Issues

If you find a bug or have a feature request:

1. Check if the issue already exists in the [Issues](https://github.com/aakash4dev/ethereum-transaction-simulator/issues) page
2. Create a new issue with:
   - Clear title and description
   - Steps to reproduce (for bugs)
   - Expected vs actual behavior
   - Environment details (Go version, OS, etc.)

### Submitting Changes

1. **Fork the repository**
2. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes**:
   - Follow Go coding standards
   - Write clear, concise code
   - Add comments for complex logic
   - Keep functions focused and small
4. **Test your changes**:
   ```bash
   go build ./cmd/simulator
   go test ./...
   ```
5. **Commit your changes**:
   ```bash
   git commit -m "Add: description of your change"
   ```
   - Use clear, descriptive commit messages
   - Reference issue numbers if applicable (e.g., "Resolves #123")
6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```
7. **Create a Pull Request**:
   - Provide a clear description of changes
   - Reference related issues
   - Wait for review and feedback

## Development Guidelines

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use `golangci-lint` for linting
- Keep functions small and focused
- Use meaningful variable and function names
- Add comments for exported functions and types

### Project Structure

```
├── cmd/simulator/          # Main application entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── transaction/       # Transaction handling
│   ├── contract/         # Smart contract operations
│   └── wallet/           # Wallet management
├── scripts/              # Utility scripts
└── .env.example          # Configuration template
```

### Testing

- Write tests for new features
- Ensure existing tests pass
- Test with different EVM-compatible chains when possible

### Documentation

- Update README.md for user-facing changes
- Add comments for complex code
- Update .env.example if adding new configuration options

## Types of Contributions

### Bug Fixes
- Fix existing issues
- Improve error handling
- Add missing edge case handling

### Features
- New transaction modes
- Additional blockchain support
- Performance improvements
- Better configuration options

### Documentation
- Improve README clarity
- Add code examples
- Fix typos and errors
- Add troubleshooting guides

### Testing
- Add unit tests
- Improve test coverage
- Add integration tests

## Questions?

If you have questions about contributing:
- Open an issue with the `question` label
- Check existing issues and discussions

Thank you for contributing! 🚀


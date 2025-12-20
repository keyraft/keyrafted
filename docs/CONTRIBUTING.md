# Contributing to Keyraft

Thank you for your interest in contributing to Keyraft!

## Ways to Contribute

- **Report bugs** - Open an issue with details
- **Suggest features** - Open an issue with your idea
- **Improve documentation** - Fix typos, clarify instructions
- **Submit code** - Fix bugs or implement features

## Getting Started

### 1. Fork and Clone

```bash
git clone https://github.com/YOUR_USERNAME/keyrafted.git
cd keyrafted
```

### 2. Set Up Development Environment

```bash
# Install dependencies
go mod download

# Run tests
make test

# Build
make build
```

### 3. Make Changes

```bash
# Create branch
git checkout -b feature/your-feature-name

# Make changes and test
make test
make lint

# Commit
git commit -m "Add feature: your feature description"
```

### 4. Submit Pull Request

1. Push to your fork
2. Open PR against `main` branch
3. Describe your changes
4. Link any related issues

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Run `go vet` to catch common errors
- Use `make lint` to check code quality

### Testing

- Write tests for new features
- Maintain test coverage >80%
- Run `make test` before submitting PR
- Add integration tests for API changes

### Commit Messages

Use clear, descriptive commit messages:

```
Add feature: support for wildcard namespaces

- Implement wildcard matching in namespace resolver
- Add tests for wildcard patterns
- Update documentation

Fixes #123
```

### Pull Request Guidelines

- **Title**: Clear and descriptive
- **Description**: Explain what and why
- **Tests**: Include relevant tests
- **Documentation**: Update if needed
- **Breaking changes**: Clearly noted

## Code of Conduct

### Be Respectful

- Use welcoming and inclusive language
- Be respectful of differing viewpoints
- Accept constructive criticism gracefully
- Focus on what's best for the community

### Report Issues

Report unacceptable behavior to keyrafted@gmail.com

## Questions?

- Open an issue for questions
- Join discussions in GitHub Discussions
- Check existing issues and PRs first

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

---

Thank you for contributing to Keyraft! 🎉


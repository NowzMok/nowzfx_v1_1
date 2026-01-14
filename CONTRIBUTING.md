# Contributing to NowzFX

Thank you for your interest in contributing to NowzFX! We welcome contributions from everyone. This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/nowzfx_v1_1.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Push to your fork: `git push origin feature/your-feature-name`
6. Create a Pull Request

## Development Setup

### Prerequisites
- Go 1.25 or later
- Node.js 20 or later
- Docker and Docker Compose
- Git

### Local Development

1. Install dependencies:
   ```bash
   cd source
   go mod tidy
   cd ../scripts
   ./build_backend.sh
   ./build_frontend.sh
   ```

2. Configure environment variables:
   ```bash
   cp config/.env.example config/.env
   # Edit config/.env with your settings
   ```

3. Start the development environment:
   ```bash
   docker-compose -f docker-compose.yml up -d
   ```

4. Access the application:
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080

## Code Style

- **Go**: Follow [Effective Go](https://golang.org/doc/effective_go)
- **TypeScript/React**: Follow [Google TypeScript Style Guide](https://google.github.io/styleguide/tsconfig.json)
- Format code before committing:
  - Go: `go fmt ./...`
  - Node: `npm run format`

## Commit Messages

Use clear, descriptive commit messages:
- `feat: Add new feature`
- `fix: Fix bug in component`
- `docs: Update documentation`
- `test: Add test cases`
- `refactor: Refactor module`
- `perf: Improve performance`

## Pull Request Process

1. Ensure all tests pass: `npm test` and `go test ./...`
2. Update documentation if needed
3. Add a clear description of your changes
4. Link any related issues
5. Ensure your branch is up to date with main

## Reporting Issues

Please use the GitHub issue tracker to report bugs or suggest features. Include:
- Clear description of the issue
- Steps to reproduce (for bugs)
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Screenshots or logs if applicable

## Code Review

All submissions require review. We use GitHub pull requests for this purpose.
The maintainers will review your code for:
- Code quality and style
- Test coverage
- Documentation
- Performance implications
- Security considerations

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Questions?

Feel free to open an issue or discussion if you have any questions about contributing.

Thank you for contributing to NowzFX!

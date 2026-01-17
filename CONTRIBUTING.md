# Contributing to go-salesforce-emulator

Thank you for your interest in contributing to go-salesforce-emulator!

## Development Setup

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/YOUR_USERNAME/go-salesforce-emulator.git
   cd go-salesforce-emulator
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Run tests:
   ```bash
   make test
   ```

4. Build the binary:
   ```bash
   make build
   ```

## Code Style

- Follow standard Go conventions and idioms
- Run `make fmt` before committing to format code
- Run `make lint` to check for issues (requires [golangci-lint](https://golangci-lint.run/))

## Making Changes

1. Create a new branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes

3. Add tests for new functionality

4. Ensure all tests pass:
   ```bash
   make test
   ```

5. Commit your changes with a clear message:
   ```bash
   git commit -m "Add feature: description of your changes"
   ```

6. Push and create a pull request

## Pull Request Guidelines

- Provide a clear description of the changes
- Include any relevant issue numbers
- Ensure CI checks pass
- Keep changes focused - one feature or fix per PR

## Adding New Salesforce APIs

When adding support for a new Salesforce API:

1. Create a new package under `pkg/` if needed
2. Implement the handler following existing patterns
3. Register routes in `pkg/rest/router.go`
4. Add integration tests in `examples/integration_test/`
5. Update README.md with the new endpoint

## Reporting Issues

- Check existing issues first
- Provide clear reproduction steps
- Include Go version and OS information
- Share relevant error messages or logs

## Questions?

Feel free to open an issue for discussion.

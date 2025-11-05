## Contributing

We welcome contributions to the Tado Prometheus Exporter! This guide will help you get started.

### Before You Start

New to this project? Start with the **[ONBOARDING.md](ONBOARDING.md)** guide! It covers:
- Project architecture and design patterns
- How the codebase is organized
- Key concepts (OAuth2, graceful degradation, metric validation)
- Common development tasks (adding metrics, running tests, debugging)
- Troubleshooting guide

This guide focuses on the contribution workflow. For deep technical context, refer to ONBOARDING.md.

---

## Contribution Workflow

1. **Fork** the repository
2. **Create a feature branch**: `git checkout -b feature/my-feature` or `git checkout -b fix/my-bug`
3. **Make your changes** and write tests
4. **Run full check**:
   ```bash
   make check  # Runs: build + lint + test
   ```
5. **Commit with clear, descriptive messages** (see [Commit Messages](#commit-messages) below)
6. **Push to your fork** and **open a Pull Request**

### Commit Messages

Use imperative mood and be descriptive:
- ✅ Good: `Add window open/closed detection metric`
- ✅ Good: `Fix authentication token refresh race condition`
- ❌ Bad: `Updated stuff`
- ❌ Bad: `WIP`

---

## Development Setup

### Prerequisites
- **Go 1.24+** (required, see go.mod)
- **Make** (optional but recommended)
- **golangci-lint** (for linting checks)

### Quick Start

```bash
# 1. Clone and enter directory
git clone git@github.com:YOUR_USERNAME/tado-prometheus-exporter.git
cd tado-prometheus-exporter

# 2. Build the binary
make build

# 3. Run tests
make test

# 4. Run full checks before committing
make check
```

### Useful Make Targets

```bash
make build          # Build the binary
make test           # Run unit tests
make test-coverage  # Run tests with coverage report
make coverage       # Open HTML coverage report in browser
make lint           # Run golangci-lint
make check          # Full check (build + lint + test)
make run            # Build and run locally (requires TOKEN_PASSPHRASE)
make docker-build   # Build Docker image
make clean          # Remove build artifacts
```

---

## Development Guidelines

### Code Style & Standards

- Follow Go conventions (use `gofmt`, `go vet`)
- Linting: `golangci-lint run ./...`
- Error handling: Always wrap errors with context using `fmt.Errorf("context: %w", err)`
- Imports: Group in three sections (stdlib, external, internal)
- Keep functions small and focused
- Never log secrets (tokens, passphrases, credentials)

### Testing

- Write tests for new features and bug fixes
- Use table-driven tests for multiple test cases
- Use `testify` assertions (`require`, `assert`)
- Run with race detector: `go test -v -race ./...`
- Current coverage: ~80+ tests across all packages

See ONBOARDING.md's [Testing](#testing) section for detailed examples.

### Logging

Use structured logging with context:

```go
log.Info("Collecting metrics", "home_id", homeID, "zone_count", zoneCount)
log.Warn("Failed to fetch weather", "home_id", homeID, "error", err.Error())
```

**Never log**: tokens, passphrases, or any credentials.

---

## Common Contribution Scenarios

### Adding a New Tado Metric

See ONBOARDING.md's **[How To: Add a New Tado Metric](ONBOARDING.md#add-a-new-tado-metric)** section for step-by-step instructions.

Quick overview:
1. Define the metric in `pkg/metrics/metrics.go`
2. Register it in the `Register()` method
3. Collect data in `pkg/collector/collector.go`
4. Update `Describe()` and `Collect()` methods
5. Write tests in `pkg/collector/collector_test.go`
6. Update `docs/examples/dashboards/tado-exporter.json` if user-facing

### Adding Metric Validation

See ONBOARDING.md's **[How To: Add Metric Validation](ONBOARDING.md#add-metric-validation)** for detailed steps on ensuring data quality.

### Fixing Authentication Issues

Refer to ONBOARDING.md's **[Debug Authentication Issues](ONBOARDING.md#debug-authentication-issues)** section.

---

## Architecture Overview

For a detailed understanding of:
- **System design** and **data flow**
- **Key architectural patterns** (graceful degradation, metric validation, etc.)
- **Package responsibilities**
- **Design decisions**

See ONBOARDING.md's **[Architecture & Code Organization](ONBOARDING.md#architecture--code-organization)** section.

---

## Before Submitting a PR

Checklist for contributors:

- [ ] Code follows Go conventions and style guidelines
- [ ] Tests added/updated for new features
- [ ] All tests pass: `make test`
- [ ] Linting passes: `make lint`
- [ ] Build succeeds: `make build`
- [ ] Documentation updated if needed (README.md, ONBOARDING.md, etc.)
- [ ] Commit messages are clear and descriptive
- [ ] One feature/fix per PR (keep PRs focused)

Run this before pushing:
```bash
make check
```

---

## Getting Help

- **Project Architecture**: See [ONBOARDING.md](ONBOARDING.md)
- **Quick Reference**: ONBOARDING.md's [Quick Reference](#quick-reference) section
- **Troubleshooting**: ONBOARDING.md's [Troubleshooting](#troubleshooting) section
- **Issues**: Check existing GitHub issues before opening a new one
- **Discussions**: Use GitHub Discussions for questions

---

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
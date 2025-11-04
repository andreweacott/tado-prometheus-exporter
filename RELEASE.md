# Release Management Guide

Process for releasing new versions of tado-prometheus-exporter, including versioning strategy, release checklist, and automation.

## Table of Contents

1. [Versioning Strategy](#versioning-strategy)
2. [Release Process](#release-process)
3. [Pre-Release Checklist](#pre-release-checklist)
4. [Release Checklist](#release-checklist)
5. [Post-Release Checklist](#post-release-checklist)
6. [Automated Workflows](#automated-workflows)

---

## Versioning Strategy

This project follows **Semantic Versioning 2.0.0** (https://semver.org/).

### Version Format

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
```

Example: `v1.2.3-alpha.1+build.123`

### Version Components

- **MAJOR**: Incompatible API or configuration changes (rare)
  - Example: v1.0.0 → v2.0.0
  - Requires migration guide

- **MINOR**: New features, backwards-compatible
  - Example: v1.0.0 → v1.1.0
  - Users can upgrade without configuration changes

- **PATCH**: Bug fixes, maintenance
  - Example: v1.0.0 → v1.0.1
  - Should not change behavior, only fixes bugs

- **Pre-release** (optional): Pre-release versions for testing
  - Format: `-alpha`, `-beta`, `-rc` (e.g., `v1.1.0-beta.1`)
  - Example: v1.1.0-alpha.1, v1.1.0-rc.1

- **BUILD** (optional): Build metadata, doesn't affect version precedence
  - Not recommended for public releases

### Version Decision Examples

| Change | Version Jump | Example |
|--------|--------------|---------|
| Bug fix | PATCH | v1.0.0 → v1.0.1 |
| New metric | MINOR | v1.0.0 → v1.1.0 |
| New feature (auth) | MINOR | v1.0.0 → v1.1.0 |
| Config change | MAJOR | v1.0.0 → v2.0.0 |
| Performance improvement | PATCH or MINOR | v1.0.0 → v1.0.1 or v1.1.0 |
| New OS support | PATCH | v1.0.0 → v1.0.1 |
| Docker optimization | PATCH | v1.0.0 → v1.0.1 |

---

## Release Process

### Overview

```
git workflow                    GitHub Actions              Output
┌─────────────────────┐
│ 1. Create PR        │
│ 2. Get reviews      │
│ 3. Merge to main    │
│ 4. Tag release      │───────> test.yaml           ✓ Tests pass
│                     │
│                     │
│ 5. Push tag         │───────> build.yaml          • Build image
│    to remote        │                             • Push to Docker Hub
│                     │                             • Create GitHub release
└─────────────────────┘
```

### Step-by-Step Release Process

#### 1. Prepare Release

```bash
# Switch to main branch
git checkout main
git pull origin main

# Ensure tests pass locally
go test -v ./...

# Ensure build works
go build ./cmd/exporter
```

#### 2. Update Version

Update version references to new version (e.g., v1.1.0):

```bash
# Check where version might be hardcoded
grep -r "1.0.0" . --exclude-dir=.git

# Update if needed (usually only in docs/CHANGELOG)
# - Don't hardcode version in code (use git tags)
```

#### 3. Update CHANGELOG

Edit `CHANGELOG.md`:

```markdown
## [1.1.0] - 2025-11-15

### Added
- New feature description
- Another feature

### Changed
- Behavior change description

### Fixed
- Bug fix description

### Security
- Security issue fix

[1.1.0]: https://github.com/andreweacott/tado-prometheus-exporter/releases/tag/v1.1.0
```

#### 4. Commit and Tag

```bash
# Stage changes
git add CHANGELOG.md

# Commit
git commit -m "Release v1.1.0"

# Create annotated tag (GitHub Actions uses this)
git tag -a v1.1.0 -m "Release version 1.1.0"

# Show tag
git show v1.1.0
```

#### 5. Push to GitHub

```bash
# Push commit
git push origin main

# Push tag (triggers build workflow)
git push origin v1.1.0

# Or push both
git push origin main --tags
```

#### 6. Monitor GitHub Actions

Go to: https://github.com/andreweacott/tado-prometheus-exporter/actions

- **test.yaml**: Runs on push to main (should pass)
- **build.yaml**: Runs on new tag v* (builds and pushes image)

### Manual Release (if automation fails)

#### Build and Push Docker Image Manually

```bash
# Build locally
docker build -t andreweacott/tado-prometheus-exporter:v1.1.0 .
docker build -t andreweacott/tado-prometheus-exporter:latest .

# Login to Docker Hub
docker login -u andreweacott

# Push images
docker push andreweacott/tado-prometheus-exporter:v1.1.0
docker push andreweacott/tado-prometheus-exporter:latest

# Logout
docker logout
```

#### Create GitHub Release Manually

```bash
# Using GitHub CLI
gh release create v1.1.0 \
  --title "Release v1.1.0" \
  --notes-file CHANGELOG.md
```

---

## Pre-Release Checklist

Complete 1-2 weeks before release:

- [ ] Create feature branch for release preparation
- [ ] Review all open PRs and issues
- [ ] Decide MAJOR/MINOR/PATCH version number
- [ ] Plan changelog content
- [ ] Identify breaking changes (if any)
- [ ] Start discussion/RFC if major version

### Pre-Release Testing

- [ ] Build binary: `go build ./cmd/exporter`
- [ ] Run unit tests: `go test -v -race ./...`
- [ ] Run linter: `golangci-lint run`
- [ ] Build Docker image: `docker build -t test .`
- [ ] Test Docker image: `docker run test --help`
- [ ] Manual integration test (if applicable)
- [ ] Test docker-compose stack: `docker-compose up`
- [ ] Verify all documentation is accurate

### Breaking Changes

If releasing a MAJOR version with breaking changes:

1. Create migration guide
2. Update README with warnings
3. Add section to CHANGELOG explaining changes
4. Consider releasing as pre-release (rc1, rc2) first
5. Allow extra time for community feedback

---

## Release Checklist

Complete on release day:

### Code Preparation

- [ ] All PRs merged and reviewed
- [ ] Main branch is stable
- [ ] Local tests pass: `go test -v ./...`
- [ ] Linter passes: `golangci-lint run`
- [ ] Binary builds: `go build ./cmd/exporter`
- [ ] No uncommitted changes: `git status` (should be clean)

### Documentation

- [ ] CHANGELOG.md updated with all changes
- [ ] Version in CHANGELOG matches intended release
- [ ] README.md is current and accurate
- [ ] DEPLOYMENT.md reflects any configuration changes
- [ ] ARCHITECTURE.md reflects any design changes
- [ ] API documentation (HTTP_ENDPOINTS.md) is current
- [ ] All code comments are accurate

### Docker Image

- [ ] Dockerfile builds successfully
- [ ] Image is production-ready (minimal, secure)
- [ ] Image size is reasonable (<100MB)
- [ ] Image runs without errors
- [ ] Health check works

### Testing

- [ ] All unit tests pass
- [ ] No race conditions: `go test -race ./...`
- [ ] Code coverage acceptable: `go test -cover ./...`
- [ ] Integration tests pass (if applicable)
- [ ] Docker Compose stack starts: `docker-compose up -d`
- [ ] All endpoints respond:
  ```bash
  curl http://localhost:9100/health
  curl http://localhost:9100/metrics
  ```

### Release Creation

- [ ] Verify version format: `v1.x.x` (semver)
- [ ] Create git tag: `git tag -a v1.x.x -m "Release v1.x.x"`
- [ ] Verify tag created: `git tag -l v1.x.x`
- [ ] Push tag to GitHub: `git push origin v1.x.x`
- [ ] Verify GitHub Actions triggered:
  - Go to Actions page
  - See build.yaml running
  - Wait for build to complete

### Release Verification

- [ ] Docker image pushed to Docker Hub
- [ ] GitHub release created with CHANGELOG
- [ ] Release is marked as "Latest" (if not pre-release)
- [ ] GitHub release has:
  - [ ] Correct version tag (v1.x.x)
  - [ ] Descriptive title
  - [ ] Full CHANGELOG for this version
  - [ ] Pre-release checkbox (if applicable)

---

## Post-Release Checklist

Complete after successful release:

### Documentation

- [ ] Update repository homepage with new version
- [ ] Publish release notes to any documentation sites
- [ ] Send release announcement (if applicable)
- [ ] Update status page (if applicable)

### Communication

- [ ] Tag maintainers in release
- [ ] Notify stakeholders if applicable
- [ ] Post to relevant forums/communities (if applicable)
- [ ] Update any integrations that reference versions

### Monitoring

- [ ] Monitor for issues on new release
  - [ ] Check GitHub issues for bug reports
  - [ ] Monitor Docker Hub pull counts
  - [ ] Check usage metrics (if available)

### Next Steps

- [ ] Create milestone for next version
- [ ] Close v1.x.x milestone
- [ ] Plan features for next release
- [ ] Update long-term roadmap if needed

---

## Automated Workflows

### GitHub Actions: test.yaml

**Trigger**: Push to main/develop or PR to main/develop

**Steps**:
1. Checkout code
2. Set up Go 1.25
3. Download dependencies
4. Run linter (golangci-lint)
5. Run tests with race detection
6. Upload coverage to codecov

**Success Criteria**:
- All tests pass
- No race conditions
- No linting errors

### GitHub Actions: build.yaml

**Trigger**: Push tag matching `v*` (e.g., `v1.0.0`)

**Steps**:
1. Checkout code
2. Set up Docker Buildx
3. Extract version from tag
4. Login to Docker Hub (if credentials available)
5. Build multi-platform Docker image:
   - linux/amd64
   - linux/arm64
6. Push images with tags:
   - `<registry>/tado-prometheus-exporter:v1.x.x`
   - `<registry>/tado-prometheus-exporter:latest`
7. Create GitHub release with CHANGELOG.md

**Secrets Required** (set in GitHub repository settings):
- `DOCKER_USERNAME`: Docker Hub username
- `DOCKER_PASSWORD`: Docker Hub token or password
- `GITHUB_TOKEN`: (automatically provided by GitHub)

**Output**:
- Docker images pushed to registry
- GitHub release created
- Release notes populated from CHANGELOG.md

### Manual CI/CD

If GitHub Actions is unavailable:

```bash
# Run tests manually
go test -v -race -coverprofile=coverage.out ./...

# Run linter manually
golangci-lint run

# Build Docker image manually
docker build -t andreweacott/tado-prometheus-exporter:v1.x.x .

# Push to Docker Hub
docker login -u andreweacott
docker push andreweacott/tado-prometheus-exporter:v1.x.x
docker push andreweacott/tado-prometheus-exporter:latest
docker logout
```

---

## Version History

| Version | Release Date | Status | Notes |
|---------|--------------|--------|-------|
| v1.0.0 | 2025-11-04 | Latest | Initial release |
| v1.1.0 | TBD | Planned | Multi-zone improvements |
| v2.0.0 | TBD | Planned | Major refactor (TBD) |

---

## FAQ

### Q: How do I know what version to release?

A: Look at the changes since last release:
- Only bug fixes? → PATCH (v1.0.0 → v1.0.1)
- New features? → MINOR (v1.0.0 → v1.1.0)
- Breaking changes? → MAJOR (v1.0.0 → v2.0.0)

### Q: Can I release out-of-order versions?

A: No. Follow semver:
- v1.0.0 → v1.0.1 → v1.1.0 → v2.0.0

### Q: What if I release the wrong version?

A: You can delete and re-release:
```bash
# Delete tag locally and remotely
git tag -d v1.1.0
git push origin :refs/tags/v1.1.0

# Delete GitHub release in UI
# Delete Docker images:
# - Go to Docker Hub
# - Manage image tags
# - Delete incorrect versions

# Re-tag and release correctly
git tag -a v1.1.1 -m "Release v1.1.1"
git push origin v1.1.1
```

### Q: Can I have multiple releases in development?

A: Yes, use pre-release tags:
- `v1.1.0-rc.1` - Release candidate 1
- `v1.1.0-rc.2` - Release candidate 2
- `v1.1.0` - Final release

### Q: What about hotfixes?

A: Use PATCH version:
- v1.0.0 released
- Found critical bug
- Release v1.0.1 immediately
- Continue v1.1.0 work separately

### Q: How long should I support old versions?

A: Not yet defined. Consider:
- Active maintenance for: 2 releases back
- Security fixes for: 3 releases back
- EOL for: 4+ releases back

---

## Tools & Resources

- [Semantic Versioning](https://semver.org/) - Version specification
- [Keep a Changelog](https://keepachangelog.com/) - Changelog format
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github) - Release documentation
- [Docker Hub Repositories](https://docs.docker.com/docker-hub/) - Image publishing

---

## Related Documentation

- [CHANGELOG.md](CHANGELOG.md) - Version history
- [DEPLOYMENT.md](DEPLOYMENT.md) - Deployment instructions
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
- [README.md](README.md) - Quick start

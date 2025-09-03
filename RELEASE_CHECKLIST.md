# Release Checklist for gomdlint

> A comprehensive checklist ensuring high-quality, professional releases following semantic versioning and industry best practices.

## Pre-Release Planning

### 📋 **Step 1: Determine Release Type**

**Review changes since last release:**
```bash
# Check commit history since last tag
git log $(git describe --tags --abbrev=0)..HEAD --oneline

# Review CHANGELOG.md for unreleased changes
# Check for any breaking changes, new features, or bug fixes
```

**Choose version type according to [Semantic Versioning (SemVer)](https://semver.org/):**

| Change Type | Version Bump | Example | When to Use |
|-------------|--------------|---------|-------------|
| 🚨 **MAJOR** | `v0.2.3` → `v1.0.0` | Breaking API changes, removed features | Interface changes, CLI breaking changes |
| ✨ **MINOR** | `v0.2.3` → `v0.3.0` | New features, new rules, new commands | Plugin system, new CLI commands, new rules |
| 🔧 **PATCH** | `v0.2.3` → `v0.2.4` | Bug fixes, documentation, internal improvements | CI fixes, bug fixes, docs updates |

### 📊 **Step 2: Pre-Release Quality Gate**

**Verify CI is passing:**
```bash
# Check latest CI status
gh run list --limit 3

# Ensure all platforms passing:
# ✅ Windows, macOS, Ubuntu tests
# ✅ Code quality and security  
# ✅ Fuzz testing
# ✅ Binary builds
```

**Run comprehensive local tests:**
```bash
make test-cover     # Full test suite with coverage
make lint          # Code quality checks  
make security      # Security vulnerability scan
make build-all     # Cross-platform builds
```

**Check for regressions:**
```bash
# Test core functionality
./bin/gomdlint version
./bin/gomdlint lint README.md
./bin/gomdlint check docs/ --config .markdownlint.json
```

## Release Preparation

### 📝 **Step 3: Update Documentation**

**Update CHANGELOG.md:**
```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New features and enhancements
- New rules or commands

### Changed  
- Modified behavior (non-breaking)
- Performance improvements
- Updated dependencies

### Fixed
- Bug fixes and issue resolutions
- Security fixes

### Deprecated
- Features marked for removal in future versions

### Removed
- Deleted features or breaking changes

### Security
- Security-related changes and fixes
```

**Update version references:**
```bash
# Check all files that reference the version
grep -r "v0\." docs/ README.md --exclude-dir=.git
grep -r "1.24" . --exclude-dir=.git --exclude-dir=vendor | grep -v CHANGELOG.md

# Update:
# - README.md installation examples
# - docs/*.md version requirements  
# - go.mod Go version if changed
# - .github/workflows/ Go versions
```

**Verify documentation accuracy:**
- [ ] README.md installation instructions work
- [ ] API examples in README.md are current
- [ ] Configuration examples are valid
- [ ] All links are working (use link checker)

### 🏷️ **Step 4: Version Management**

**Update go.mod if needed:**
```bash
# Only if Go version requirement changed
go mod edit -go=1.23
go mod tidy
```

**Commit documentation changes:**
```bash
git add CHANGELOG.md README.md docs/ go.mod
git commit -m "docs: update documentation and changelog for vX.Y.Z release

- Update CHANGELOG.md with vX.Y.Z release notes
- Update version references in documentation  
- Verify all installation instructions and examples
- Update Go version requirement if changed"
```

## Release Execution

### 🚀 **Step 5: Create and Push Release**

**Create release tag:**
```bash
# Ensure working directory is clean
git status

# Create annotated tag with release notes summary
git tag -a v1.2.3 -m "Release v1.2.3 - Brief summary of major changes

Major highlights:
- Key feature 1
- Key improvement 2  
- Important fix 3

See CHANGELOG.md for full details."

# Push tag to trigger release pipeline
git push origin v1.2.3
```

**Monitor goreleaser execution:**
```bash
# Watch the release build
gh run watch

# Check release creation
gh release list --limit 3

# Verify release assets
gh release view v1.2.3
```

### 📦 **Step 6: Verify Release Assets**

**Check all platform binaries:**
```bash
# Download and test key binaries
mkdir -p /tmp/release-test
cd /tmp/release-test

# Test different architectures
wget https://github.com/jamesainslie/gomdlint/releases/download/v1.2.3/gomdlint_Linux_x86_64.tar.gz
tar -xzf gomdlint_Linux_x86_64.tar.gz
./gomdlint version

wget https://github.com/jamesainslie/gomdlint/releases/download/v1.2.3/gomdlint_Darwin_arm64.tar.gz  
tar -xzf gomdlint_Darwin_arm64.tar.gz
./gomdlint version
```

**Verify package managers:**
```bash
# Check Homebrew formula generation
gh api repos/jamesainslie/homebrew-gomdlint/contents/gomdlint.rb || echo "Formula not yet generated"

# Check Scoop manifest generation  
gh api repos/jamesainslie/scoop-gomdlint/contents/gomdlint.json || echo "Manifest not yet generated"

# Verify package manager installation (after propagation)
# brew install jamesainslie/gomdlint/gomdlint
# scoop bucket add gomdlint https://github.com/jamesainslie/scoop-gomdlint && scoop install gomdlint
```

**Verify checksums and signatures:**
```bash
# Download checksums
wget https://github.com/jamesainslie/gomdlint/releases/download/v1.2.3/gomdlint_v1.2.3_checksums.txt

# Verify a binary checksum
sha256sum gomdlint_Linux_x86_64.tar.gz
grep "gomdlint_Linux_x86_64.tar.gz" gomdlint_v1.2.3_checksums.txt
```

## Post-Release Validation

### ✅ **Step 7: Installation Testing**

**Test installation methods:**
```bash
# Test go install (most common method)
go install github.com/jamesainslie/gomdlint/cmd/gomdlint@v1.2.3
gomdlint version  # Should show v1.2.3

# Test binary download  
curl -L -o gomdlint https://github.com/jamesainslie/gomdlint/releases/download/v1.2.3/gomdlint_Linux_x86_64.tar.gz
tar -xzf gomdlint && ./gomdlint version

# Test source build with new tag
git clone --depth 1 --branch v1.2.3 https://github.com/jamesainslie/gomdlint.git test-build
cd test-build && make build && ./bin/gomdlint version
```

**Verify core functionality:**
```bash
# Test essential commands with new release
gomdlint lint README.md
gomdlint check docs/ 
gomdlint fix --dry-run test.md
gomdlint config show
gomdlint rules list
```

### 📢 **Step 8: Release Communication**

**Update project visibility:**
- [ ] Update GitHub repository topics and description if needed
- [ ] Update any badges in README.md if version format changed
- [ ] Star/watch the release to boost visibility

**Monitor release adoption:**
```bash
# Check download stats after 24-48 hours
gh api repos/jamesainslie/gomdlint/releases/latest

# Monitor for user issues
gh issue list --label="bug" --state=open
gh issue list --label="installation" --state=open
```

## Quality Assurance Checklist

### 🔍 **Step 9: Post-Release Verification**

**CI Badge Status:**
- [ ] ✅ Build Status badge shows passing
- [ ] ✅ Go Report Card score is good (A- or higher)  
- [ ] ✅ Code coverage badge shows appropriate percentage
- [ ] ✅ Release badge shows latest version
- [ ] ✅ All badges point to correct repository

**Release Artifacts Completeness:**
- [ ] ✅ Source code archives (`.tar.gz`, `.zip`)
- [ ] ✅ All platform binaries (Linux, macOS, Windows × AMD64/ARM64)
- [ ] ✅ Package manager manifests (Homebrew formula, Scoop manifest)
- [ ] ✅ Checksums file with SHA256 hashes
- [ ] ✅ Debian/RPM/Alpine packages
- [ ] ✅ Release notes are comprehensive

**Functional Verification:**
- [ ] ✅ Binary execution on all platforms
- [ ] ✅ Core linting functionality works
- [ ] ✅ Configuration loading works  
- [ ] ✅ Plugin system functional (if applicable)
- [ ] ✅ Help and version commands work
- [ ] ✅ Exit codes behave correctly

### 🐛 **Step 10: Issue Response Plan**

**Monitor for 48-72 hours after release:**
- [ ] Watch GitHub issues for installation problems
- [ ] Monitor GitHub Discussions for user questions
- [ ] Check package manager feedback (Homebrew/Scoop)
- [ ] Review any CI failures in downstream projects

**Prepare hotfix process:**
- [ ] Have plan for quick patch releases if critical bugs found
- [ ] Keep main branch in releasable state
- [ ] Monitor dependency updates and security advisories

## Release Automation

### 🔧 **Tools and Commands Reference**

**Key commands used in release process:**
```bash
# Development and testing
make test-cover lint security build-all

# Git operations
git log $(git describe --tags --abbrev=0)..HEAD --oneline
git tag -a v1.2.3 -m "Release message"
git push origin v1.2.3

# GitHub CLI operations  
gh run watch
gh release list --limit 3
gh release view v1.2.3
gh api repos/jamesainslie/gomdlint/releases/latest

# Package manager verification
brew install jamesainslie/gomdlint/gomdlint
scoop bucket add gomdlint https://github.com/jamesainslie/scoop-gomdlint
```

**Goreleaser validation:**
```bash
# Test release configuration before tagging
goreleaser release --snapshot --clean --skip-publish

# Check goreleaser configuration
goreleaser check
```

## Emergency Procedures

### 🚨 **Rollback Plan**

**If major issues found after release:**

1. **Delete problematic tag:**
   ```bash
   git tag -d v1.2.3
   git push origin :refs/tags/v1.2.3
   ```

2. **Delete GitHub release:**
   ```bash
   gh release delete v1.2.3 --yes
   ```

3. **Issue hotfix or revert:**
   ```bash
   # For critical bugs: create patch release v1.2.4
   # For broken releases: revert to previous working version
   ```

### 📋 **Release Success Criteria**

**✅ Release is considered successful when:**
- [ ] CI shows all green badges
- [ ] All platform binaries download and execute correctly
- [ ] `go install` works with new version  
- [ ] No critical user-reported issues within 48 hours
- [ ] Package managers receive formulas/manifests successfully
- [ ] Documentation is accurate and examples work
- [ ] Previous functionality remains intact (no regressions)

---

## Version History Reference

**Current release:** v0.2.3  
**Go version:** 1.23+  
**Release cadence:** As needed (feature-driven)  
**Support policy:** Latest release + previous major version

**Semantic versioning guide for gomdlint:**
- **0.x.y**: Pre-1.0 development (current state)
- **1.0.0**: First stable release with complete API
- **1.x.y**: Stable releases with backwards compatibility
- **2.0.0**: Next major version with breaking changes

---

*This checklist should be followed for every release to ensure consistent, high-quality releases that users can depend on.*

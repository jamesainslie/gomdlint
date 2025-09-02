#!/bin/bash
# Script to determine version based on GEICO-compliant conventional commits
# Supports both single-repo and monorepo versioning strategies

set -euo pipefail

SERVICE="${1:-gomdlint}"
BUILD_TYPE="${2:-release}"  # release or dev

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Determining version for service: ${YELLOW}$SERVICE${NC}" >&2
echo -e "${BLUE}Build type: ${YELLOW}$BUILD_TYPE${NC}" >&2

# Get current version from git tags (supports both v* and service/v* patterns)
get_current_version() {
    local service="$1"
    local version
    
    # Try monorepo pattern first: service/v*
    version=$(git tag -l "${service}/v*" 2>/dev/null | sort -V | tail -1 | sed "s|${service}/v||" || echo "")
    
    # If no monorepo tags, try single repo pattern: v*
    if [ -z "$version" ]; then
        version=$(git tag -l "v*" 2>/dev/null | sort -V | tail -1 | sed 's/v//' || echo "")
    fi
    
    # Default to 0.0.0 if no tags found
    if [ -z "$version" ]; then
        version="0.0.0"
    fi
    
    echo "$version"
}

# Get current version
CURRENT_VERSION=$(get_current_version "$SERVICE")
echo -e "${BLUE}Current version: ${YELLOW}$CURRENT_VERSION${NC}" >&2

if [[ "$BUILD_TYPE" == "dev" ]]; then
    # Development build - use short SHA
    SHORT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    VERSION="${CURRENT_VERSION}-dev.${SHORT_SHA}"
    SHOULD_TAG="false"
    echo -e "${YELLOW}Development build version: $VERSION${NC}" >&2
else
    # Production release - analyze commits for version bump
    
    # Determine the commit range to analyze
    LAST_TAG=""
    if [ "$CURRENT_VERSION" != "0.0.0" ]; then
        # Check for monorepo tag first
        if git rev-parse "${SERVICE}/v${CURRENT_VERSION}" >/dev/null 2>&1; then
            LAST_TAG="${SERVICE}/v${CURRENT_VERSION}"
        elif git rev-parse "v${CURRENT_VERSION}" >/dev/null 2>&1; then
            LAST_TAG="v${CURRENT_VERSION}"
        fi
    fi
    
    if [ -n "$LAST_TAG" ]; then
        COMMIT_RANGE="${LAST_TAG}..HEAD"
        echo -e "${BLUE}Analyzing commits since: ${YELLOW}$LAST_TAG${NC}" >&2
    else
        COMMIT_RANGE="HEAD"
        echo -e "${BLUE}No previous tags found, analyzing all commits${NC}" >&2
    fi
    
    # Get commit messages since last tag
    COMMITS=$(git log "$COMMIT_RANGE" --pretty=format:"%s" --grep="^feat\|^fix\|^BREAKING CHANGE\|^perf\|^refactor\|!" 2>/dev/null || echo "")
    
    # Analyze commits for version bump
    MAJOR_BUMP=false
    MINOR_BUMP=false
    PATCH_BUMP=false
    
    echo -e "${BLUE}Analyzing commit messages for version bump...${NC}" >&2
    
    while IFS= read -r commit; do
        [[ -z "$commit" ]] && continue
        
        echo -e "${BLUE}  Checking: ${NC}$commit" >&2
        
        # Check for breaking changes (major bump)
        if [[ "$commit" =~ BREAKING\ CHANGE ]] || [[ "$commit" =~ ^[^:]+!: ]]; then
            echo -e "${RED}    → Breaking change detected${NC}" >&2
            MAJOR_BUMP=true
        # Check for new features (minor bump)
        elif [[ "$commit" =~ ^feat ]]; then
            echo -e "${GREEN}    → Feature detected${NC}" >&2
            MINOR_BUMP=true
        # Check for fixes and improvements (patch bump)
        elif [[ "$commit" =~ ^fix ]] || [[ "$commit" =~ ^perf ]] || [[ "$commit" =~ ^refactor ]]; then
            echo -e "${YELLOW}    → Fix/improvement detected${NC}" >&2
            PATCH_BUMP=true
        fi
    done <<< "$COMMITS"
    
    # Calculate new version
    IFS='.' read -ra VERSION_PARTS <<< "$CURRENT_VERSION"
    MAJOR="${VERSION_PARTS[0]:-0}"
    MINOR="${VERSION_PARTS[1]:-0}"
    PATCH="${VERSION_PARTS[2]:-0}"
    
    # Apply version bumps
    if [[ "$MAJOR_BUMP" == "true" ]]; then
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        echo -e "${RED}Major version bump: ${YELLOW}${CURRENT_VERSION} → ${MAJOR}.${MINOR}.${PATCH}${NC}" >&2
        SHOULD_TAG="true"
    elif [[ "$MINOR_BUMP" == "true" ]]; then
        MINOR=$((MINOR + 1))
        PATCH=0
        echo -e "${GREEN}Minor version bump: ${YELLOW}${CURRENT_VERSION} → ${MAJOR}.${MINOR}.${PATCH}${NC}" >&2
        SHOULD_TAG="true"
    elif [[ "$PATCH_BUMP" == "true" ]]; then
        PATCH=$((PATCH + 1))
        echo -e "${YELLOW}Patch version bump: ${YELLOW}${CURRENT_VERSION} → ${MAJOR}.${MINOR}.${PATCH}${NC}" >&2
        SHOULD_TAG="true"
    else
        # No version-changing commits found
        echo -e "${BLUE}No version-changing commits found${NC}" >&2
        VERSION="$CURRENT_VERSION"
        SHOULD_TAG="false"
    fi
    
    if [[ "$SHOULD_TAG" == "true" ]]; then
        VERSION="${MAJOR}.${MINOR}.${PATCH}"
    fi
fi

# GEICO-specific version metadata
GEICO_ENV="${GEICO_ENV:-local}"
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_SHA=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

echo -e "${GREEN}Final version determination:${NC}" >&2
echo -e "  Service: ${YELLOW}$SERVICE${NC}" >&2
echo -e "  Version: ${YELLOW}$VERSION${NC}" >&2
echo -e "  Should Tag: ${YELLOW}$SHOULD_TAG${NC}" >&2
echo -e "  GEICO Environment: ${YELLOW}$GEICO_ENV${NC}" >&2
echo -e "  Build Date: ${YELLOW}$BUILD_DATE${NC}" >&2
echo -e "  Commit: ${YELLOW}${COMMIT_SHA:0:8}${NC}" >&2

# Output for CI systems
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    echo "version=$VERSION" >> "$GITHUB_OUTPUT"
    echo "should_tag=$SHOULD_TAG" >> "$GITHUB_OUTPUT"
    echo "service=$SERVICE" >> "$GITHUB_OUTPUT"
    echo "geico_env=$GEICO_ENV" >> "$GITHUB_OUTPUT"
    echo "build_date=$BUILD_DATE" >> "$GITHUB_OUTPUT"
    echo "commit=$COMMIT_SHA" >> "$GITHUB_OUTPUT"
    echo -e "${GREEN}GitHub Actions outputs set${NC}" >&2
elif [[ -n "${AZURE_HTTP_USER_AGENT:-}" ]]; then
    echo "##vso[task.setvariable variable=Version;isOutput=true]$VERSION"
    echo "##vso[task.setvariable variable=ShouldTag;isOutput=true]$SHOULD_TAG"
    echo "##vso[task.setvariable variable=Service;isOutput=true]$SERVICE"
    echo "##vso[task.setvariable variable=GeicoEnv;isOutput=true]$GEICO_ENV"
    echo "##vso[task.setvariable variable=BuildDate;isOutput=true]$BUILD_DATE"
    echo "##vso[task.setvariable variable=Commit;isOutput=true]$COMMIT_SHA"
    echo -e "${GREEN}Azure DevOps variables set${NC}" >&2
fi

# Output the version (this is what calling scripts will capture)
echo "$VERSION"

echo -e "${GREEN}Version determination completed successfully${NC}" >&2

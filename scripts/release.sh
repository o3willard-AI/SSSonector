#!/bin/bash
set -e

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

# Validate version format
if ! echo "$VERSION" | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' > /dev/null; then
    echo "Error: Version must be in format X.Y.Z"
    exit 1
fi

echo "Creating release v$VERSION..."

# Ensure we're in the repository root
cd "$(dirname "$0")/.."
REPO_ROOT=$(pwd)

# Ensure working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: Working directory is not clean. Please commit or stash changes."
    exit 1
fi

# Update version in files
echo "Updating version numbers..."
sed -i "s/VERSION := .*/VERSION := $VERSION/" Makefile
sed -i "s/version = \".*\"/version = \"$VERSION\"/" cmd/tunnel/main.go
sed -i "s/Version: .*/Version: $VERSION/" installers/linux/DEBIAN/control
sed -i "s/version=\".*\"/version=\"$VERSION\"/" installers/macos/distribution.xml

# Create release branch
BRANCH="release/v$VERSION"
git checkout -b "$BRANCH"

# Commit version updates
git add Makefile cmd/tunnel/main.go installers/linux/DEBIAN/control installers/macos/distribution.xml
git commit -m "Release v$VERSION"

# Create release tag
git tag -a "v$VERSION" -m "Release v$VERSION"

# Build all packages
echo "Building release packages..."
make clean
make build
make package

# Create release directory
RELEASE_DIR="dist/v$VERSION"
mkdir -p "$RELEASE_DIR"

# Move packages to release directory
mv build/*.deb "$RELEASE_DIR/"
mv build/*.rpm "$RELEASE_DIR/"
mv build/*.pkg "$RELEASE_DIR/"
mv build/*.exe "$RELEASE_DIR/"
mv build/checksums.txt "$RELEASE_DIR/"

# Generate release notes template
cat > "$RELEASE_DIR/RELEASE_NOTES.md" << EOF
# SSSonector v$VERSION

## Changes

- [List major changes here]

## Installation

### Linux (Debian/Ubuntu)
\`\`\`bash
wget https://github.com/o3willard-AI/SSSonector/dist/v$VERSION/sssonector_${VERSION}_amd64.deb
sudo dpkg -i sssonector_${VERSION}_amd64.deb
sudo apt-get install -f
\`\`\`

### macOS
\`\`\`bash
curl -LO https://github.com/o3willard-AI/SSSonector/dist/v$VERSION/sssonector-${VERSION}.pkg
sudo installer -pkg sssonector-${VERSION}.pkg -target /
\`\`\`

### Windows
1. Download sssonector-${VERSION}-setup.exe from dist/v$VERSION/
2. Run the installer with administrator privileges

## Checksums

\`\`\`
$(cat "$RELEASE_DIR/checksums.txt")
\`\`\`

## Documentation

Full documentation is available at: https://github.com/o3willard-AI/SSSonector/tree/v$VERSION/docs
EOF

echo
echo "Release v$VERSION prepared!"
echo
echo "Next steps:"
echo "1. Review and update $RELEASE_DIR/RELEASE_NOTES.md"
echo "2. Push the release branch: git push origin $BRANCH"
echo "3. Create a pull request for the release"
echo "4. After merge, push the tag: git push origin v$VERSION"
echo "5. Create a GitHub release with the contents of RELEASE_NOTES.md"
echo "6. Upload the packages from $RELEASE_DIR"
echo
echo "Release artifacts are in: $RELEASE_DIR"

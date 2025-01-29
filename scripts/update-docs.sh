#!/bin/bash
set -e

# Configuration
DOCS_DIR="docs"
MAIN_BRANCH="main"

# Ensure we're in the git repository
if [ ! -d ".git" ]; then
    echo "Error: Not in git repository root directory"
    exit 1
fi

# Check if there are uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo "Warning: You have uncommitted changes"
    read -p "Do you want to continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Update documentation files
echo "Updating documentation files..."

# Create docs directory if it doesn't exist
mkdir -p "$DOCS_DIR"

# List of documentation files to update
FILES=(
    "README.md"
    "docs/ubuntu_install.md"
    "docs/linux_install.md"
    "docs/macos_install.md"
    "docs/windows_install.md"
    "docs/qa_guide.md"
)

# Check if all files exist
for file in "${FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "Error: $file not found"
        exit 1
    fi
done

# Create git commit
echo "Creating git commit..."
git add "${FILES[@]}"
git commit -m "docs: Update installation guides and documentation

- Update README.md with correct download links
- Add Ubuntu installation guide
- Add Red Hat/Rocky Linux installation guide
- Add macOS installation guide
- Add Windows installation guide
- Update QA guide with platform-specific tests"

# Push changes
echo "Pushing changes to remote repository..."
git push origin "$MAIN_BRANCH"

echo "Documentation update complete!"
echo
echo "Next steps:"
echo "1. Verify the changes on GitHub"
echo "2. Update release notes if needed"
echo "3. Update documentation links in the repository settings"

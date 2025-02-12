#!/bin/bash
set -e

BUILD_DIR="build"
VERSION=$(cat "$BUILD_DIR/version.txt" | head -n 1)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo "Verifying SSSonector $VERSION builds..."

# Expected platforms and types
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
)

TYPES=(
    "admin"
    "benchmark"
)

# Function to verify SHA256 checksum
verify_checksum() {
    local file=$1
    local checksum_file="$file.sha256"
    
    if [ ! -f "$checksum_file" ]; then
        echo -e "${RED}❌ Missing checksum file for $file${NC}"
        return 1
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        if ! sha256sum -c "$checksum_file" &>/dev/null; then
            echo -e "${RED}❌ Checksum verification failed for $file${NC}"
            return 1
        fi
    elif command -v shasum >/dev/null 2>&1; then
        if ! shasum -a 256 -c "$checksum_file" &>/dev/null; then
            echo -e "${RED}❌ Checksum verification failed for $file${NC}"
            return 1
        fi
    else
        echo -e "${RED}❌ No checksum tool found (sha256sum or shasum)${NC}"
        return 1
    fi
    
    return 0
}

# Function to verify binary
verify_binary() {
    local file=$1
    local type=$2
    local platform=$3
    
    # Check file exists
    if [ ! -f "$file" ]; then
        echo -e "${RED}❌ Missing binary: $file${NC}"
        return 1
    fi
    
    # Verify size is reasonable (> 1MB)
    local size=$(wc -c < "$file")
    if [ "$size" -lt 1000000 ]; then
        echo -e "${RED}❌ Binary too small ($size bytes): $file${NC}"
        return 1
    fi
    
    # Verify checksum
    if ! verify_checksum "$file"; then
        return 1
    fi
    
    # Verify file type
    if [[ "$platform" == *"windows"* ]]; then
        if ! file "$file" | grep -q "PE32"; then
            echo -e "${RED}❌ Invalid Windows executable: $file${NC}"
            return 1
        fi
    elif [[ "$platform" == *"darwin"* ]]; then
        if ! file "$file" | grep -q "Mach-O"; then
            echo -e "${RED}❌ Invalid macOS executable: $file${NC}"
            return 1
        fi
    else
        if ! file "$file" | grep -q "ELF"; then
            echo -e "${RED}❌ Invalid Linux executable: $file${NC}"
            return 1
        fi
    fi
    
    echo -e "${GREEN}✓ Verified $type for $platform${NC}"
    return 0
}

# Verify version file
if [ ! -f "$BUILD_DIR/version.txt" ]; then
    echo -e "${RED}❌ Missing version.txt${NC}"
    exit 1
fi

# Track failures
failures=0

# Verify all builds
for platform in "${PLATFORMS[@]}"; do
    # Split platform into OS and ARCH
    IFS="/" read -r -a platform_split <<< "$platform"
    GOOS="${platform_split[0]}"
    GOARCH="${platform_split[1]}"
    
    for type in "${TYPES[@]}"; do
        binary="$BUILD_DIR/ssonector-$type-$VERSION-$GOOS-$GOARCH"
        if [ "$GOOS" = "windows" ]; then
            binary="$binary.exe"
        fi
        
        if ! verify_binary "$binary" "$type" "$platform"; then
            ((failures++))
        fi
    done
    
    # Verify archive
    archive="$BUILD_DIR/ssonector-$VERSION-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        if [ ! -f "$archive.zip" ]; then
            echo -e "${RED}❌ Missing archive: $archive.zip${NC}"
            ((failures++))
        fi
    else
        if [ ! -f "$archive.tar.gz" ]; then
            echo -e "${RED}❌ Missing archive: $archive.tar.gz${NC}"
            ((failures++))
        fi
    fi
done

echo "-------------------"
if [ "$failures" -eq 0 ]; then
    echo -e "${GREEN}✓ All builds verified successfully${NC}"
    exit 0
else
    echo -e "${RED}❌ $failures verification failures${NC}"
    exit 1
fi

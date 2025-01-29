#!/bin/bash
set -e

# Build configuration
VERSION="1.0.0"
PLATFORMS=("linux/amd64" "darwin/amd64" "windows/amd64")
OUTPUT_DIR="dist"

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    # Split platform into OS and architecture
    IFS="/" read -r -a parts <<< "$platform"
    GOOS="${parts[0]}"
    GOARCH="${parts[1]}"
    
    # Set output binary name based on platform
    if [ "$GOOS" = "windows" ]; then
        binary_name="SSSonector.exe"
    else
        binary_name="SSSonector"
    fi

    echo "Building for $GOOS/$GOARCH..."
    
    # Create platform-specific directory
    platform_dir="$OUTPUT_DIR/$GOOS-$GOARCH"
    mkdir -p "$platform_dir"
    
    # Build binary
    GOOS=$GOOS GOARCH=$GOARCH go build -v \
        -ldflags "-X main.Version=$VERSION" \
        -o "$platform_dir/$binary_name" \
        cmd/tunnel/main.go

    # Copy configuration files
    cp -r configs "$platform_dir/"
    cp -r mibs "$platform_dir/"
    
    # Copy platform-specific service files
    case $GOOS in
        "linux")
            mkdir -p "$platform_dir/service"
            cp scripts/service/systemd/SSSonector.service "$platform_dir/service/"
            ;;
        "darwin")
            mkdir -p "$platform_dir/service"
            cp scripts/service/launchd/com.SSSonector.plist "$platform_dir/service/"
            ;;
        "windows")
            mkdir -p "$platform_dir/service"
            cp scripts/service/windows/install-service.ps1 "$platform_dir/service/"
            ;;
    esac

    # Create documentation directory
    mkdir -p "$platform_dir/docs"
    cp docs/qa_guide.md "$platform_dir/docs/"
    
    # Create platform-specific installation guide
    cat > "$platform_dir/docs/INSTALL.md" << EOF
# SSSonector Installation Guide for $(echo "$GOOS" | tr '[:lower:]' '[:upper:]')

## Prerequisites

$(case $GOOS in
    "linux")
        echo "- Linux kernel with TUN/TAP support
- Root privileges for TUN device creation
- systemd for service management (optional)"
        ;;
    "darwin")
        echo "- macOS 10.15 or later
- Root privileges for TUN device creation
- Command Line Tools package"
        ;;
    "windows")
        echo "- Windows 10 or later
- Administrator privileges
- TAP-Windows Adapter V9 driver"
        ;;
esac)

## Installation Steps

1. Extract the archive to your preferred location:
   \`\`\`bash
   tar xzf SSSonector-$GOOS-$GOARCH.tar.gz
   cd SSSonector
   \`\`\`

2. Configure the application:
   - Edit \`configs/server.yaml\` or \`configs/client.yaml\` based on your needs
   - Ensure proper permissions on configuration files

3. Install as a service (optional):
$(case $GOOS in
    "linux")
        echo "   \`\`\`bash
   sudo cp service/SSSonector.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable SSSonector
   sudo systemctl start SSSonector
   \`\`\`"
        ;;
    "darwin")
        echo "   \`\`\`bash
   sudo cp service/com.SSSonector.plist /Library/LaunchDaemons/
   sudo launchctl load /Library/LaunchDaemons/com.SSSonector.plist
   \`\`\`"
        ;;
    "windows")
        echo "   \`\`\`powershell
   # Run as Administrator
   ./service/install-service.ps1
   Start-Service SSLTunnel
   \`\`\`"
        ;;
esac)

4. Verify installation:
   \`\`\`bash
   ./SSSonector --version
   \`\`\`

## Monitoring Setup

1. Install monitoring tools:
   \`\`\`bash
   cd monitoring
   docker-compose up -d
   \`\`\`

2. Access monitoring interfaces:
   - Grafana: http://localhost:3000
   - Prometheus: http://localhost:9090

## Troubleshooting

See \`docs/qa_guide.md\` for detailed troubleshooting steps.

## Support

For issues and support, please visit:
https://github.com/yourusername/SSSonector/issues
EOF

    # Create archive
    echo "Creating archive for $GOOS/$GOARCH..."
    cd "$OUTPUT_DIR"
    if [ "$GOOS" = "windows" ]; then
        zip -r "SSSonector-$GOOS-$GOARCH.zip" "$GOOS-$GOARCH"
    else
        tar czf "SSSonector-$GOOS-$GOARCH.tar.gz" "$GOOS-$GOARCH"
    fi
    cd ..
done

echo "Build complete! Archives available in $OUTPUT_DIR"

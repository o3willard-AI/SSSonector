# Building SSSonector for macOS

This guide provides instructions for building the SSSonector macOS package (.pkg) installer. Since package creation requires macOS-specific tools, these steps must be performed on a macOS system.

## Prerequisites

1. Install Xcode Command Line Tools:
```bash
xcode-select --install
```

2. Install Go 1.21 or later:
```bash
brew install go
```

3. Clone the repository:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
```

## Building the Package

1. Build the binary:
```bash
GOOS=darwin GOARCH=amd64 go build -o build/sssonector-darwin-amd64 ./cmd/tunnel
```

2. Create package structure:
```bash
mkdir -p build/macos/root/usr/local/bin
cp build/sssonector-darwin-amd64 build/macos/root/usr/local/bin/sssonector
chmod 755 build/macos/root/usr/local/bin/sssonector
```

3. Create the installer package:
```bash
pkgbuild --root build/macos/root \
         --identifier com.o3willard-ai.sssonector \
         --version 1.0.0 \
         --install-location / \
         build/sssonector-1.0.0.pkg
```

4. Generate checksum:
```bash
cd build
sha256sum sssonector-1.0.0.pkg >> checksums.txt
```

## Contributing Back

After building the package:

1. Fork the repository on GitHub
2. Create a new branch:
```bash
git checkout -b add-macos-package
```

3. Copy the package and updated checksums to the dist directory:
```bash
cp build/sssonector-1.0.0.pkg dist/v1.0.0/
cp build/checksums.txt dist/v1.0.0/
```

4. Commit and push your changes:
```bash
git add dist/v1.0.0/sssonector-1.0.0.pkg dist/v1.0.0/checksums.txt
git commit -m "Add macOS package for v1.0.0"
git push origin add-macos-package
```

5. Create a Pull Request on GitHub

## Package Contents

The macOS package installs:
- `/usr/local/bin/sssonector` - Main executable
- System service integration (launchd plist)
- Required certificates and configuration files

## Verification

After installation, verify the package:

1. Check binary installation:
```bash
which sssonector
sssonector --version
```

2. Verify service installation:
```bash
launchctl list | grep sssonector
```

## Troubleshooting

Common issues and solutions:

1. **Code Signing Issues**
   - Ensure Xcode Command Line Tools are installed
   - Use `codesign -vvv` to verify signatures

2. **Permission Issues**
   - Check file permissions in build/macos/root
   - Ensure all files are owned by root:wheel

3. **Installation Failures**
   - Check system.log for installation errors
   - Verify package structure with `pkgutil --expand`

For additional help, please create an issue on GitHub with the following information:
- macOS version
- Xcode version
- Go version
- Full build output
- Any relevant error messages

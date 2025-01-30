# macOS Build Guide for Contributors

This guide provides instructions for macOS contributors to build and submit the macOS installer package for SSSonector.

## Prerequisites

1. Install Xcode Command Line Tools:
```bash
xcode-select --install
```

2. Install Homebrew (if not already installed):
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

3. Install Go:
```bash
brew install go
```

## Building the macOS Installer

1. Clone the repository:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
```

2. Build the project:
```bash
# The build script will automatically detect macOS and build the .pkg installer
./scripts/build-installers.sh
```

The macOS installer will be created at `build/sssonector-1.0.0.pkg`

## Testing the Build

1. Install the package:
```bash
sudo installer -pkg build/sssonector-1.0.0.pkg -target /
```

2. Verify the installation:
```bash
# Check binary
ls -l /usr/local/bin/sssonector

# Check configuration files
ls -l /etc/sssonector/

# Check LaunchDaemon
ls -l /Library/LaunchDaemons/com.o3willard.sssonector.plist
```

3. Test the service:
```bash
# Start the service
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist

# Check status
sudo launchctl list | grep sssonector

# Stop the service
sudo launchctl unload /Library/LaunchDaemons/com.o3willard.sssonector.plist
```

## Submitting the Build

1. Create a new branch:
```bash
git checkout -b macos-installer
```

2. Add the built installer:
```bash
git add build/sssonector-1.0.0.pkg
```

3. Commit your changes:
```bash
git commit -m "Add macOS installer package"
```

4. Push to GitHub:
```bash
git push origin macos-installer
```

5. Create a Pull Request:
   - Go to https://github.com/o3willard-AI/SSSonector
   - Click "New Pull Request"
   - Select your `macos-installer` branch
   - Add description of your changes
   - Submit the PR

## Notes

- The installer package is built using `pkgbuild`
- Installation paths follow macOS conventions:
  - Binary: `/usr/local/bin/sssonector`
  - Config: `/etc/sssonector/`
  - LaunchDaemon: `/Library/LaunchDaemons/com.o3willard.sssonector.plist`
  - Logs: `/var/log/sssonector/`
- The package includes proper postinstall scripts for permissions and service setup
- The LaunchDaemon is configured to start the service automatically on boot

## Troubleshooting

If you encounter any issues:

1. Check the build logs:
```bash
./scripts/build-installers.sh 2>&1 | tee build.log
```

2. Verify package contents before installation:
```bash
pkgutil --expand build/sssonector-1.0.0.pkg expanded
```

3. Check system logs after installation:
```bash
sudo log show --predicate 'subsystem == "com.o3willard.sssonector"' --last 15m
```

For additional help, please open an issue on GitHub with the following information:
- macOS version
- Go version
- Build logs
- Any error messages

# AI Context Restoration Guide for SSSonector

This document provides instructions for restoring the full project context state for new VSCode with Cline AI instances.

## Project Structure Overview

The SSSonector project is a cross-platform SSL tunneling application located in:
```
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/
```

## Restoration Steps

1. Ensure all project files are accessible in the correct paths
2. Open VSCode with Cline extension
3. Start a new chat session
4. Copy the "Context Initialization Block" below into your first task message

## Context Initialization Block

Copy and paste the following block when starting a new task session with the Planning AI:

```
I need to initialize project context for the SSSonector SSL tunneling application. Here are the relevant files and their paths:

Core Documentation:
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/project_context.md
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/README.md
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/RELEASE_NOTES.md

Installation Documentation:
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/installation.md
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/linux_install.md
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/windows_install.md
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/ubuntu_install.md
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/macos_build.md

Build Configuration:
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/Makefile
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/installers/windows.nsi
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/.gitignore

Testing and Development:
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/virtualbox_testing.md

Distribution:
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/dist/v1.0.0/README.md
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/build/checksums.txt

Code Structure:
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/code_structure_snapshot.md

Please read these files to understand the project context and provide guidance on [specify your task here].
```

## Additional Context Notes

1. Key Documentation Files:
   - project_context.md: Comprehensive project overview
   - code_structure_snapshot.md: Detailed source code organization
   
2. The project_context.md contains:
   - Core components and architecture
   - Recent updates and version history
   - Known issues and limitations
   - Development guidelines
   - Testing procedures
   - Maintenance requirements

2. Key Version Information:
   - Current Version: 1.0.0
   - Last Updated: January 31, 2025
   - Major Features: Cross-platform SSL tunneling with bandwidth control

3. Critical Areas for AI Understanding:
   - Thread safety implementations
   - Platform-specific interface management
   - Security considerations
   - Resource management
   - Error handling patterns

## Verification Steps

After initializing context with a new AI instance:

1. Verify the AI can access and read all specified files
2. Confirm understanding of core components and architecture
3. Test knowledge of platform-specific implementations
4. Validate awareness of current issues and limitations

## Troubleshooting

If the AI seems to lack context:
1. Ensure all file paths are correct and files are accessible
2. Verify the project_context.md file is up to date
3. Check if any new documentation has been added that should be included
4. Consider providing additional context about specific areas of focus

Remember to update this restoration guide whenever significant changes are made to the project structure or documentation.

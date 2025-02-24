# QA Scripts Directory

This directory contains essential QA scripts that are maintained as part of Project SENTINEL.

## Active Scripts

1. `cleanup_resources.sh`
   - Purpose: Ensures clean state by removing any existing tunnel processes and interfaces
   - Used by: Test environment setup and teardown
   - Status: Actively maintained

2. `tunnel_control.sh`
   - Purpose: Manages tunnel server and client processes for testing
   - Used by: Core functionality testing
   - Status: Actively maintained

## Legacy Scripts

All legacy QA scripts have been archived to `.legacy_qa_backup/` as part of the transition to Project SENTINEL. These scripts are preserved for reference but are no longer actively maintained.

## Project SENTINEL

This directory is now managed under Project SENTINEL (SSSonector ENvironment Testing & Integration Layer), which provides a comprehensive QA environment restoration and validation initiative for the SSSonector project.

For new QA script development, please refer to the Project SENTINEL documentation and follow the established guidelines for test environment validation and automation.

## Known Good Working State

The last known good working state of the QA environment is preserved in `/test/known_good_working/`. This serves as a reference point and fallback configuration until Project SENTINEL establishes an improved baseline.

#!/bin/bash
#
# test/test_install.sh - Tests for install.sh macOS launchd integration
#
# Verifies (must fail if reverted):
#   1. Darwin branch installs plist + launchctl loads it (dry-run with fake launchctl)
#   2. Re-running is idempotent (no duplicate-load error)
#   3. Linux systemd path is unchanged
#
# Usage: bash test/test_install.sh
set -uo pipefail

TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$TEST_DIR/.." && pwd)"
INSTALL_SH="$REPO_ROOT/install.sh"

PASS=0
FAIL=0

ok()  { PASS=$((PASS+1)); echo "  PASS: $1"; }
bad() { FAIL=$((FAIL+1)); echo "  FAIL: $1"; }

assert_eq() {
    if [ "$1" = "$2" ]; then
        ok "$3"
    else
        bad "$3 (expected: '$2', got: '$1')"
    fi
}

assert_contains() {
    if grep -q -- "$2" "$1" 2>/dev/null; then
        ok "$3"
    else
        bad "$3 (pattern '$2' not found in $1)"
    fi
}

assert_file_exists() {
    if [ -f "$1" ]; then
        ok "$2"
    else
        bad "$2 (file not found: $1)"
    fi
}

assert_func_exists() {
    if type "$1" &>/dev/null; then
        ok "$2"
    else
        bad "$2 (function '$1' not found)"
    fi
}

# ---------------------------------------------------------------------------
# Create a fake launchctl that records all calls and always succeeds
# ---------------------------------------------------------------------------
create_fake_launchctl() {
    local bin_dir="$1"
    local log_file="$2"

    cat > "$bin_dir/launchctl" << LAUNCHCTL_EOF
#!/bin/bash
# Fake launchctl for testing - records calls, always exits 0
echo "\$*" >> "$log_file"

case "\$1" in
    load|unload|bootstrap|bootout|list|start|stop|enable|disable)
        exit 0
        ;;
    *)
        exit 0
        ;;
esac
LAUNCHCTL_EOF
    chmod +x "$bin_dir/launchctl"
}

# ---------------------------------------------------------------------------
# Create a fake systemctl that records calls and always succeeds
# ---------------------------------------------------------------------------
create_fake_systemctl() {
    local bin_dir="$1"
    local log_file="$2"

    cat > "$bin_dir/systemctl" << SYSTEMCTL_EOF
#!/bin/bash
# Fake systemctl for testing - records calls, always exits 0
echo "\$*" >> "$log_file"
exit 0
SYSTEMCTL_EOF
    chmod +x "$bin_dir/systemctl"
}

# ---------------------------------------------------------------------------
# Set up a temp environment with overridden paths
# ---------------------------------------------------------------------------
setup_env() {
    local tmpdir="$1"

    mkdir -p "$tmpdir/bin"
    mkdir -p "$tmpdir/etc/sssonector/instances"
    mkdir -p "$tmpdir/var/log/sssonector"

    # Set env vars for install.sh path overrides
    export SSSONECTOR_INSTALL_DIR="$tmpdir/bin"
    export SSSONECTOR_CONFIG_DIR="$tmpdir/etc/sssonector"
    export SSSONECTOR_INSTANCE_DIR="$tmpdir/etc/sssonector/instances"
    export SSSONECTOR_LOG_DIR="$tmpdir/var/log/sssonector"

    # Clear any previous launchd/systemd dir overrides so we can set them per-test
    unset SSSONECTOR_LAUNCHD_DIR
    unset SSSONECTOR_SYSTEMD_DIR

    export PATH="$tmpdir/bin:$PATH"
}

# ---------------------------------------------------------------------------
# Source install.sh without running main
# ---------------------------------------------------------------------------
source_install() {
    # install.sh has "set -e"; disable it after sourcing so test assertions
    # don't abort the whole script on a non-zero return.
    # Also disable set -u (nounset) for compatibility with install.sh functions.
    source "$INSTALL_SH"
    set +e +u
}

# ===========================================================================
# Test 1: Darwin install creates plist + calls launchctl load
# ===========================================================================
test_darwin_plist_install() {
    echo "=== Test 1: Darwin install creates plist + launchctl load ==="

    local tmpdir
    tmpdir=$(mktemp -d)
    setup_env "$tmpdir"

    # macOS-specific setup
    export SSSONECTOR_LAUNCHD_DIR="$tmpdir/Library/LaunchDaemons"
    mkdir -p "$SSSONECTOR_LAUNCHD_DIR"

    # Create fake launchctl
    local launchctl_log="$tmpdir/launchctl_calls.log"
    : > "$launchctl_log"
    create_fake_launchctl "$tmpdir/bin" "$launchctl_log"

    # Create a dummy instance config (simulates prior interactive_setup)
    mkdir -p "$tmpdir/etc/sssonector/instances/test-instance/certs"
    echo "schema_version: 2.0.0" > "$tmpdir/etc/sssonector/instances/test-instance/config.yaml"

    # Create dummy binary
    echo '#!/bin/bash' > "$tmpdir/bin/sssonector"
    chmod +x "$tmpdir/bin/sssonector"

    source_install
    assert_func_exists "install_launchd_service" "install_launchd_service function exists"

    # Call install_launchd_service
    install_launchd_service "test-instance" 2>&1
    local rc=$?
    assert_eq "$rc" "0" "install_launchd_service exits 0"

    local plist_path="$SSSONECTOR_LAUNCHD_DIR/com.o3willard.sssonector.test-instance.plist"
    assert_file_exists "$plist_path" "Plist file created"

    assert_contains "$plist_path" "com.o3willard.sssonector.test-instance" "Plist has correct instance Label"
    assert_contains "$plist_path" "$tmpdir/bin/sssonector" "Plist has correct binary path"
    assert_contains "$plist_path" "instances/test-instance/config.yaml" "Plist has correct config path"
    assert_contains "$plist_path" "<key>RunAtLoad</key>" "Plist has RunAtLoad"
    assert_contains "$plist_path" "<key>KeepAlive</key>" "Plist has KeepAlive"

    # Verify launchctl was called
    assert_contains "$launchctl_log" "unload" "launchctl unload called (idempotent pre-step)"
    assert_contains "$launchctl_log" "load" "launchctl load called"

    # Verify load was called with the plist file
    if grep -q "load.*\.plist" "$launchctl_log" 2>/dev/null; then
        ok "launchctl load called with plist file path"
    else
        bad "launchctl load not called with plist file path"
    fi

    rm -rf "$tmpdir"
}

# ===========================================================================
# Test 2: Idempotency - re-running does not error
# ===========================================================================
test_idempotency() {
    echo "=== Test 2: Re-running install_launchd_service is idempotent ==="

    local tmpdir
    tmpdir=$(mktemp -d)
    setup_env "$tmpdir"

    export SSSONECTOR_LAUNCHD_DIR="$tmpdir/Library/LaunchDaemons"
    mkdir -p "$SSSONECTOR_LAUNCHD_DIR"

    local launchctl_log="$tmpdir/launchctl_calls.log"
    : > "$launchctl_log"
    create_fake_launchctl "$tmpdir/bin" "$launchctl_log"

    mkdir -p "$tmpdir/etc/sssonector/instances/test-instance/certs"
    echo "schema_version: 2.0.0" > "$tmpdir/etc/sssonector/instances/test-instance/config.yaml"
    echo '#!/bin/bash' > "$tmpdir/bin/sssonector"
    chmod +x "$tmpdir/bin/sssonector"

    source_install

    local plist_path="$SSSONCTOR_LAUNCHD_DIR/com.o3willard.sssonector.test-instance.plist"

    # First run
    install_launchd_service "test-instance" 2>&1
    assert_eq "$?" "0" "First run exits 0"

    # Second run
    install_launchd_service "test-instance" 2>&1
    assert_eq "$?" "0" "Second run exits 0 (idempotent)"

    # Third run
    install_launchd_service "test-instance" 2>&1
    assert_eq "$?" "0" "Third run exits 0 (idempotent)"

    # Plist should still exist with correct content
    assert_file_exists "$SSSONECTOR_LAUNCHD_DIR/com.o3willard.sssonector.test-instance.plist" "Plist persists after re-runs"

    # Count load and unload calls - each run should have one unload + one load
    local load_count
    load_count=$(grep -c "load" "$launchctl_log" 2>/dev/null || echo 0)
    if [ "$load_count" -ge 3 ]; then
        ok "launchctl load called on each run ($load_count times total)"
    else
        bad "launchctl load called $load_count times, expected >= 3"
    fi

    local unload_count
    unload_count=$(grep -c "unload" "$launchctl_log" 2>/dev/null || echo 0)
    if [ "$unload_count" -ge 3 ]; then
        ok "launchctl unload called on each run ($unload_count times total)"
    else
        bad "launchctl unload called $unload_count times, expected >= 3"
    fi

    # Verify no duplicate plist entries (idempotent)
    local plist_count
    plist_count=$(grep -c "com.o3willard.sssonector.test-instance.plist" "$launchctl_log" 2>/dev/null || echo 0)
    if [ "$plist_count" -ge 6 ]; then
        ok "Plist path passed to launchctl on each unload+load call"
    else
        bad "Plist path only passed $plist_count times, expected >= 6"
    fi

    rm -rf "$tmpdir"
}

# ===========================================================================
# Test 3: Multiple instances create separate plists
# ===========================================================================
test_multiple_instances() {
    echo "=== Test 3: Multiple instances create separate plists ==="

    local tmpdir
    tmpdir=$(mktemp -d)
    setup_env "$tmpdir"

    export SSSONECTOR_LAUNCHD_DIR="$tmpdir/Library/LaunchDaemons"
    mkdir -p "$SSSONECTOR_LAUNCHD_DIR"

    local launchctl_log="$tmpdir/launchctl_calls.log"
    : > "$launchctl_log"
    create_fake_launchctl "$tmpdir/bin" "$launchctl_log"

    # Create dummy configs for two instances
    for inst in tunnel-a tunnel-b; do
        mkdir -p "$tmpdir/etc/sssonector/instances/$inst/certs"
        echo "schema_version: 2.0.0" > "$tmpdir/etc/sssonector/instances/$inst/config.yaml"
    done

    echo '#!/bin/bash' > "$tmpdir/bin/sssonector"
    chmod +x "$tmpdir/bin/sssonector"

    source_install

    install_launchd_service "tunnel-a" 2>&1
    assert_eq "$?" "0" "First instance install exits 0"

    install_launchd_service "tunnel-b" 2>&1
    assert_eq "$?" "0" "Second instance install exits 0"

    local plist_a="$SSSONECTOR_LAUNCHD_DIR/com.o3willard.sssonector.tunnel-a.plist"
    local plist_b="$SSSONECTOR_LAUNCHD_DIR/com.o3willard.sssonector.tunnel-b.plist"

    assert_file_exists "$plist_a" "Instance A plist created"
    assert_file_exists "$plist_b" "Instance B plist created"

    assert_contains "$plist_a" "com.o3willard.sssonector.tunnel-a" "Plist A has correct Label"
    assert_contains "$plist_b" "com.o3willard.sssonector.tunnel-b" "Plist B has correct Label"
    assert_contains "$plist_a" "instances/tunnel-a/config.yaml" "Plist A has correct config path"
    assert_contains "$plist_b" "instances/tunnel-b/config.yaml" "Plist B has correct config path"

    rm -rf "$tmpdir"
}

# ===========================================================================
# Test 4: Linux systemd path is unchanged
# ===========================================================================
test_linux_path_unchanged() {
    echo "=== Test 4: Linux systemd path is unchanged ==="

    local tmpdir
    tmpdir=$(mktemp -d)
    setup_env "$tmpdir"

    # Linux-specific setup
    export SSSONECTOR_SYSTEMD_DIR="$tmpdir/etc/systemd/system"
    mkdir -p "$SSSONECTOR_SYSTEMD_DIR"

    local systemctl_log="$tmpdir/systemctl_calls.log"
    : > "$systemctl_log"
    create_fake_systemctl "$tmpdir/bin" "$systemctl_log"

    source_install

    assert_func_exists "install_systemd_service" "install_systemd_service function exists"

    # Call install_systemd_service
    install_systemd_service 2>&1
    assert_eq "$?" "0" "install_systemd_service exits 0"

    local service_file="$SSSONECTOR_SYSTEMD_DIR/sssonector@.service"
    assert_file_exists "$service_file" "Systemd service template file created"

    assert_contains "$service_file" "Type=simple" "Service file has Type=simple"
    assert_contains "$service_file" "ExecStart=/usr/local/bin/sssonector -config /etc/sssonector/instances/%i/config.yaml" "Service file has correct ExecStart"
    assert_contains "$service_file" "WantedBy=multi-user.target" "Service file has correct Install section"
    assert_contains "$service_file" "User=root" "Service file runs as root"
    assert_contains "$service_file" "ProtectSystem=strict" "Service file has security hardening"

    # Verify systemctl daemon-reload was called
    assert_contains "$systemctl_log" "daemon-reload" "systemctl daemon-reload called"

    rm -rf "$tmpdir"
}

# ===========================================================================
# Test 5: install_launchd_service function exists and is not empty
# ===========================================================================
test_function_exists() {
    echo "=== Test 5: install_launchd_service function exists ==="

    local tmpdir
    tmpdir=$(mktemp -d)
    setup_env "$tmpdir"

    source_install

    assert_func_exists "install_launchd_service" "install_launchd_service is defined as a function"

    # Verify the function has meaningful content
    local func_body
    func_body=$(declare -f install_launchd_service 2>/dev/null || true)
    if [ -n "$func_body" ]; then
        ok "install_launchd_service has a body"
    else
        bad "install_launchd_service has empty body"
    fi

    # Verify key launchd operations are present
    if echo "$func_body" | grep -q "launchctl"; then
        ok "Function contains launchctl calls"
    else
        bad "Function missing launchctl calls"
    fi

    if echo "$func_body" | grep -q "plist"; then
        ok "Function generates plist content"
    else
        bad "Function does not generate plist content"
    fi

    if echo "$func_body" | grep -q "unload"; then
        ok "Function implements idempotent unload"
    else
        bad "Function missing idempotent unload"
    fi

    rm -rf "$tmpdir"
}

# ===========================================================================
# Test 6: Darwin early-exit removed from main() (source-level check)
# ===========================================================================
test_darwin_early_exit_removed() {
    echo "=== Test 6: Darwin early-exit removed from main() ==="

    # The old behavior was: detect darwin → print error → exit 1
    # If someone reverts main(), this test must fail.
    if grep -q "This install script is for Linux. For macOS" "$INSTALL_SH"; then
        bad "install.sh still has the Linux-only error message (darwin early-exit was restored)"
    else
        ok "install.sh no longer has the Linux-only error message"
    fi

    # Verify the darwin branch in main calls install_launchd_service
    if grep -A5 'install_launchd_service.*instance_name' "$INSTALL_SH" | grep -q 'install_launchd_service'; then
        ok "install.sh has darwin launchd branch in main()"
    else
        bad "install.sh missing darwin launchd branch in main()"
    fi

    # Verify the darwin early-exit block (detect darwin → error → exit 1) is gone
    # The old code had: detect_os returns "darwin", then main() exits 1 with a message
    # We verify by checking the error message is absent (covers the whole block)
    if grep -q 'For macOS, please use install_macos.sh' "$INSTALL_SH"; then
        bad "install.sh still has the macOS redirect message (darwin early-exit restored)"
    else
        ok "install.sh no longer redirects macOS to install_macos.sh"
    fi

    # Verify print_success receives $os argument
    if grep -q 'print_success.*\$os' "$INSTALL_SH"; then
        ok "print_success receives os argument"
    else
        bad "print_success missing os argument"
    fi
}

# ===========================================================================
# Run all tests
# ===========================================================================
echo ""
echo "============================================"
echo "install.sh macOS launchd integration tests"
echo "============================================"
echo ""

test_darwin_plist_install
echo ""

test_idempotency
echo ""

test_multiple_instances
echo ""

test_linux_path_unchanged
echo ""

test_function_exists
echo ""

test_darwin_early_exit_removed
echo ""

echo "============================================"
echo "Results: ${PASS} passed, ${FAIL} failed"
echo "============================================"
exit "$FAIL"

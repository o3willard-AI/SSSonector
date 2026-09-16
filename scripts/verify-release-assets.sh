#!/bin/bash
#
# verify-release-assets.sh — dry-run of the release asset list.
#
# Derives the asset set .github/workflows/release.yml will publish from its
# build matrix, then checks that every download URL the installers construct
# (install.sh and scripts/install.ps1, per the fixed asset-name convention
# sssonector-<os>-<arch>[.exe]) resolves to a published asset, and that the
# SHA256SUMS command in the workflow covers every binary asset — including
# the windows .exe files, whose extension is part of the asset name.
#
# Purely static: no network, no tag needed. Run locally and in CI.
#
# Usage: scripts/verify-release-assets.sh

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

RELEASE_WF=".github/workflows/release.yml"
REPO="o3willard-AI/SSSonector"
failures=0

fail() {
    echo "FAIL: $1" >&2
    failures=$((failures + 1))
}

# ---------------------------------------------------------------------------
# 1. Asset names release.yml will publish (parsed from the build matrix).
#    Mirrors the build step's own naming: sssonector-<os>-<arch>, with ".exe"
#    appended for windows.
# ---------------------------------------------------------------------------
published=()
cur_os=""
while IFS= read -r line; do
    case "$line" in
        *"os: "*)
            cur_os="${line##*os: }"
            cur_os="${cur_os%% *}"
            ;;
        *"arch: "*)
            arch="${line##*arch: }"
            arch="${arch%% *}"
            name="sssonector-${cur_os}-${arch}"
            [ "$cur_os" = "windows" ] && name="${name}.exe"
            published+=("$name")
            ;;
    esac
done < <(sed -n '/^        include:/,/^      steps:/p' "$RELEASE_WF")

# Non-binary assets published by the release job.
published+=("SHA256SUMS" "sbom.cyclonedx.json" "install.sh" "install.ps1")

echo "== release.yml publishes ${#published[@]} assets:"
printf '   %s\n' "${published[@]}"

is_published() {
    local want="$1" a
    for a in "${published[@]}"; do
        [ "$a" = "$want" ] && return 0
    done
    return 1
}

# ---------------------------------------------------------------------------
# 2. Every download URL the installers construct must resolve to a
#    published asset. install.sh/upgrade.sh build
#    sssonector-<os>-<arch> for every (os, arch) their detection accepts;
#    install.ps1 (WI 7.1 convention) downloads sssonector-windows-<arch>.exe.
#    A URL "resolves" iff its basename is in the published set.
# ---------------------------------------------------------------------------
check_url() {
    local os="$1" arch="$2" src="$3"
    local name="sssonector-${os}-${arch}"
    [ "$os" = "windows" ] && name="${name}.exe"
    local url="https://github.com/${REPO}/releases/download/<tag>/${name}"
    if is_published "$name"; then
        echo "   OK   ${src}: ${url}"
    else
        fail "${src} downloads ${url} but release.yml does not publish '${name}'"
        echo "   MISS ${src}: ${url}" >&2
    fi
}

echo
echo "== download URLs used by installers (dry-run against asset list):"
# install.sh: detect_os/detect_arch accept these combinations on Linux.
check_url linux amd64 "install.sh"
check_url linux arm64 "install.sh"
check_url linux arm    "install.sh" # armv7l hosts
# install.ps1 (WI 7.1 asset convention): windows binaries.
check_url windows amd64 "install.ps1"
check_url windows arm64 "install.ps1"

# Guard the convention itself: install.sh/upgrade.sh must construct the
# asset name as sssonector-${os}-${arch} and fetch SHA256SUMS from the same
# release. If a script is renamed/reworked, this fails loudly.
for script in install.sh upgrade.sh; do
    grep -q 'binary_name="sssonector-${os}-${arch}"' "$script" \
        || fail "$script no longer constructs asset names as sssonector-\${os}-\${arch}"
    grep -q 'SHA256SUMS' "$script" \
        || fail "$script no longer verifies against SHA256SUMS"
done

# ---------------------------------------------------------------------------
# 3. The SHA256SUMS command in release.yml must cover every binary asset.
#    Simulate its shell glob patterns against the binary names.
# ---------------------------------------------------------------------------
shasum_line="$(grep -E '^\s*sha256sum .*>\s*SHA256SUMS' "$RELEASE_WF")" \
    || fail "cannot find the sha256sum ... > SHA256SUMS step in release.yml"
patterns="$(printf '%s\n' "$shasum_line" \
    | sed -E 's/^[[:space:]]*sha256sum //; s/[[:space:]]*>[[:space:]]*SHA256SUMS[[:space:]]*$//')"

echo
echo "== SHA256SUMS coverage of binary assets:"
for name in "${published[@]}"; do
    case "$name" in
        sssonector-*) ;;
        *) continue ;;
    esac
    covered=0
    # shellcheck disable=SC2086
    for pat in $patterns; do
        case "$name" in
            $pat) covered=1; break ;;
        esac
    done
    if [ "$covered" -eq 1 ]; then
        echo "   OK   SHA256SUMS covers ${name}"
    else
        fail "SHA256SUMS patterns (in release.yml) do not cover '${name}'"
    fi
done

echo
if [ "$failures" -ne 0 ]; then
    echo "RESULT: ${failures} failure(s)" >&2
    exit 1
fi
echo "RESULT: all installer download URLs resolve against the release.yml asset list"
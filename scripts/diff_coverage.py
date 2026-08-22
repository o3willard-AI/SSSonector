#!/usr/bin/env python3
"""Fail unless a threshold of changed executable lines is covered by tests.

Usage:
  diff_coverage.py --profile cover.out --base FETCH_HEAD [--threshold 80]

Changed lines come from `git diff --unified=0 <base>...HEAD` restricted to
non-test Go files. A changed line counts as covered if it falls inside any
Go coverage block whose execution count is > 0. Lines outside every block
(closing braces, comments) are treated as non-executable and skipped.
"""
import argparse
import re
import subprocess
import sys

HUNK_RE = re.compile(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@")
BLOCK_RE = re.compile(r"^(.+?):(\d+)\.\d+,(\d+)\.\d+ (\d+) (\d+)$")


def changed_lines(base: str) -> dict[str, set[int]]:
    out = subprocess.run(
        ["git", "diff", "--unified=0", f"{base}...HEAD"],
        capture_output=True, text=True, check=True,
    ).stdout
    result: dict[str, set[int]] = {}
    current = None
    new_line = 0
    for line in out.splitlines():
        if line.startswith("+++ b/"):
            current = line[6:]
            continue
        if line.startswith(("diff ", "index ", "--- ", "new file mode",
                            "deleted file mode", "similarity index",
                            "rename from", "rename to")) or \
           line.startswith("\\" + " No newline"):
            continue
        m = HUNK_RE.match(line)
        if m:
            new_line = int(m.group(1))
            continue
        if current is None or not current.endswith(".go") or current.endswith("_test.go"):
            continue
        if line.startswith("+"):
            result.setdefault(current, set()).add(new_line)
            new_line += 1
        elif line.startswith("-") or line.startswith("\\"):
            pass
        else:
            new_line += 1
    return result


def covered_lines(profile: str) -> tuple[dict[str, dict[int, bool]], set[str]]:
    """Return {file: {line: covered}} for all executable lines, plus file list."""
    exec_lines: dict[str, dict[int, bool]] = {}
    for raw in open(profile):
        m = BLOCK_RE.match(raw.strip())
        if not m:
            continue
        path, start, end, _, count = m.groups()
        covered = int(count) > 0
        lines = exec_lines.setdefault(path, {})
        for ln in range(int(start), int(end) + 1):
            lines[ln] = lines.get(ln, False) or covered
    return exec_lines, set(exec_lines)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--profile", required=True)
    ap.add_argument("--base", required=True)
    ap.add_argument("--threshold", type=float,
                    default=float(__import__("os").environ.get("DIFF_COVERAGE_THRESHOLD", "80")))
    args = ap.parse_args()

    changed = changed_lines(args.base)

    # Coverage profiles carry the full Go module path; diff paths are
    # repository-relative. Normalize profile keys to repo-relative.
    module = subprocess.run(["go", "list", "-m"], capture_output=True,
                            text=True, check=True).stdout.strip()
    exec_map, exec_files = covered_lines(args.profile)
    exec_map = {
        (path[len(module) + 1:] if path.startswith(module + "/") else path): blocks
        for path, blocks in exec_map.items()
    }

    total = covered = 0
    misses: list[tuple[str, list[int]]] = []
    for path, lines in sorted(changed.items()):
        blocks = exec_map.get(path)
        if not blocks:
            continue  # new/changed files without instrumentation cannot be judged here
        missed = []
        for ln in sorted(lines):
            state = blocks.get(ln)
            if state is None:
                continue  # non-executable line
            total += 1
            if state:
                covered += 1
            else:
                missed.append(ln)
        if missed:
            misses.append((path, missed))

    pct = 100.0 * covered / total if total else 100.0
    print(f"Diff coverage: {covered}/{total} changed executable lines ({pct:.1f}%)")
    for path, missed in misses[:20]:
        preview = ",".join(map(str, missed[:25]))
        more = "" if len(missed) <= 25 else f" (+{len(missed)-25} more)"
        print(f"  UNCOVERED {path}: {preview}{more}")
    if total and pct < args.threshold:
        print(f"FAIL: below {args.threshold:.0f}% threshold")
        return 1
    print("OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())

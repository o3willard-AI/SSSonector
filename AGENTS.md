# AGENTS.md — Conventions for Agentic & Human Contributors

This repository is maintained with heavy use of AI agents. These rules exist
because their absence produced measurable debt: four parallel buffer pools,
five never-imported packages (~14% of LoC), broken build scripts, committed
private keys, and shipped features that did not work. Follow these rules so
further automation stops compounding that debt.

## Build & verify before you claim done

```bash
go build ./...          # must compile
go vet ./...            # must be silent
go test -race -count=1 ./...   # all green; no skipped failures
make build-linux-amd64  # packaging path works end-to-end
```

CI enforces the same gates plus govulncheck and Gitleaks. Never merge red.

## Architecture invariants

1. **One implementation per concept.** Before writing a new pool, limiter,
   validator, daemon, or proxy: search for the existing one and extend it.
   Do not regenerate near-duplicates under a new name.
2. **Wire it or delete it.** Code that nothing imports gets deleted, not
   parked. Dead packages are liability, not inventory.
3. **Single entry point direction:** `cmd/daemon` is the service binary.
   Do not add competing mains without an ADR.
4. **Data-path correctness:** any bidirectional copier must half-close
   (`CloseWrite`) when one direction finishes. See `internal/facade/proxy.go`
   as the reference implementation.
5. **No sleeps while holding locks.** Pace via reservation/debt models
   (see `internal/throttle/token_bucket.go`).

## Security rules (non-negotiable)

- **Fail closed.** Missing credentials/secrets are fatal errors, never
  derived from public material or defaulted to permissive values.
  The facade token secret MUST come from explicit configuration.
- Never default an ambiguous config into *server* mode or any listening state.
- No private keys, certificates, tokens, or passwords in git — ever — not
  even "temporarily". Gitleaks runs in CI on every PR.
- New dependencies must be canonical modules, pinned in go.sum, justified
  in the PR description, and free of known vulns (`govulncheck`).
- Test-only behavior (fake CAs, short cert lifetimes) lives in `_test.go`
  files, never behind flags inside production code paths.

## Tests

- Behavior changes ship with tests that can fail. A test suite that passes
  regardless of implementation is a defect; mutation-test critical paths.
- Rate/QoS claims require timing assertions backed by deterministic math
  (see `internal/throttle/*_test.go`).
- Fix drifted fixtures to match the *current* documented schema; if the
  schema is wrong, change the schema deliberately — not silently both.

## Commits

- Sign every commit (SSH or GPG: `git config commit.gpgsign true` plus
  `user.signingkey`). Unsigned commits fail provenance review even when all
  other gates pass.
- One logical change per commit; do not commit binaries,
  object files, logs, QA snapshots, or editor state (.gitignore covers them).

## Supply chain / CI

- All GitHub Actions are pinned by commit SHA (verified against the remote).
  Adding an action by floating tag is rejected in review.
- Release artifacts publish SHA256SUMS and installers verify them;
  smoke tests run BEFORE publishing, not after.
- Commits: keep one logical change per commit; do not commit binaries,
  object files, logs, QA snapshots, or editor state (.gitignore covers them).

## Platform binary builders

Build each platform's binary on its native runner so the release carries
genuine per-OS artifacts (see `.github/workflows/release.yml` for the matrix
and authoritative naming). Bring the binary online by uploading it as a
release asset alongside `SHA256SUMS`. Version is injected via
`-X main.Version=<tag>`; the Linux agent provides `sssonector-linux-amd64` /
`sssonector-linux-arm64`.

- **macOS builder** — produce and upload:
  - `sssonector-darwin-amd64`
  - `sssonector-darwin-arm64`
  - Build on a macOS host (or the `macos-latest` runner):
    `CGO_ENABLED=0 GOOS=darwin GOARCH=<arch> go build -trimpath \
    -ldflags="-s -w -X main.Version=<tag>" -o dist/sssonector-darwin-<arch> ./cmd/daemon`
  - Sanity: `./dist/sssonector-darwin-<arch> -version` must print the tag.

- **Windows builder** — produce:
  - `sssonector-windows-amd64.exe`
  - Build on a Windows host (or the `windows-latest` runner):
    `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath \
    -ldflags="-s -w -X main.Version=v<tag>" -o dist/sssonector-windows-amd64.exe ./cmd/daemon`
  - Sanity: run `dist\sssonector-windows-amd64.exe -version` and confirm it
    prints the tag.

Upload both to the same GitHub release as the Linux artifacts and the
`SHA256SUMS` manifest before marking the release live.

## Documentation

- Update docs in the same PR that changes behavior. Drift between docs and
  code is treated as a bug in the code PR.
- `Issues.md` tracks only currently-true issues. Stale roadmaps get deleted.

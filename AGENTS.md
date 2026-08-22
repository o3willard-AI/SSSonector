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

## Supply chain / CI

- All GitHub Actions are pinned by commit SHA (verified against the remote).
  Adding an action by floating tag is rejected in review.
- Release artifacts publish SHA256SUMS and installers verify them;
  smoke tests run BEFORE publishing, not after.
- Commits: keep one logical change per commit; do not commit binaries,
  object files, logs, QA snapshots, or editor state (.gitignore covers them).

## Documentation

- Update docs in the same PR that changes behavior. Drift between docs and
  code is treated as a bug in the code PR.
- `Issues.md` tracks only currently-true issues. Stale roadmaps get deleted.

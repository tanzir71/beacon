# Build State

**Current milestone:** B0 - New repository, the gate, and the landing page
**Last updated:** 2026-11-24
**Gate status:** green
**Iterations completed:** 1

## Milestone Progress

- [ ] B0 - New repository, the gate, and the landing page
- [ ] B1 - Identity and people on this network
- [ ] B2 - Messages on the LAN
- [ ] B3 - Phones join by QR
- [ ] B4 - Invites and the address book
- [ ] B5 - Lighthouses, links beyond the LAN, reach
- [ ] B6 - Sites over peers
- [ ] B7 - Off-grid: saved copies, shelves, USB, fire drill
- [ ] B8 - Safety and squares
- [ ] B9 - Hardening and release 1.0

## Current Milestone Criteria

- [ ] Renamed project passes the complete inherited `make check` gate.
- [ ] Beacon install/uninstall preserves an existing Dropserve installation.
- [ ] Landing translations have identical keys; generated pages are current.
- [ ] HTML validation, 150 KB budget, and outside-resource checks pass.
- [ ] Browser language precedence and privacy tests pass.
- [ ] Both languages and themes pass axe AA and the 320 px layout check.
- [ ] Empty release metadata produces no dead download links.
- [ ] ADR-015 and ADR-016 are written with verified evidence.
- [ ] Published Pages URL returns HTTP 200.
- [ ] Demo transcript, commit, milestone tag, push, and binary pre-release exist.

## Repository Bootstrap

- Source: local clean clone of `tanzir71/dropserve` at `1c1e433`.
- Work directory: `D:\claude-upgrade-26\local-lan\beacon`.
- Created public repository: https://github.com/tanzir71/beacon.
- Cloned with `--no-tags`; source remote removed before adding Beacon origin.
- Neither the Dropserve checkout nor the Python prototype has been modified.
- Original Dropserve ledger remains available in Git at `1c1e433:STATE.md`.
- Original handover and ADRs 001 through 014 remain historical references.

## Iteration Evidence

### Baseline Gate Repair

- This shell initially had neither Go nor Make on PATH.
- Installed portable Go 1.27.4 under the approved temporary directory, verified
  against official SHA-256 `178ab98950ac69ab25069307dc6ade79f3816e414cce95b4da166a115cde1e7e`.
- Installed portable LLVM-MinGW 20261117 UCRT x86_64 for race-detector tests,
  verified against release SHA-256 `4d002e330a0f703625aa5ccbc37e96f8f824413e163bfc7336774aabd2c3a412`.
- Existing Make: Android NDK 28.2.13676358 `prebuilt/windows-x86_64/bin`.
- Build commands prepend these toolchain `bin` directories to the process PATH.
  Race tests use `CGO_ENABLED=1`; the gate still forces `CGO_ENABLED=0` for all
  shipped cross-builds and the optional Windows tray build.
- First full gate failed `TestLANChangeReissuesLeafWithinOneMonitorInterval`:
  `leaf was not reissued within 250ms (elapsed 260.9589ms, last read error: <nil>)`.
  Cleanup also failed because cancellation did not join the writing goroutine.
- Fixed test teardown to cancel and join the monitor before temporary-file
  cleanup. Kept the 250 ms deadline, all assertions, and the gate unchanged.
- The original refresh assertion passed 20 focused race-detector repetitions
  in 0.03-0.05 seconds per case. The initial cold/full-suite deadline failure
  remains recorded as a cold-start timing risk, not erased by retries.
- After the teardown fix, `make check` exited 0, including formatting, portable
  lint, the complete race suite, version injection, all three zero-CGO target
  builds, the optional tray build, and the shipped-file scan: `gate: green`.
- A separate `go test -race -count=3 ./...` exited 0 with all packages passing
  uncached three times, including discovery (1.543s) and server (102.129s).
- The gate's existing portable lint fallback is `go vet`; golangci-lint is not
  installed locally. Hosted CI retains the full separate golangci-lint job.

## Decisions Made This Build

- Use the specified default repository name, `tanzir71/beacon`.
- Keep all original product invariants and gate stages; do not inherit milestone
  completion claims or tags from Dropserve.

## Open Questions For The Human

- None blocking local implementation. Release-signing secrets, the Banglish
  read-aloud review, lighthouse deployment, and real hardware proofs remain
  human-owned acceptance steps under the handover.

## Deviations From The Spec

- None. Beacon milestones are not complete until every required proof exists.

## Dependency Count

- Inherited: 10 direct, 20 indirect modules. No Beacon dependencies added yet.

## Next Target

Implement B0 product and installation identifier isolation test-first. The
baseline bootstrap repair is ready to commit; no milestone is yet complete.

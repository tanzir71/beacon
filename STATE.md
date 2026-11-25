# Build State

**Current milestone:** U1 - The app's own skeleton
**Last updated:** 2026-11-25
**Gate status:** green (local full lint/race/cross-build gate; timing risks recorded below)
**Iterations completed:** 7

## Milestone Progress

- [ ] U1 - The app's own skeleton
- [ ] B0 - New repository, the gate, and the landing page
- [ ] B1 - Identity and people on this network
- [ ] B2 - Messages on the LAN and over the internet
- [ ] B3 - Phones join by QR
- [ ] B4 - Invites and the address book
- [ ] B5 - Lighthouses, links beyond the LAN, reach
- [ ] B6 - Sites over peers
- [ ] B7 - Off-grid: saved copies, shelves, USB, fire drill
- [ ] B8 - Safety and squares
- [ ] B9 - Hardening and release 1.0

## Current Milestone Criteria

- [ ] `cmd/beacon` starts one `internal/app` supervisor with four equal services: sites, links, events, ui.
- [ ] First run asks for a display name and creates the identity key; adding a site is optional.
- [ ] One configuration has sites, people, messages, network and safety sections; one state directory and HTTP port.
- [ ] One left-rail shell: Reachable now Home with On this computer; hosting controls under Sites; no launchpad.
- [ ] The CLI is organised by sites, people, messages, network, bundle and global commands.
- [x] One behavior-preserving commit moves reused code into the layout in section 7.3; gate green before and after.
- [ ] No shipped/user-visible string, generated output, environment variable or written file carries the earlier product name; history exempt.
- [ ] `beacon.json` and legacy-only site manifests are honoured, and neither is written.
- [ ] `TestServiceFailureIsContained` proves independent panic containment for each of the four services.
- [ ] I1-I5 tests pass with unchanged meaning through the new skeleton.
- [x] Earlier documents are in docs/history, ADR-015 is rewritten, and both revision sections are recorded.

## B0 Criteria

- [x] `make check` exits 0 on a clean checkout.
- [ ] Beacon install/uninstall preserves an existing Dropserve installation.
- [ ] Landing translations have identical keys; generated pages are current.
- [ ] HTML validation, 150 KB budget, and outside-resource checks pass.
- [ ] Browser language precedence and privacy tests pass.
- [ ] Both languages and themes pass axe AA and the 320 px layout check.
- [ ] Empty release metadata produces no dead download links.
- [ ] ADR-015, ADR-016 (S6) and ADR-017 (delivery paths) are written with verified evidence.
- [ ] Published Pages URL returns HTTP 200.
- [ ] Demo transcript, commit, milestone tag, push, and binary pre-release exist.

## Starting Point

- The repository contains reused hosting code from `tanzir71/dropserve` at
  `1c1e433`, being fitted under Blackout Beacon's own application skeleton.
  That earlier project is prior work, not the base or specification of this app.
- Work directory: `D:\claude-upgrade-26\local-lan\beacon`.
- Repository: https://github.com/tanzir71/beacon.
- Cloned with `--no-tags`; source remote removed before adding Beacon origin.
- Neither the earlier hosting checkout nor Python prototype was modified.
- The prior build ledger is available in Git at `1c1e433:STATE.md`.
- The prior handover and ADRs 001-014 are historical references only, now moved
  unchanged into `docs/history/` under revision 1.2.

## Handover Revision 1.1 Applied

- Read and adopted on 2026-11-24. U1 now precedes B0; B2 is Messages on the LAN
  and over the internet. Added requirements are recorded, not claimed complete.
- There is one app with four equal services, not one product inside another.
- Internet delivery is optional: automatic offers every allowed available path;
  internet/beacon modes restrict paths; cautious implies beacon and mailbox-only.
- B1 adds I8: empty internet lists give zero WAN dials and no host literals
  outside seed lists. B2 adds S7, signed editable internet lists, max four
  connections, delivery-path labels, blocked-path tests and I12 deduplication.
- B5 adds fastest-path site/file selection and GP-8 internet-loss fallback.
- Refresh landing copy/designs from the authoritative sibling handover directory
  after U1. B0 adds the delivery rule, six FAQs, internet-delivery status,
  and ADR-017. No unchanged 1.0 copy is considered authoritative.

## Handover Revision 1.2 Applied

- Re-read revision 1.2 in full on 2026-11-24 after the owner's update. This
  document alone specifies Blackout Beacon; prior project documents bind nothing.
- Kept commits `2691ff8` and `0e553cf`, history, independent install identifiers,
  and the gate. No history rewrite or force push.
- Renamed Repository Bootstrap to Starting point, made U1 current, inserted
  all U1 criteria, renamed B2, and kept factual toolchain/test proof.
- Rewrote README/CONTRIBUTING/SECURITY, moved prior documents to history,
  replaced withdrawn ADR-015 with 015-one-app, put storage in ADR-018, and
  added delivery-path ADR-017. API documents explicitly note the U1 transition;
  the full API rewrite follows the app-owned shell/API implementation.
- Deleted the old landing page/screenshots. Hosting-centred setup/launchpad
  removal follows runtime restructuring; they are no longer presented as the
  product in active documentation. The move-only commit is separate from
  supervisor, configuration, identity, shell and CLI behavior changes.
- The old-name child environment decision is withdrawn. U1 emits only Beacon
  names and reads the legacy site manifest only without changing user files.
- Owner instruction at 2026-11-24T23:23Z: run no GitHub Actions. Use local
  equivalents; do not dispatch, rerun or push commits that trigger workflows.
  Local commits continue. External release/publication proofs stay pending.

## Iteration Evidence

### 1: Initial Gate Repair

- This shell initially had neither Go nor Make on PATH.
- Installed portable Go 1.27.4 under the approved temporary directory, verified
  SHA-256 `178ab98950ac69ab25069307dc6ade79f3816e414cce95b4da166a115cde1e7e`.
- Installed portable LLVM-MinGW 20261117 UCRT x86_64 for race tests, verified
  SHA-256 `4d002e330a0f703625aa5ccbc37e96f8f824413e163bfc7336774aabd2c3a412`.
- Existing Make: Android NDK 28.2.13676358 `prebuilt/windows-x86_64/bin`.
- Commands prepend these toolchain bin directories to process PATH. Race tests
  use CGO_ENABLED=1; every shipped cross-build and tray build forces 0.
- First full gate failed the TLS monitor's 250 ms deadline at 260.9589 ms and
  cleanup raced the still-writing goroutine. Joined the monitor after cancel,
  keeping the deadline and assertions. Twenty focused repetitions passed in
  0.03-0.05s; cold-start contention remains a recorded timing risk.
- `make check` and separate uncached `go test -race -count=3 ./...` exited 0.
- Committed as `2691ff8`.

### 2: Product Identifier Isolation

- Two new identity tests failed first on module, config/state/Apps roots, binary
  outputs, OS registrations and missing Beacon command/installer paths.
- Isolated the module, binaries, task, firewall, service, LaunchAgent, storage,
  API, sessions/CSRF, updates and release scripts. New installer AppId:
  `49BAE7EA-599A-4F46-A24B-7905900A85E9`.
- Preserved attribution, history, tests and prior documents. Kept legacy app
  manifest/environment contracts at that time; revision 1.2 now changes output.
- Focused tests and the full local gate exited 0. No OS task/firewall/trust or
  Tailscale action was run. This is not the side-by-side installer proof.
- Committed as `0e553cf`; pushed before the no-Actions instruction.
- No milestone tag or release was created.

### 3: U1 Gate Repair Before New Features

- Prior hosted run 45506076380 failed full lint and module-only security. It was
  read before the owner's no-Actions instruction. No workflow has been started,
  rerun or triggered since that instruction.
- Installed portable golangci-lint v2.15.0, verified SHA-256
  `cd7ea6063c95de245e17e5246e42bdc14bb221792c429dbd239ab3a6ade6ca6a`.
  Add its `golangci-lint-2.15.0-windows-amd64` directory to the temporary PATH.
- `make check` reproduced three errcheck failures locally. Child exit-status
  receives now explicitly document intentional discard during liveness/shutdown.
  Neither process behavior nor lint configuration was weakened.
- Local govulncheck reproduced eight x/net module advisories. Upgraded x/net
  v0.56.0 to the minimum all-fixed v0.62.0, requiring x/sys v0.49.0 and Go 1.26.0.
  Inspected module floors and BSD license; documented the new Go floor.
- The supported Go floor exposed deprecated ReverseProxy Director use. Moved
  to Rewrite, SetURL and SetXForwarded while preserving public Host and site
  prefixes. Proxy, cookie, download and WebSocket regression tests pass.
- Full `make check` exited 0: full lint `0 issues.`, race suite, version injection,
  Windows/Linux/macOS zero-CGO builds, tray build and shipped-file scan.
- Local Linux-target lint: `0 issues.`. Verbose govulncheck v1.7.0:
  `No vulnerabilities found.` across 20 modules including module-only results.
- Standalone gosec v2.29.0 could not decode Go 1.27 export data; used current
  v2.30.0 (Go floor 1.26) instead. `gosec -quiet ./...` exited 0. This is a
  local tooling workaround, not a source or rule exclusion. Tool-only modules
  were not added to the app's go.mod.
- Committed locally as `a832ce4`; not pushed.

### 4: Revision 1.2 Product Documents And History

- New history/product documentation assertions failed first on both root
  documents, all fourteen active prior ADRs, missing new ADRs and old framing.
- Moved the prior handover, blocker and ADRs unchanged under docs/history;
  added the required one-line historical notice. Replaced the withdrawn first
  ADR-015 draft with 015-one-app, added ADR-017 and moved storage to ADR-018.
- Rewrote README, CONTRIBUTING and SECURITY around this product, twelve
  invariants and delivery/safety requirements. Unimplemented guarantees and
  commands are explicitly labelled; no former dashboard screenshots are shown.
- Deleted docs/index.html and docs/screenshots as required. Retargeted existing
  outside-resource, installer-elevation, plain-copy and Sigstore assertions to
  active product docs; replaced the obsolete screenshot requirement with an
  assertion that neither superseded landing page nor screenshots are published.
  B0 must replace that interim absence assertion with real generated-page tests.
- The shipped-file scanner now exempts docs/history, instead of exempting the
  old root handover by name. Runtime scanning and all other checks remain.
- Updated repository description/topics to one app for messages, people and
  sites. Metadata editing runs no workflow and pushed no source commit.
- `go test ./scripts -count=1` and full local `make check` exited 0 with full
  lint, race tests, injection, three zero-CGO targets, tray and shipped scan.
- UI, setup, manifest compatibility and CLI runtime changes remain unchecked;
  no U1 completion claim is made by this documentation pass.

### 5: Test Watcher Ownership Repair Before The Move-Only Commit

- Pre-move full local `make check` was green. Mechanically moved 27 complete
  package trees and adjusted import/linker/source/fixture paths only.
- First post-move full race run failed `TestSSEStreamSurvivesThreeAppChanges`:
  `no apps-changed event for change 3` after 4.29 seconds. Full lint passed.
- This is treated as a failing flaky test, not accepted by retrying. Reversed
  only the uncommitted mechanical move and verified a clean worktree before
  recording this incident. Diagnosis/repair will be a separate [U1] commit so
  the eventual layout commit contains no behavior changes. No deadlines or
  event assertions have been weakened. No GitHub Actions were run.
- Diagnostic stress could not reproduce the original SSE timeout: isolated
  30/30, mixed live-watcher 50/50, traced mixed 50/50 and wider shuffled 15/15
  passed. Its exact cause remains unproven and is retained as a timing risk.
- Native Windows traces did prove orphaned test watchers falling back to the
  shared TEMP directory after their owned roots were removed. The independent
  ownership audit found 11 never-closed servers plus two fatal-path gaps.
- Added immediate cleanup ownership to all 13 constructor results across
  dashboard, preference and trust-lifecycle tests. Preserved explicit early
  Close calls and every assertion/deadline. Production code is unchanged.
- An invalid concurrent verification attempt ran two full fixture processes;
  they reached each other's command apps and failed on wrong response bodies.
  The 7400-7999 reservation map is process-local. Run whole suites sequentially,
  preserving each suite's usual internal parallelism; do not change test limits.
- With only one fixture suite active, full `make check` exited 0; shuffled
  `go test -race -count=3 -shuffle=on ./internal/server` exited 0 (59.177s).
- Full native-traced race suite exited 0 with zero `AddWith` calls for shared
  TEMP. This verifies the demonstrated watcher leak was removed, not that the
  original one-off SSE timeout's causality has been conclusively established.
- Committed separately as `b395409`, without runtime changes or a push.

### 6: Behavior-Preserving Package Relocation

- Full local `make check` passed immediately before and after the move, with
  full lint, the complete race suite, version injection, all three zero-CGO
  targets, optional tray and the shipped-file scan unchanged.
- Moved 27 complete package trees: hosting code to internal/sites, shared OS
  code to internal/platform, discovery to internal/links/discovery and the
  current dashboard/preferences to internal/ui. Package declarations and public
  APIs are unchanged. Kept embedded assets, platform files, tests and fixtures.
- Existing sites/app remains the hosted-site model; sites/supervisor remains
  the hosted-child-process manager. Neither is the future app supervisor.
- Updated only imports, linker symbols, fixture/source paths and the fuzz
  workflow's package targets. Editing workflow source did not execute it.
- A read-only mechanical verifier compared all 274 tracked files against HEAD
  with exactly the declared path transformations and gofmt. It found no other
  content changes; resources are byte-identical. This STATE entry is the sole
  additional non-path bookkeeping change in the refactor commit.
- The moved server race suite passed in 37.414s. No GitHub Actions, source
  pushes, OS registration changes or dependency changes were performed.
- The executable still has the earlier entry-point behavior until the next
  U1 iterations; this move is not app-supervisor or unified-shell completion.
- Committed locally as `2c9fcbf`, after the separate `b395409` ownership fix.

### 7: Independent Application Supervisor Core

- Added failing tests in the newly free internal/app package first; compilation
  failed on absent Service, Options and Supervisor definitions as expected.
- Added a stdlib-only supervisor requiring exactly sites, links, events and ui.
  Each service has its own cancellation context, panic boundary, bounded
  exponential retry delay, status facts and independent lifecycle controls.
- Cancellation joins all service invocations. Restart does not overlap the old
  and replacement invocations. Normal shutdown is not counted as a failure;
  unexpected clean exits are failures, and sibling failures never cancel peers.
- Service entry points must honor cancellation and join their internal workers;
  worker failures must return to the supervised entry point, not panic in an
  unowned goroutine. Lifecycle state is explicitly not reach/readiness evidence.
- `TestServiceFailureIsContained` injects a panic into each of four service
  areas and verifies restart without restarting/failing the other three.
  Other tests cover independent controls, retry timing, cancellation of a
  one-hour backoff, invalid setup, concurrent Run rejection, early clean exits,
  constructor/snapshot ownership, and already-canceled parent startup.
- `go test -race -count=100 -shuffle=on ./internal/app` exited 0 (14.224s).
  Full local `make check` exited 0 with full lint, complete race suite, injection,
  zero-CGO targets, tray and shipped-file scanning. No dependencies were added.
- This is the supervisor core, not command-entry-point integration: actual
  services are not yet wired through it. The corresponding U1 integration
  criteria deliberately remain unchecked until cmd/beacon starts all four.
- Independent lifecycle review found no concrete defect, but identified missing
  cleanup-joining and exponential-policy proofs. Added blocked-cleanup tests
  proving no overlapping replacement, concurrent Stop/Start while draining,
  and no application return before final cleanup. Extracted the pure backoff
  policy to test growth, saturation, healthy-run reset and overflow bounds
  without wall-clock-dependent assertions. The real timer delay remains tested.
- Local native/Linux standalone gosec and module security checks were clean in
  the prior gate repair. No GitHub Actions or pushes were run in this iteration.

## Decisions Made This Build

- Revision 1.2 is authoritative. Earlier product architecture decisions and
  milestones do not prescribe this app. Keep factual reuse and attribution.
- The first ADR-015 draft is withdrawn and replaced. Only legacy manifest
  reading remains a compatibility requirement.
- No GitHub Actions. Use local native/cross-target lint, race tests, security
  tools, browsers, installer tooling or local VMs where available.

## Open Human Work

- No question blocks current U1 work. Eventual release secrets, relay-list
  approval, Banglish read-aloud, lighthouse deployment and field proofs stay
  human-owned under section 13; none is claimed complete.

## Dependencies

- 10 direct modules; no direct dependency added. x/net v0.62.0 and x/sys v0.49.0
  require Go 1.26.0 (locally running Go 1.27.4).

## Next Target

Wire cmd/beacon through the new supervisor and four real service entry points,
without nesting services. Keep the service adapters' resources and cleanup
under their own Run contexts. Follow with unified config, identity first run,
shell, CLI and manifest compatibility. No B0 features until U1 passes. Read this
ledger and run `make check` at each iteration start. Run fixture suites
sequentially. Do not push or run GitHub Actions.

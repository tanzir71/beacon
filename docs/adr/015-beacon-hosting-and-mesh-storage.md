# ADR-015: Beacon Hosting Engine And Mesh Storage

Date: 2026-11-24. Status: accepted. Supersedes: ADR-005 for mesh data only.

## Context

Blackout Beacon builds on Dropserve at commit `1c1e433`, retaining its history,
MIT attribution, hosting architecture, product invariants, and full build gate.
Beacon must coexist with Dropserve without reading, replacing, or deleting its
installation, state, configuration, operating-system registrations, or keys.
The planned signed event log, address book, and resumable transfer state need
transactional storage that the small hosting configuration does not.

## Options Considered

1. Modify the existing Dropserve product in place. This risks installed-user
   data and contradicts the required independent product identity.
2. Fork with preserved history and isolate every owned identifier. This keeps
   the tested hosting engine while making install and removal independent.
3. Keep all mesh data in JSON. This avoids a database but cannot provide the
   transactional querying and event reconciliation required by the handover.

## Decision

Use `github.com/tanzir71/beacon` with `beacon` and `beacon-cli` binaries, Beacon
storage directories, a new Windows installer AppId, the `Beacon` task/firewall
name, `beacon.service`, and `dev.beacon.agent`. Do not migrate Dropserve state.
The dashboard owns the separate `/_beacon` namespace and Beacon session/CSRF
identifiers. Configuration and hosting snapshots keep their atomic TOML/JSON
formats. Mesh events, the address book, and transfer state will use the existing
pure-Go `modernc.org/sqlite` dependency, with encryption added and verified in
B2 before any private-message storage ships. This ADR is not evidence that
encrypted messaging or a mesh database is already implemented.

Keep the shipped hosting-engine app contract: `dropserve.json` manifests and
the `DROPSERVE_BASE_PATH` / `DROPSERVE_BASE_URL` child environment variables.
These belong to user apps, not the install identity. Existing apps can already
depend on them; changing them would force edits inside user-owned folders.
No fallback to another product's state, config, certificates, or OS registration
is introduced. Original ADRs 001-014 remain unmodified historical records.

## Consequences

Beacon and Dropserve can be installed independently; uninstallation must target
only Beacon-owned paths and registrations. New identity tests enforce module,
build, storage, task, service, LaunchAgent, and installer separation. A separate
Windows side-by-side lifecycle proof is required before B0 completion.
The shared external Tailscale configuration is not an owned installation path;
its existing explicit-consent behavior remains and no Tailscale configuration
is changed during this bootstrap. The mesh dependency cap exception does not
apply to unrelated hosting dependencies. No dependencies are added by this ADR.

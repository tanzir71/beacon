# ADR-015: One App, Specified On Its Own Terms

Date: 2026-11-24. Status: accepted.

This replaces a withdrawn first draft, `015-beacon-hosting-and-mesh-storage.md`.
The draft treated hosting as the foundation of another subsystem. That framing
is withdrawn by handover revision 1.2, which is the sole product specification.
The draft remains in Git history, not as a current architecture decision.

## Context

Blackout Beacon is one everyday app for messages, people and sites that works
over the internet, its own network, both, or off-grid. Reused code is useful
material, not a product boundary or a source of requirements. A web-server-first
entry point and a launchpad with messaging attached cannot express this product.

## Decision

`cmd/beacon` starts an application supervisor in `internal/app`. Four services
have equal standing: `sites`, `links`, `events` and `ui`. Each starts, stops and
restarts independently with backoff; none runs inside another. A service panic
must not stop the other three. There is one identity, configuration, state
directory, HTTP port, shell and command-line tool.

Move reused hosting models, routing, indexing and process management under
`internal/sites`; shared OS code belongs in `internal/platform`. The existing
hosted-process supervisor is not the application supervisor. The move-only
commit preserves behavior and passes the gate on both sides; behavior changes
follow as distinct test-first commits.

Home is Reachable now, initially with On this computer and optional Add a folder.
Hosting controls belong in Sites; the app does not require a site to be useful.
The old launchpad and folder-centred first-run flow are removed. The CLI is
organised by sites, people, messages, network and bundle, plus global commands.

Separate OS/storage identifiers remain mandatory for coexistence with an
earlier installation. No automatic migration writes to that installation.
An optional import reads settings and folder references only. Site manifests
use `beacon.json`; a legacy-only manifest can be read, never written. Child
environment output uses Beacon names. Prior specifications and ADRs are kept
under `docs/history/` for attribution and explanation, not as authority.

## Consequences And Verification

U1 precedes further B0 work. I1-I5 retain their meaning, I9 adds four-service
panic containment, and assertions must reject old product strings in output.
This ADR records direction, not completed functionality; `STATE.md` holds proof.
Delivery is decided separately in ADR-017 and storage in ADR-018. No dependency
is introduced by this decision.

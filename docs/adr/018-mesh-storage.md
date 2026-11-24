# ADR-018: Mesh Storage

Date: 2026-11-24. Status: accepted; encrypted storage must be verified in B2.

## Context

Site configuration and preferences are small atomic snapshots. Events, signed
address records and resumable transfer state need transactions, indexes and
reconciliation. Giving one format all these roles would either complicate
simple configuration or make event recovery fragile.

## Decision

Keep site configuration and preferences as atomic JSON snapshots. The one app
configuration exposes sites, people, messages, network and safety sections.
Use pure-Go SQLite for events, the address book and transfer state, under the
same application state directory as the installation identity.

Reuse `modernc.org/sqlite` v1.57.0, already present, rather than introducing a
CGO-backed driver. This is a product data requirement, not a decision inherited
from another app. The duplicate storage discussion in the withdrawn ADR-015
first draft is superseded by this document.

Private event storage must be encrypted at rest using supported libraries and
documented constructions selected during B2. Do not claim that the current
SQLite dependency provides encryption by itself. Relays/mailboxes persist only
gift-wrapped ciphertext. A known plaintext string must not occur in the store,
and opening with a wrong key must fail. Atomic site preferences are not a
message store and must never contain private-message plaintext.

## Consequences And Verification

Cross-build every target with CGO_ENABLED=0. S2 must demonstrate the selected
event relay against a pure-Go store; S3 must prove reconciliation after a
partition. Check encryption, key permissions, restart identity and duplicate
imports separately. No new direct dependency is introduced by this ADR; any
later library needs its own recorded version, Go floor and license review.

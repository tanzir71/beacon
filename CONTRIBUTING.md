# Contributing To Blackout Beacon

This is one app specified by the current Blackout Beacon handover, not by any
earlier project's documents. Read that specification and STATE.md. Historical
material in docs/history explains code reuse but is not product authority.

## Twelve Invariants

| ID | Required behavior | Proof begins |
|---|---|---|
| I1 | A folder becomes a reachable site within two seconds, with no manifest or restart. | U1: TestDropFolderBecomesReachable |
| I2 | Never create, change or delete a file inside a site folder. | U1: retain the existing read-only-folder tests |
| I3 | Every address shown by the interface resolves. | U1: TestAdvertisedURLsAllRespond |
| I4 | A broken site cannot stop another site. | U1: TestOneBrokenAppDoesNotAffectOthers |
| I5 | No exposure beyond LAN or OS trust installation without a specific action in that session. | U1: TestNoImplicitPublicExposure and TestTrustStoreRequiresExplicitConsent |
| I6 | Every beyond-LAN dial has a signed introduction or explicit action; refuse all others. | B4: TestNoDialWithoutIntroduction |
| I7 | Enforce the complete shared traffic budget even under hostile input. | B5: TestTrafficBudgetHolds |
| I8 | Internet is optional; endpoints come only from signed, visible, editable lists. | B1: TestWorksWithInternetPathAbsent and TestNoHostLiteralsOutsideSeedLists |
| I9 | A failure in sites, links, events or interface cannot stop the other services. | U1: TestServiceFailureIsContained |
| I10 | Relays and mailboxes hold ciphertext only; private-message plaintext stays on device. | B2: TestMailboxStoresOnlyCiphertext |
| I11 | Delivery, reach and reachable labels reflect fresh stored facts. | B2: TestShownStateMatchesStoredState |
| I12 | Offer messages to all allowed working paths and show each event once. | B2: TestMultipathDeliversOnce |

I5 applies to sites, lighthouse roles, router port mapping and square listings.
Do not treat a relay acknowledgment as device delivery or silence as reach.

## Iterative Development

Read STATE.md and run `make check` first. A red gate makes repair the entire
iteration. Otherwise choose the lowest unmet criterion, write the failing test,
make the smallest passing change, run the whole gate, record exact evidence,
and commit one logical change, for example `feat(app): isolate services [U1]`.
After three failing iterations record a blocker and work on an independent
criterion. A flaky test is a failure; no retries disguised as acceptance.

Preserve meaning of the hosting invariants while fitting reused code into
internal/sites. Keep a move-only commit separate from behavior changes.
Never rewrite history, commit secrets, write inside site folders, invent crypto,
bypass the policy dialer or add runtime host literals outside signed seed lists.
Document open product decisions in ADRs; library additions need current version,
Go floor and license review. Claim features only after their proofs pass.

Follow the owner's current instruction to run no GitHub Actions. Use full local
lint, race, security, browser and cross-build checks. Do not dispatch, rerun, or
push commits that trigger workflows. Record unavailable external proofs rather
than inventing them or declaring a milestone complete.

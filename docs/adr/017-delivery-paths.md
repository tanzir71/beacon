# ADR-017: Delivery Paths

Date: 2026-11-24. Status: accepted; implementation belongs to B2 and B5.

## Context

People should use the same app on an ordinary day and during a partial or total
internet outage. Requiring internet infrastructure breaks that promise; refusing
available internet paths makes everyday delivery unnecessarily unreliable.

## Decision

Use two independent path families: internet relays and introduced internet links;
and Beacon-network links, lighthouses, mailboxes and offline bundles. Internet
endpoints are optional and come only from signed, visible, editable seed lists.
They never bypass the policy dialer or its traffic budget. No DHT, endpoint
crawling, runtime host literals or built-in external bootstrap peers are allowed.

In `automatic`, offer a message to every allowed path that is up, record each
attempt/outcome, and count the first confirmed device arrival. Deduplicate at
the receiver by signed event ID. `internet` and `beacon` restrict the family.
Cautious mode implies `beacon` and mailbox-only; never dial a contact directly.
No more than four long-lived internet relay connections may be active.

For sites and files choose the fastest working path, falling back in order:
LAN, direct internet, Beacon-network link, relay, saved copy. Evidence drives
reach labels; never probe unrelated public hosts. A relay or abroad-flagged
lighthouse response can establish Internet reach. Never promote a stale or
unconfirmed attempt to Delivered, and never infer device arrival from relay
acceptance. Store first-confirmed and all-attempt paths for honest labels.

## Consequences And Verification

I8 means nothing outside the user's network is required, not that internet use
is forbidden. I12 requires multipath sending and once-only presentation. Test
all internet behavior with endpoints blocked and open; queued events leave when
an allowed path returns. Relay operators see address/recipient metadata and
ciphertext, not private-message plaintext. S7 must establish gift-wrap support
and retention before a default list is proposed for human approval.

This ADR adds no dependency and does not assert implemented delivery or approved
seed infrastructure. Protocol and encryption libraries are selected in the
specified spikes; no cryptographic construction is implemented here.

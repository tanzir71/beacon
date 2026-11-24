# Security Policy

Report vulnerabilities privately to tanzir71@gmail.com or the repository's
private security advisory form. Include version, OS, reproduction steps and
impact. Do not publish an unpatched vulnerability in a public issue.

## Scope And Current Status

Blackout Beacon is one app for people, messages and sites. U1 is in progress.
Message encryption, mailbox delivery, panic wipe, phone identities and the
internet-path safety model described below are not implemented yet. These are
requirements, not current security guarantees. The app is not a public hosting
platform, account service, anonymity network or state-adversary concealment tool.

## Message And Device Safety Requirements

Private messages use library-provided NIP-17, NIP-44 and NIP-59 constructions,
with no forward secrecy in this ladder. Relays and mailboxes keep ciphertext
only; local event storage is encrypted at rest. A six-digit PIN controls optional
non-loopback access. A phone's identity key and messages are held on its nearby
computer; the operator of that computer can read them. Join must disclose this.

Key files require owner-only access. Backup and restore must reproduce the
identity. A panic wipe requires typed confirmation and removes keys/database,
while disclosing that secure erasure of SSD pages, backups and other people's
copies cannot be guaranteed. Cautious mode uses Beacon-network mailboxes only,
never direct contact dials or internet relays.

## Network And Metadata Requirements

An internet relay can see the recipient key, sender address and encrypted text.
A provider can see that Beacon runs and the addresses it contacts, not private
message content or which sites are opened through encrypted links. Beacon does
not disguise traffic. Ordinary TLS 1.3 and QUIC are used.

Every beyond-LAN connection needs signed introduction or explicit user action.
There is no address scanning outside the user's subnet, no public DHT and no
relay crawling. Signed editable lists are the only internet endpoint source;
internet paths remain optional and use the same policy budget. At most four
internet relay connections may be held open. Delivery labels require evidence;
mailbox acceptance is not device receipt. Deduplicate multipath events by ID.

## Hosting And Exposure Boundaries

Site folders are read-only to Beacon. Static-file path checks, same-origin/CSRF
validation, PIN sessions and bounded requests remain tested. Hosted app code
runs with the signed-in user's privileges: adding untrusted code is equivalent
to running it, not putting it in a security sandbox.

Exposure past LAN, lighthouse roles, public listings, router mappings and OS
trust installation require explicit, specific consent in the current session.
Creating a local CA must not install it into the OS trust store. Peer-site
gateways must use distinct browser origins, strip credentials, set sandbox
headers, reject service workers and cap responses before they ship.

Only Beacon-owned install/config/state paths and OS registrations may be
removed. An optional import reads earlier settings and folder references but
does not change the source installation or site content.

## Distribution

Release checksums, SBOMs, provenance, Sigstore bundles and Authenticode provide
integrity evidence, not trust in arbitrary hosted code or SmartScreen reputation.
No Beacon release is complete yet. Fixes are maintained on main during pre-1.0
development. Do not claim a signed or externally verified release without the
corresponding artifact and successful verification command.

# Blackout Beacon

[Website](https://tanzir71.github.io/beacon/) · [English](https://tanzir71.github.io/beacon/en/) · [Banglish](https://tanzir71.github.io/beacon/bn/)

Blackout Beacon is one app for messages, people and sites. It runs on a computer
without an account, phone number or required server. It is designed to deliver
over the internet when available, over Beacon's own network when it is not, and
over both when both work, continuing off-grid when neither is available.

The product finds people through introductions rather than address-space scans,
carries end-to-end encrypted messages, turns folders into websites, and lets
contacts open one another's sites. Phones join a nearby computer in a browser
by QR, with no installation. These describe the target product, not a claim
that every feature below is implemented.

## Current Development Status

**U1: the app's own skeleton is in progress.** The repository currently contains
reused hosting code that is being fitted under one application supervisor,
configuration, identity, shell and CLI. Links, messages, phone joining,
lighthouses, saved copies and USB delivery are not released. No Beacon milestone
or binary release is complete. See [STATE.md](STATE.md) for verified commands
and open acceptance criteria.

The application has four equal services: sites, links, events and interface.
Home is Reachable now; a folder becoming a website is one feature, not the
identity of the program. Earlier project records are archived in
[docs/history/](docs/history/) and do not specify this app.

## Build And Check

Repository constant: `REPO_SLUG=tanzir71/beacon`.

Use Go 1.26 or newer, GNU Make, and golangci-lint. Race tests require a supported
host C compiler, but every shipped binary is built with CGO_ENABLED=0.

```console
git clone https://github.com/tanzir71/beacon.git
cd beacon
make check
make build
```

The gate checks formatting, lint, race tests, version injection, zero-CGO
Windows/amd64, Linux/amd64 and macOS/arm64 builds, optional tray and unfinished
shipped text. The portable fallback is go vet if golangci-lint is unavailable;
development acceptance uses the full linter and standalone security checks.
Do not weaken checks to get a passing build.

The current development entry point is `go run ./cmd/beacon`; U1 is replacing
its folder-centred setup with a display name and an optional site-folder choice.
The planned CLI is organised as `beacon sites`, `beacon people`,
`beacon messages`, `beacon network`, `beacon bundle`, `beacon status`,
`beacon doctor` and `beacon version`. Unimplemented commands are not advertised
as usable. The bilingual website is a development preview, not a claim that
planned messaging or off-grid features have shipped. `make landing` regenerates
the root, English and Banglish Pages entrypoints from one template and local
copy dictionaries. Pages publishes `main /` with no external font or script CDN.

## Safety And Honest Limits

The intended message encryption has no forward secrecy in this milestone
ladder. Internet relays see recipient keys, source addresses and ciphertext;
network providers can see that Beacon runs and the addresses it contacts.
Phones keep their keys and messages on the host computer, whose operator can
read them. These screens and protocol guarantees must be verified before the
corresponding features ship.

Hosting runs local app code with the computer user's privileges, not inside a
security sandbox. Site folders are never written. Exposure beyond the LAN,
router mappings, lighthouse roles and OS trust installation require explicit,
specific consent in the current session. See [SECURITY.md](SECURITY.md).

## Release Verification

There are no downloadable Beacon binaries yet. The release design includes
SHA-256 checksums, SPDX SBOMs, provenance, Authenticode and Sigstore bundles.
For a future published asset and its matching bundle, verify with:

```sh
cosign verify-blob \
  --bundle "$ARTIFACT.sigstore.json" \
  --certificate-identity-regexp '^https://github.com/tanzir71/beacon/.github/workflows/release.yml@refs/tags/.*$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  "$ARTIFACT"
```

This is an integrity check, not a guarantee of safety or SmartScreen reputation.
The current Windows installer build requires elevation: Windows asks once to
install into Program Files and register a private-network firewall rule. A new
self-signed build can show SmartScreen warnings; check its checksum and signing
identity rather than assuming a warning-free installer or turning protection off.
Development currently runs validation locally; automated workflows are not run
under the owner's current instruction.

## Contributing And License

Read [CONTRIBUTING.md](CONTRIBUTING.md), the current build ledger and the
authoritative Blackout Beacon handover before changing the code. This app is
MIT-licensed; existing notices and Git attribution are retained in [LICENSE](LICENSE).

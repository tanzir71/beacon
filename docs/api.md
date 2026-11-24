# Beacon local API

**U1 transition:** this lists the current reused site's endpoints, not a settled
API for the unified app. U1 will replace the launchpad with the app-owned shell
and document its local API once that contract is implemented and tested. People,
messages, network delivery and guest identities are not implied by these routes.

The dashboard uses this API, and local tools may use it as an integration surface. It is available beneath `/_beacon/` on every active Beacon HTTP or HTTPS listener.

## Read-only endpoints

| Method | Path | Result |
|---|---|---|
| `GET` | `/_beacon/api/apps` | Visible app index. Add `?include_hidden=1` to include hidden cards. |
| `GET` | `/_beacon/api/apps/{slug}` | One app with detection details and warnings. |
| `GET` | `/_beacon/api/search?q=…` | Ranked, visible app matches. |
| `GET` | `/_beacon/api/urls` | Only currently verified listener, LAN, mDNS, and Tailscale addresses. |
| `GET` | `/_beacon/api/status` | Version, uptime, ports, discovery/sharing state, warnings, update notice, and CSRF token. |
| `GET` | `/_beacon/api/logs/{slug}` | Command-app snapshot as JSON; request `Accept: text/event-stream` for live `logs` events. |
| `GET` | `/_beacon/api/events` | Server-sent `apps-changed` events. |
| `GET` | `/_beacon/api/qr?url=…` | Local PNG QR code for an HTTP(S) URL. |
| `GET` | `/_beacon/api/databases/{slug}?file=…` | Read-only SQLite tables and the first 100 rows for a database discovered in that app. |
| `GET` | `/_beacon/api/addons` | Optional PHP, MariaDB, and PostgreSQL installation/process status. |
| `GET` | `/_beacon/api/https/root.pem` | The local root certificate, after one has been generated. |
| `GET` | `/_beacon/healthz` | Plain `ok` while the server is ready. |

## Mutations

| Method | Path | JSON body |
|---|---|---|
| `POST` | `/_beacon/api/apps/{slug}/restart` | `{}` |
| `POST` | `/_beacon/api/apps/{slug}/settings` | `{"pinned":true}` and/or `{"hidden":true}` |
| `POST` | `/_beacon/api/rescan` | `{}` |
| `POST` | `/_beacon/api/open-folder` | `{}` for the Apps root or `{"slug":"notes"}` |
| `POST` | `/_beacon/api/addons/{name}` | `{"action":"install"}`; actions are `install`, `remove`, `start`, or `stop`. |
| `POST` | `/_beacon/api/https` | `{"enabled":true}` |
| `POST` | `/_beacon/api/trust` | `{"installed":true}` |
| `POST` | `/_beacon/api/network-change/dismiss` | `{}` |
| `POST` | `/_beacon/api/sharing/tailscale` | `{"enabled":true}` |
| `POST` | `/_beacon/api/sharing/funnel/{slug}` | To start: `{"enabled":true,"confirmation":"<exact slug>"}`. To stop: `{"enabled":false}`. |

Every mutation requires both:

1. `X-Beacon-CSRF: <csrf_token>` using the current token returned by `GET /_beacon/api/status`.
2. An `Origin` whose scheme and authority exactly match the Beacon request URL.

When PIN lock is enabled, a non-loopback client must first authenticate at `/_beacon/login`; the API then also requires its signed session cookie. The local `beacon` CLI obtains the current token and sends the matching origin automatically.

Responses are intentionally local-machine APIs, not a versioned public cloud contract. Clients should ignore fields they do not recognize.

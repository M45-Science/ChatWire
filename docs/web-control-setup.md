# Local moderator web controls

The web service is a separate Go executable that serves plain HTTP on a loopback
listener (the example uses `127.0.0.1:8787`). It does not load certificates or
enable HTTPS; nginx handles TLS for browser connections. Existing ChatWire processes only
open a private Unix socket when started with `-webControlConfig`. No web listener
is enabled by default. The dashboard controls the local instances listed in an
explicit registry.

## Configure the host

Build `./cmd/chatwire-web` and the normal ChatWire executable using the project's
Go version. Install them during your normal maintenance/deployment window. Example
configuration and a web-service unit are in `example-files/cw-web-config.json`,
`example-files/cw-web-instance.json`, and `example-files/chatwire-web.service`.
These are templates; replace their paths, user, guild, role IDs, and instances.

Run the web service and ChatWire instances as the same dedicated Unix user. Socket
directories must not be accessible to other users. Sockets and credential files
use mode 0600; runtime directories should use 0700. Provision a separate random
credential for each instance and another for the primary's login broker. For
example, `openssl rand -hex 32` generates suitable credential content; write it to
the configured protected file, not a command-line argument or public config.

For each instance, add the following to its existing launch command:

```text
-webControlConfig /etc/chatwire-web/instance-a.json
```

Provide a per-instance runtime directory such as `/run/chatwire-a` through the
service manager, and a persistent state directory such as
`/var/lib/chatwire-control/a`. The instance ID must match the web registry. Only
the configured primary needs `broker_socket` and `broker_credential_file`; other
instances may leave them blank. The web service's `primary` must identify the
same instance as ChatWire's global `PrimaryServer` setting.

The credential paths in an instance config and its registry entry refer to the
same file/content. The web service's broker credential matches the primary's
broker credential. The server uses an advisory ownership lock and safely reclaims a stale socket
after its old process exits. It refuses to remove a live listener or a non-socket
file. Configure systemd runtime directories for socket lifecycle management.

Validate the web registry and credential files without opening listeners:

```text
chatwire-web -config /etc/chatwire-web/cw-web-config.json -check-config
```

Run the web executable with the same `-config` argument through your service
manager. Put nginx or another HTTPS reverse proxy in front of its HTTP listener. The
`public_origin` must be the exact external HTTPS origin, with no path or trailing
slash. The application checks Origin directly and does not trust forwarded
headers for authentication. Preserve Origin and cookies in the proxy.

For nginx, adapt `example-files/chatwire-web.nginx.conf` with your hostname and
certificate paths. Keep `listen` set to `127.0.0.1:8787` and set `public_origin`
to your browser-facing URL, for example `https://chatwire.example.org`.
`public_origin` controls login links and Origin checks; it does not enable TLS on
ChatWire's listener. Secure cookies work through nginx even though the upstream
connection uses HTTP. No forwarded-header trust setting is required.

Allow streamed responses for `/api/v1/events` with buffering disabled. Set upload
limits/timeouts to support your chosen saves (application maximum: 512 MiB per
save, 16 MiB per mod-settings file, 1 MiB per mod-list). Keep the private Unix
socket APIs off the public proxy. Regular browser access requires HTTPS because
session cookies are Secure.

Register slash commands using ChatWire's existing `-regCommands` workflow after
installing the updated binary. Deployment, registration, and process restarts
are operator actions; editing these source files does not perform them.

## Moderator login

Run `/web` in the configured guild. The primary verifies your moderator/admin
role and replies ephemerally with an **Open ChatWire** button. The single-use link
expires after two minutes. On the landing page, choose **Continue to dashboard**.
The GET/preview does not consume the link; exchange occurs on POST.

Run `/web action:logout-all` to revoke all your sessions and unused login links.
Normal sessions have a 30-minute idle limit and an eight-hour absolute limit.
Discord membership/roles are refreshed at least once per minute while accessing
the API. Access fails closed if membership cannot be verified. A web-service
restart invalidates existing sessions and unused links. Each link is a temporary
bearer credential, so do not share it.

The public interface has no password or OAuth-client setup. It still needs the
existing Discord bot and primary instance for new grants and authorization
refreshes. `-noDiscord` and `-localTest` never disable web authentication.

## Controls and API behavior

The dashboard includes the all-server overview, per-instance settings/actions,
online and shared player lists, saves and upload staging, mod editing/history,
RCON/logs, host firewall rules, and job activity. Administrators also get shared settings and host
registry editing. The overview supports explicit target lists for RCON-all and
config-reload batches. Batch results are independent per target.

The public API is rooted at `/api/v1`; see `docs/web-control-openapi.json` for the
implemented routes. Settings schemas come from the instance. Reads never return
bot/Factorio tokens or RCON passwords. Admin-only secret fields are write-only;
a patch with an empty string explicitly clears a stored secret.

Settings patches use `If-Match` revisions and are validated before atomic
persistence. A three-way merge protects web edits from an unrelated stale runtime
write. Settings represent desired values on disk. Reload configuration explicitly
using the instance action or an all-instance config-reload batch. Fields marked
for a Factorio/ChatWire/web restart or next map need that additional action; saving
them does not initiate a restart. Check each instance's status and batch results
when applying shared changes. The host registry is only applied on web restart.

Action submission requires a preview token and an `Idempotency-Key`. Preview
includes the exact target and player count. Repeating the same key and payload
returns the existing job; reusing it for another payload is rejected. A response
lost during submission must be reconciled using the same key or job resource.
Do not automatically generate a new key after a network error.

Jobs persist before execution. On instance restart, unfinished operations become
`unknown` rather than being replayed. ChatWire reboots have a special persisted
exit checkpoint which the new process marks reconnected. Start/restart/map-load
jobs wait for Factorio readiness before success. A staged upload stops Factorio
before applying files; on activation failure it reports failure and leaves the
server stopped. Activation of multiple files can partially complete; inspect the
job result and files before retrying. Staged uploads expire after one hour and
are discarded on instance restart.

SSE currently sends bounded snapshot-invalidation notifications every five
seconds; reconnecting clients refetch snapshots. It does not replay raw logs or
claim to be a durable event stream. Logs return a bounded recent tail; job lists
return the latest 100 operations. Shared players and saves use bounded pages.
Config changes retain the existing runtime's reload semantics; a process-wide
migration to immutable config snapshots is separate from this HTTP adapter.

The dashboard stays available when an instance reboots. If ChatWire itself is
stopped, it reports offline and its service manager must recover it. The HTTP
server does not run arbitrary shell commands or start stopped systemd units.

## State and audit

The web service stores aggregate jobs and a credential-free login/admin audit in
its `state_dir`. Each instance stores jobs and upload staging in its own
`state_dir`, plus the existing ChatWire audit log. Protect these directories:
RCON output and moderator results may contain sensitive operational information.
Back up job records if preserving idempotency across host recovery is important.
Job history and audit files currently require operator retention/rotation.

Enable the control endpoint on all instances sharing game files so their
updaters, mod sync, and map generators participate in cross-process file locks.
Existing services continue to own their Factorio process. Discord and web user
commands share command admission, and both use the same player moderation,
firewall, RCON, reload, and lifecycle services.

## Validation

The test suite uses temporary directories/sockets and fake HTTP peers. It covers
single-use/expired login grants, concurrent redemption, revocation races,
CSRF/admin checks, private credentials, instance routing/identity, durable action
idempotency, stale settings conflicts, preservation of pending web edits, and
unsafe upload paths. Run `go test ./...`, `go vet ./...`, and targeted race tests
for `./webapi ./webcontrol ./controlruntime ./cfg` before deployment. Live Discord,
Factorio, UFW, reverse-proxy, and browser interactions require deployment testing.

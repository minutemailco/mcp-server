# mcp-server

MCP (Model Context Protocol) server exposing the MinuteMail public API as
tools. It lets AI agents and MCP clients operate MinuteMail mailboxes, emails,
attachments, domains, teams, and mock OAuth identities through the
`mm-api-gateway`.

- **Endpoint:** `https://dev.mcp.minutemail.co/mcp` (Streamable HTTP,
  protocol revision `2025-06-18`)
- **Auth:** the MCP client's `Authorization: Bearer mmak_...` header is
  forwarded verbatim to the api-gateway — this server holds **no credentials**.
  Scopes and quotas are enforced by the gateway exactly as for any API client.
- **Stateless:** no sessions, no server-push SSE (`GET /mcp` and `DELETE /mcp`
  return 405). Health: `GET /health`.

## Tool surface (39 tools)

Every tool maps 1:1 to a `/v1` route of the api-gateway. All calls require the
Bearer API key; write operations require the matching scope
(`mailboxes`, `domains`, `team`, `identities`).

### Mailboxes (`mailboxes` scope)
| Tool | Gateway route |
|---|---|
| `mm_list_mailboxes` | `GET /v1/mailboxes` |
| `mm_create_mailbox` | `POST /v1/mailboxes` (owner forced to key owner, domain defaults to tenant default) |
| `mm_get_mailbox` / `mm_delete_mailbox` | `GET` / `DELETE /v1/mailboxes/{id}` |
| `mm_bulk_delete_mailboxes` | `DELETE /v1/mailboxes` |
| `mm_list_mails` / `mm_send_mail` | `GET` / `POST /v1/mailboxes/{id}/mails` (multipart) |
| `mm_get_mail` / `mm_delete_mail` / `mm_bulk_delete_mails` | `…/mails/{mailId}` |
| `mm_list_attachments` / `mm_add_attachment` / `mm_get_attachment` / `mm_delete_attachment` / `mm_bulk_delete_attachments` | `…/attachments` |

### Archived mailboxes (`mailboxes` scope)
`mm_list_archived_mailboxes`, `mm_get_archived_mailbox`,
`mm_delete_archived_mailbox`, `mm_reactivate_archived_mailbox`.

### Domains (`domains` scope)
`mm_list_domains`, `mm_register_domain`, `mm_verify_domain`, `mm_delete_domain`.

### Team (`team` scope)
`mm_list_members`, `mm_add_member`, `mm_get_member`, `mm_delete_member`,
`mm_create_invitation`, `mm_list_invitations`, `mm_delete_invitation`.

### Mock identities (`identities` scope)
`mm_list_identities`, `mm_create_identity`, `mm_get_identity`,
`mm_delete_identity`, `mm_list_oauth_clients`, `mm_create_oauth_client`,
`mm_get_oauth_client`, `mm_delete_oauth_client`, `mm_rotate_client_secret`.

### Not exposed
- `/v1/api-keys` CRUD — exempt from API-key auth (web-gateway internal plane,
  trusts `X-Internal-Tenant-Id`); manage keys from the web app instead.
- `/internal/*`, `/metrics` — internal-only.

## Example

```bash
curl -s https://dev.mcp.minutemail.co/mcp \
  -H 'Authorization: Bearer mmak_XXXXXXXX' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call",
       "params":{"name":"mm_create_mailbox","arguments":{"expiresIn":30}}}'
```

Tool results are returned as MCP text content carrying the gateway's JSON
response. Non-2xx gateway responses become `isError: true` results with the
HTTP status and body (401 invalid key, 403 scope/domain, 429 quota with
remaining count, 502 upstream down).

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | Listen port |
| `API_BASE` | `http://mm-api-gateway:80` | api-gateway base URL (in-cluster) |
| `LOG_LEVEL` | `warn` | `debug`/`info`/`warn`/`error` |
| `LOG_FORMAT` | `json` | `json`/`text` |
| `PROFILE` | `dev` | Deployment profile label |

## Development

Go 1.22, stdlib only (no dependencies). Follows the MinuteMail service
standards (see mm-mailbox-service).

```bash
go test ./...     # unit tests
go vet ./...
go run .          # local server on :8080
```

## Deployment (Flux GitOps)

- Image: `ghcr.io/minutemailco/mcp-server:develop` (build.yml, multi-arch)
- Chart: `mcp-server` → `oci://ghcr.io/minutemailco/charts` (release-chart.yml)
- HelmRelease: `flux-repo/apps/mcp-server` (namespace `minutemail`)
- Ingress: `dev.mcp.minutemail.co` (Traefik, TLS `mcp-dev-tls` via cert-manager)
- Calls the gateway internally at `http://mm-api-gateway:80` — no ingress hop.

# MinuteMail MCP Server

[MinuteMail](https://minutemail.co) as MCP (Model Context Protocol) tools: let
AI agents and MCP clients operate ephemeral mailboxes, emails, attachments,
custom domains, teams, and mock OAuth identities for testing email and auth
flows.

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/docker/pulls/chrptvn/minutemail-mcp)](https://hub.docker.com/r/chrptvn/minutemail-mcp)
[![smithery badge](https://smithery.ai/badge/minutemailco/minutemail)](https://smithery.ai/servers/minutemailco/minutemail)

- **Endpoint:** `https://mcp.minutemail.co/mcp` (Streamable HTTP, protocol
  revision `2025-06-18`)
- **Auth:** your MinuteMail API key (`Bearer mmak_...`). The server is a
  stateless proxy — it holds **no credentials**, sessions, or local storage of
  its own; your key is forwarded verbatim to the MinuteMail API, where scopes
  and quotas are enforced. All state (mailboxes, mails, identities) lives in
  the MinuteMail platform behind it.
- Also listed in the [official MCP Registry](https://registry.modelcontextprotocol.io)
  as `io.github.minutemailco/mcp-server` and on
  [Smithery](https://smithery.ai/server/minutemailco/minutemail).

## Get an API key

1. Sign up at [minutemail.co](https://minutemail.co) (free during early access)
2. Create an API key in the dashboard
3. Configure your MCP client to send it as the `Authorization` header:

   `Authorization: Bearer mmak_XXXXXXXX`

## Connect your MCP client

**Claude Code / Claude Desktop / Cursor / any Streamable HTTP client:**

- URL: `https://mcp.minutemail.co/mcp`
- Header: `Authorization: Bearer mmak_...`

**Claude Code CLI:**

```bash
claude mcp add --transport http minutemail https://mcp.minutemail.co/mcp \
  --header "Authorization: Bearer mmak_XXXXXXXX"
```

**Raw JSON-RPC (what the clients do under the hood):**

```bash
curl -s https://mcp.minutemail.co/mcp \
  -H 'Authorization: Bearer mmak_XXXXXXXX' \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call",
       "params":{"name":"mm_create_mailbox","arguments":{"expiresIn":30}}}'
```

## Tools (39)

Every tool maps 1:1 to a route of the MinuteMail API. All calls require the
Bearer API key; write operations require the matching scope (`mailboxes`,
`domains`, `team`, `identities`).

### Mailboxes (`mailboxes` scope)
| Tool | Purpose |
|---|---|
| `mm_list_mailboxes` | List active mailboxes (optional exact-address lookup) |
| `mm_create_mailbox` | Create a mailbox (optional TTL, domain, recovery tag) |
| `mm_get_mailbox` / `mm_delete_mailbox` | Fetch / delete one mailbox |
| `mm_bulk_delete_mailboxes` | Delete several mailboxes by ID |
| `mm_list_mails` / `mm_inject_test_mail` | List mails / simulate an inbound email (multipart, with attachments) |
| `mm_get_mail` / `mm_delete_mail` / `mm_bulk_delete_mails` | Read / delete mails |
| `mm_list_attachments` / `mm_add_attachment` / `mm_get_attachment` / `mm_delete_attachment` / `mm_bulk_delete_attachments` | Attachment CRUD |

### Archived mailboxes (`mailboxes` scope)
`mm_list_archived_mailboxes`, `mm_get_archived_mailbox`,
`mm_delete_archived_mailbox`, `mm_reactivate_archived_mailbox`.

### Custom domains (`domains` scope)
`mm_list_domains`, `mm_register_domain`, `mm_verify_domain`, `mm_delete_domain`.

### Team (`team` scope)
`mm_list_members`, `mm_add_member`, `mm_get_member`, `mm_delete_member`,
`mm_create_invitation`, `mm_list_invitations`, `mm_delete_invitation`.

### Mock identities & OAuth clients (`identities` scope)
`mm_list_identities`, `mm_create_identity`, `mm_get_identity`,
`mm_delete_identity`, `mm_list_oauth_clients`, `mm_create_oauth_client`,
`mm_get_oauth_client`, `mm_delete_oauth_client`, `mm_rotate_client_secret`.

Tool results are returned as MCP text content carrying the API's JSON
response. Non-2xx responses become `isError: true` results with the HTTP
status and body (401 invalid key, 403 scope/domain, 429 quota with remaining
count, 502 upstream down).

## Typical agent workflow

```
mm_create_mailbox (expiresIn 30)
        ↓
your app under test sends a verification email to the mailbox address
        ↓
mm_list_mails → mm_get_mail (extract the code / link)
        ↓
assert the flow completed — the mailbox expires on its own
```

## Self-hosting

The server is a single static Go binary in a scratch image (~7 MB):

```bash
docker run -p 8080:8080 \
  -e API_BASE=https://api.minutemail.co \
  ghcr.io/minutemailco/mcp-server:latest
# or from Docker Hub:
docker run -p 8080:8080 \
  -e API_BASE=https://api.minutemail.co \
  chrptvn/minutemail-mcp:latest
```

Self-hosting still requires MinuteMail API keys — the server is a stateless
proxy over the hosted API, not a standalone implementation. It keeps no
sessions or local data; every mailbox, mail, and identity is stored by the
MinuteMail API it forwards to.

| Env var | Default | Purpose |
|---------|---------|-------------|
| `PORT` | `8080` | Listen port |
| `API_BASE` | `http://api-gateway:80` | MinuteMail API base URL (use `https://api.minutemail.co` outside the cluster) |
| `LOG_LEVEL` | `warn` | `debug`/`info`/`warn`/`error` |
| `LOG_FORMAT` | `json` | `json`/`text` |
| `PROFILE` | `dev` | Deployment profile label |

`GET /health` for liveness; `GET /metrics` for Prometheus counters.
API-key management is not exposed via MCP — manage keys from the web app.

## Development

Go 1.23, stdlib plus `prometheus/client_golang`.

```bash
go test ./...     # unit tests
go vet ./...
go run .          # local server on :8080
```

## License

[MIT](LICENSE) — © [MinuteMail.co](https://minutemail.co)

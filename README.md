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
       "params":{"name":"mailboxes.create","arguments":{"expiresIn":30}}}'
```

## Tools (39)

Every tool maps 1:1 to a route of the MinuteMail API. All calls require the
Bearer API key; write operations require the matching scope (`mailboxes`,
`domains`, `team`, `identities`).

### Mailboxes (`mailboxes` scope)
| Tool | Purpose |
|---|---|
| `mailboxes.list` | List active mailboxes (optional exact-address lookup) |
| `mailboxes.create` | Create a mailbox (optional TTL, domain, recovery tag) |
| `mailboxes.get` / `mailboxes.delete` | Fetch / delete one mailbox |
| `mailboxes.delete_bulk` | Delete several mailboxes by ID |
| `mails.list` / `mails.inject` | List mails / simulate an inbound email (multipart, with attachments) |
| `mails.get` / `mails.delete` / `mails.delete_bulk` | Read / delete mails |
| `attachments.list` / `attachments.add` / `attachments.get` / `attachments.delete` / `attachments.delete_bulk` | Attachment CRUD |

### Archived mailboxes (`mailboxes` scope)
`archived.list`, `archived.get`,
`archived.delete`, `archived.reactivate`.

### Custom domains (`domains` scope)
`domains.list`, `domains.register`, `domains.verify`, `domains.delete`.

### Team (`team` scope)
`team.members.list`, `team.members.add`, `team.members.get`, `team.members.delete`,
`team.invitations.create`, `team.invitations.list`, `team.invitations.delete`.

### Mock identities & OAuth clients (`identities` scope)
`identities.list`, `identities.create`, `identities.get`,
`identities.delete`, `oauth.clients.list`, `oauth.clients.create`,
`oauth.clients.get`, `oauth.clients.delete`, `oauth.clients.rotate_secret`.

Tool results are returned as MCP text content carrying the API's JSON
response. Non-2xx responses become `isError: true` results with the HTTP
status and body (401 invalid key, 403 scope/domain, 429 quota with remaining
count, 502 upstream down).

## Typical agent workflow

```
mailboxes.create (expiresIn 30)
        ↓
your app under test sends a verification email to the mailbox address
        ↓
mails.list → mails.get (extract the code / link)
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

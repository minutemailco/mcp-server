package tools

// Tool metadata (title, annotations, outputSchema) per the MCP spec. Applied
// to every registered tool in NewRegistry; TestAllToolsHaveMeta enforces that
// no tool is added without an entry here.
//
// Annotations follow the spec defaults where omitted; we set them explicitly
// so clients and registries get unambiguous hints:
//   - list/get tools: readOnly, idempotent
//   - delete tools: destructive
//   - create/verify/rotate tools: mutating, non-destructive
//
// Output schemas describe the JSON payload returned in the tool result's text
// content — the MinuteMail API response echoed by the gateway. Envelopes were
// verified against the live API: list endpoints return {"items":[...]} (oauth
// clients: {"clients":[...]}) and empty-body responses are reported as
// {"status":"ok","http_status":<n>}.

// toolMeta carries per-tool metadata applied at registry build time.
type toolMeta struct {
	title        string
	annotations  map[string]any
	outputSchema map[string]any
}

// ro marks a read-only, idempotent tool.
func ro(title string, out map[string]any) toolMeta {
	return toolMeta{
		title: title,
		annotations: map[string]any{
			"title":           title,
			"readOnlyHint":    true,
			"destructiveHint": false,
			"idempotentHint":  true,
			"openWorldHint":   false,
		},
		outputSchema: out,
	}
}

// create marks a mutating, non-destructive tool.
func create(title string, out map[string]any) toolMeta {
	return toolMeta{
		title: title,
		annotations: map[string]any{
			"title":           title,
			"readOnlyHint":    false,
			"destructiveHint": false,
			"idempotentHint":  false,
			"openWorldHint":   false,
		},
		outputSchema: out,
	}
}

// del marks a destructive tool.
func del(title string, out map[string]any) toolMeta {
	return toolMeta{
		title: title,
		annotations: map[string]any{
			"title":           title,
			"readOnlyHint":    false,
			"destructiveHint": true,
			"idempotentHint":  false,
			"openWorldHint":   false,
		},
		outputSchema: out,
	}
}

// ---------- output schema fragments ----------

func outObj(props map[string]any, required ...string) map[string]any {
	return schema(props, required...)
}

func outList(item map[string]any) map[string]any {
	return schema(map[string]any{
		"items": map[string]any{"type": "array", "items": item},
	}, "items")
}

// outStatus describes the synthesized empty-body success result.
func outStatus() map[string]any {
	return outObj(map[string]any{
		"status":      map[string]any{"type": "string", "const": "ok"},
		"http_status": integer("HTTP status of the API response"),
	}, "status", "http_status")
}

var (
	outMailbox = outObj(map[string]any{
		"id":           str("Mailbox ID"),
		"alias":        str("Local part of the address"),
		"domain":       str("Mailbox domain"),
		"address":      str("Full mailbox address"),
		"owner":        str("Owner (tenant) ID"),
		"recoverable":  boolean("Whether the mailbox can be recovered after expiry"),
		"messageCount": integer("Number of mails in the mailbox"),
		"expiresAt":    str("Expiry timestamp (RFC 3339); null when permanent"),
		"permanent":    boolean("Whether the mailbox never expires"),
		"createdAt":    str("Creation timestamp (RFC 3339)"),
	}, "id", "address")

	outMail = outObj(map[string]any{
		"id":         str("Mail ID"),
		"sender":     str("Sender email address"),
		"subject":    str("Email subject"),
		"receivedAt": str("Received timestamp (RFC 3339)"),
		"expiresAt":  str("Expiry timestamp (RFC 3339)"),
		"body":       str("Plain-text body (present on single-mail fetches)"),
	}, "id", "sender", "subject")

	outAttachment = outObj(map[string]any{
		"id":          str("Attachment ID"),
		"filename":    str("Attachment file name"),
		"contentType": str("MIME type"),
		"sizeBytes":   integer("Size in bytes"),
		"expiresAt":   str("Expiry timestamp (RFC 3339)"),
	}, "id", "filename")

	outAttachmentData = outObj(map[string]any{
		"id":          str("Attachment ID"),
		"filename":    str("Attachment file name"),
		"contentType": str("MIME type"),
		"sizeBytes":   integer("Size in bytes"),
		"expiresAt":   str("Expiry timestamp (RFC 3339)"),
		"data":        str("File contents, standard base64"),
	}, "id", "filename", "data")

	outDomain = outObj(map[string]any{
		"id":        str("Domain ID"),
		"name":      str("Domain name"),
		"status":    strEnum("DNS verification status", "pending", "verified"),
		"txtToken":  str("TXT token to add to the domain's DNS before verification"),
		"mxTarget":  str("MX target to add to the domain's DNS before verification"),
		"createdAt": str("Creation timestamp (RFC 3339)"),
	}, "id", "name", "status")

	outMember = outObj(map[string]any{
		"id":        str("Member ID"),
		"user_id":   str("Member user ID"),
		"username":  str("Member username"),
		"email":     str("Member email address"),
		"status":    str("Member status (e.g. ACTIVE)"),
		"createdAt": str("Creation timestamp (RFC 3339)"),
	}, "id", "email", "status")

	outInvitation = outObj(map[string]any{
		"id":        str("Invitation ID"),
		"email":     str("Invitee email address"),
		"status":    str("Invitation status (e.g. PENDING)"),
		"createdAt": str("Creation timestamp (RFC 3339)"),
	}, "id", "email", "status")

	outIdentity = outObj(map[string]any{
		"id":             str("Identity ID"),
		"clientId":       str("OAuth client ID the identity belongs to"),
		"mailboxAddress": str("Mailbox address the identity is linked to"),
		"username":       str("Identity username"),
		"name":           str("Display name"),
		"avatarUrl":      str("Avatar URL"),
	}, "id", "clientId", "mailboxAddress")

	outOAuthClient = outObj(map[string]any{
		"clientId":      str("Public OAuth client ID"),
		"name":          str("Client display name"),
		"providerType":  strEnum("Identity provider type", "google", "github", "apple", "facebook", "custom"),
		"providerLabel": str("Custom provider label"),
		"redirectUris":  arr("Allowed redirect URIs"),
		"clientSecret":  str("Plaintext secret — returned once at creation or rotation"),
		"createdAt":     str("Creation timestamp (RFC 3339)"),
	}, "clientId", "name")

	// outClientsList matches the {"clients":[...]} envelope.
	outClientsList = outObj(map[string]any{
		"clients": map[string]any{"type": "array", "items": outOAuthClient},
	})
)

// toolMetaTable maps tool name to its metadata. Must cover every tool.
var toolMetaTable = map[string]toolMeta{
	// mailboxes
	"mailboxes.list":          ro("List mailboxes", outList(outMailbox)),
	"mailboxes.create":        create("Create mailbox", outMailbox),
	"mailboxes.get":           ro("Get mailbox", outMailbox),
	"mailboxes.delete":        del("Delete mailbox", outStatus()),
	"mailboxes.delete_bulk":   del("Bulk delete mailboxes", outStatus()),
	"mails.list":              ro("List mails", outList(outMail)),
	"mails.inject":            create("Inject test mail", outMail),
	"mails.get":               ro("Get mail", outMail),
	"mails.delete":            del("Delete mail", outStatus()),
	"mails.delete_bulk":       del("Bulk delete mails", outStatus()),
	"attachments.list":        ro("List attachments", outList(outAttachment)),
	"attachments.add":         create("Add attachment", outAttachment),
	"attachments.get":         ro("Get attachment", outAttachmentData),
	"attachments.delete":      del("Delete attachment", outStatus()),
	"attachments.delete_bulk": del("Bulk delete attachments", outStatus()),
	// archived mailboxes
	"archived.list":       ro("List archived mailboxes", outList(outMailbox)),
	"archived.get":        ro("Get archived mailbox", outMailbox),
	"archived.delete":     del("Delete archived mailbox", outStatus()),
	"archived.reactivate": create("Reactivate archived mailbox", outMailbox),
	// domains
	"domains.list":     ro("List domains", outList(outDomain)),
	"domains.register": create("Register domain", outDomain),
	"domains.verify":   create("Verify domain", outDomain),
	"domains.delete":   del("Delete domain", outStatus()),
	// team
	"team.members.list":       ro("List team members", outList(outMember)),
	"team.members.add":        create("Add team member", outMember),
	"team.members.get":        ro("Get team member", outMember),
	"team.members.delete":     del("Delete team member", outStatus()),
	"team.invitations.create": create("Create invitation", outInvitation),
	"team.invitations.list":   ro("List invitations", outList(outInvitation)),
	"team.invitations.delete": del("Delete invitation", outStatus()),
	// identities & oauth clients
	"identities.list":             ro("List identities", outList(outIdentity)),
	"identities.create":           create("Create identity", outIdentity),
	"identities.get":              ro("Get identity", outIdentity),
	"identities.delete":           del("Delete identity", outStatus()),
	"oauth.clients.list":          ro("List OAuth clients", outClientsList),
	"oauth.clients.create":        create("Create OAuth client", outOAuthClient),
	"oauth.clients.get":           ro("Get OAuth client", outOAuthClient),
	"oauth.clients.delete":        del("Delete OAuth client", outStatus()),
	"oauth.clients.rotate_secret": create("Rotate client secret", outOAuthClient),
}

// applyMeta fills Title, Annotations and OutputSchema on the registry's tools.
// TestAllToolsHaveMeta catches tools missing from the table.
func applyMeta(tools []Tool) {
	for i := range tools {
		if m, ok := toolMetaTable[tools[i].Name]; ok {
			tools[i].Title = m.title
			tools[i].Annotations = m.annotations
			tools[i].OutputSchema = m.outputSchema
		}
	}
}

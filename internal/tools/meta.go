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
	"mm_list_mailboxes":          ro("List mailboxes", outList(outMailbox)),
	"mm_create_mailbox":          create("Create mailbox", outMailbox),
	"mm_get_mailbox":             ro("Get mailbox", outMailbox),
	"mm_delete_mailbox":          del("Delete mailbox", outStatus()),
	"mm_bulk_delete_mailboxes":   del("Bulk delete mailboxes", outStatus()),
	"mm_list_mails":              ro("List mails", outList(outMail)),
	"mm_inject_test_mail":        create("Inject test mail", outMail),
	"mm_get_mail":                ro("Get mail", outMail),
	"mm_delete_mail":             del("Delete mail", outStatus()),
	"mm_bulk_delete_mails":       del("Bulk delete mails", outStatus()),
	"mm_list_attachments":        ro("List attachments", outList(outAttachment)),
	"mm_add_attachment":          create("Add attachment", outAttachment),
	"mm_get_attachment":          ro("Get attachment", outAttachmentData),
	"mm_delete_attachment":       del("Delete attachment", outStatus()),
	"mm_bulk_delete_attachments": del("Bulk delete attachments", outStatus()),
	// archived mailboxes
	"mm_list_archived_mailboxes":     ro("List archived mailboxes", outList(outMailbox)),
	"mm_get_archived_mailbox":        ro("Get archived mailbox", outMailbox),
	"mm_delete_archived_mailbox":     del("Delete archived mailbox", outStatus()),
	"mm_reactivate_archived_mailbox": create("Reactivate archived mailbox", outMailbox),
	// domains
	"mm_list_domains":    ro("List domains", outList(outDomain)),
	"mm_register_domain": create("Register domain", outDomain),
	"mm_verify_domain":   create("Verify domain", outDomain),
	"mm_delete_domain":   del("Delete domain", outStatus()),
	// team
	"mm_list_members":      ro("List team members", outList(outMember)),
	"mm_add_member":        create("Add team member", outMember),
	"mm_get_member":        ro("Get team member", outMember),
	"mm_delete_member":     del("Delete team member", outStatus()),
	"mm_create_invitation": create("Create invitation", outInvitation),
	"mm_list_invitations":  ro("List invitations", outList(outInvitation)),
	"mm_delete_invitation": del("Delete invitation", outStatus()),
	// identities & oauth clients
	"mm_list_identities":      ro("List identities", outList(outIdentity)),
	"mm_create_identity":      create("Create identity", outIdentity),
	"mm_get_identity":         ro("Get identity", outIdentity),
	"mm_delete_identity":      del("Delete identity", outStatus()),
	"mm_list_oauth_clients":   ro("List OAuth clients", outClientsList),
	"mm_create_oauth_client":  create("Create OAuth client", outOAuthClient),
	"mm_get_oauth_client":     ro("Get OAuth client", outOAuthClient),
	"mm_delete_oauth_client":  del("Delete OAuth client", outStatus()),
	"mm_rotate_client_secret": create("Rotate client secret", outOAuthClient),
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

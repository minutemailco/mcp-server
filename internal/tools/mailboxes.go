package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"

	"mcp-server/internal/gateway"
)

// mailboxTools covers /v1/mailboxes (proxied to mm-mailbox-service).
func mailboxTools() []Tool {
	return []Tool{
		{
			Name:        "mm_list_mailboxes",
			Description: "List the tenant's active mailboxes (owner is derived from the API key). Optionally look up a single mailbox by exact address.",
			InputSchema: schema(map[string]any{
				"address": str("Exact mailbox address to look up (e.g. user@minutemail.cc). Optional."),
			}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				p := "/v1/mailboxes"
				if address := argStringOpt(args, "address"); address != "" {
					p += "?address=" + queryParams(address)
				}
				return callRaw(ctx, gw, "GET", p, nil, bearer)
			},
		},
		{
			Name:        "mm_create_mailbox",
			Description: "Create a new temporary mailbox. The domain defaults to the tenant's default domain; the owner is always the API key's owner.",
			InputSchema: schema(map[string]any{
				"domain":       str("Mailbox domain. Defaults to the tenant default domain (e.g. minutemail.cc)."),
				"expiresIn":    integer("Lifetime in minutes, 1-60. Omit for the service default TTL."),
				"noExpiration": boolean("Set true for a permanent mailbox (mutually exclusive with expiresIn)."),
				"recoverable":  boolean("Set true to keep the mailbox recoverable after expiry (requires tag)."),
				"tag":          str("Recovery tag, required when recoverable is true."),
			}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				body := map[string]any{}
				if v := argStringOpt(args, "domain"); v != "" {
					body["domain"] = v
				}
				if n, ok, err := argIntOpt(args, "expiresIn"); err != nil {
					return nil, err
				} else if ok {
					body["expiresIn"] = n
				}
				if v, ok := argBoolOpt(args, "noExpiration"); ok {
					body["noExpiration"] = v
				}
				if v, ok := argBoolOpt(args, "recoverable"); ok {
					body["recoverable"] = v
				}
				if v := argStringOpt(args, "tag"); v != "" {
					body["tag"] = v
				}
				return callJSON(ctx, gw, "POST", "/v1/mailboxes", body, bearer)
			},
		},
		{
			Name:        "mm_get_mailbox",
			Description: "Fetch a single mailbox by ID.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
			}, "mailboxId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "GET", path("v1", "mailboxes", id), nil, bearer)
			},
		},
		{
			Name:        "mm_delete_mailbox",
			Description: "Delete a mailbox and its contents by ID.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
			}, "mailboxId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "DELETE", path("v1", "mailboxes", id), nil, bearer)
			},
		},
		{
			Name:        "mm_bulk_delete_mailboxes",
			Description: "Delete several mailboxes at once by ID.",
			InputSchema: schema(map[string]any{
				"ids": arr("Mailbox IDs to delete"),
			}, "ids"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				ids, err := argStringsOpt(args, "ids")
				if err != nil {
					return nil, err
				}
				if len(ids) == 0 {
					return nil, fmt.Errorf("argument %q must contain at least one id", "ids")
				}
				return callJSON(ctx, gw, "DELETE", "/v1/mailboxes", map[string]any{"ids": ids}, bearer)
			},
		},
		{
			Name:        "mm_list_mails",
			Description: "List the emails in a mailbox, newest first.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
			}, "mailboxId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "GET", path("v1", "mailboxes", id, "mails"), nil, bearer)
			},
		},
		{
			Name:        "mm_inject_test_mail",
			Description: "Inject a simulated inbound email into a mailbox (multipart upload). No external mail is sent; use this to simulate inbound mail for flow testing.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
				"sender":    str("Sender email address"),
				"subject":   str("Email subject"),
				"body":      str("Plain-text body"),
				"expiresIn": integer("Mail lifetime in minutes (>=1). Optional."),
				"attachments": map[string]any{
					"type":        "array",
					"description": "Attachments to include",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"filename":    str("Attachment file name"),
							"contentType": str("MIME type, defaults to application/octet-stream"),
							"data":        str("File contents, standard base64"),
						},
						"required": []string{"filename", "data"},
					},
				},
			}, "mailboxId", "sender", "subject", "body"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				fields := map[string]string{}
				for _, name := range []string{"sender", "subject", "body"} {
					v, err := argString(args, name)
					if err != nil {
						return nil, err
					}
					fields[name] = v
				}
				if n, ok, err := argIntOpt(args, "expiresIn"); err != nil {
					return nil, err
				} else if ok {
					fields["expiresIn"] = strconv.FormatInt(n, 10)
				}
				objs, err := argObjectsOpt(args, "attachments")
				if err != nil {
					return nil, err
				}
				files := make([]gateway.FilePart, 0, len(objs))
				for _, obj := range objs {
					name, err := argString(obj, "filename")
					if err != nil {
						return nil, err
					}
					dataB64, err := argString(obj, "data")
					if err != nil {
						return nil, err
					}
					data, err := base64.StdEncoding.DecodeString(dataB64)
					if err != nil {
						return nil, fmt.Errorf("attachment %q: invalid base64 data", name)
					}
					files = append(files, gateway.FilePart{
						Field:       "files",
						Filename:    name,
						ContentType: argStringOpt(obj, "contentType"),
						Data:        data,
					})
				}
				resp, err := gw.DoMultipart(ctx, "POST", path("v1", "mailboxes", mailboxID, "mails"), fields, files, bearer)
				if err != nil {
					return nil, err
				}
				return resultFromResponse(resp)
			},
		},
		{
			Name:        "mm_get_mail",
			Description: "Fetch a single email by ID, including body and attachment metadata.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
				"mailId":    str("Mail ID"),
			}, "mailboxId", "mailId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, mailID, err := twoStrings(args, "mailboxId", "mailId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "GET", path("v1", "mailboxes", mailboxID, "mails", mailID), nil, bearer)
			},
		},
		{
			Name:        "mm_delete_mail",
			Description: "Delete a single email.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
				"mailId":    str("Mail ID"),
			}, "mailboxId", "mailId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, mailID, err := twoStrings(args, "mailboxId", "mailId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "DELETE", path("v1", "mailboxes", mailboxID, "mails", mailID), nil, bearer)
			},
		},
		{
			Name:        "mm_bulk_delete_mails",
			Description: "Delete several emails of one mailbox at once.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
				"ids":       arr("Mail IDs to delete"),
			}, "mailboxId", "ids"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				ids, err := argStringsOpt(args, "ids")
				if err != nil {
					return nil, err
				}
				if len(ids) == 0 {
					return nil, fmt.Errorf("argument %q must contain at least one id", "ids")
				}
				return callJSON(ctx, gw, "DELETE", path("v1", "mailboxes", mailboxID, "mails"), map[string]any{"ids": ids}, bearer)
			},
		},
		{
			Name:        "mm_list_attachments",
			Description: "List the attachments of an email (metadata only).",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
				"mailId":    str("Mail ID"),
			}, "mailboxId", "mailId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, mailID, err := twoStrings(args, "mailboxId", "mailId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "GET", path("v1", "mailboxes", mailboxID, "mails", mailID, "attachments"), nil, bearer)
			},
		},
		{
			Name:        "mm_add_attachment",
			Description: "Attach a file to a test-injected email (base64 payload).",
			InputSchema: schema(map[string]any{
				"mailboxId":   str("Mailbox ID"),
				"mailId":      str("Mail ID"),
				"filename":    str("Attachment file name"),
				"contentType": str("MIME type, defaults to application/octet-stream"),
				"sizeBytes":   integer("Expected size in bytes. Optional; validated against the decoded data when set."),
				"expiresIn":   integer("Attachment lifetime in minutes (>=1). Optional."),
				"data":        str("File contents, standard base64"),
			}, "mailboxId", "mailId", "filename", "data"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, mailID, err := twoStrings(args, "mailboxId", "mailId")
				if err != nil {
					return nil, err
				}
				body := map[string]any{}
				for _, name := range []string{"filename", "data"} {
					v, err := argString(args, name)
					if err != nil {
						return nil, err
					}
					body[name] = v
				}
				if v := argStringOpt(args, "contentType"); v != "" {
					body["contentType"] = v
				}
				if n, ok, err := argIntOpt(args, "sizeBytes"); err != nil {
					return nil, err
				} else if ok {
					body["sizeBytes"] = n
				}
				if n, ok, err := argIntOpt(args, "expiresIn"); err != nil {
					return nil, err
				} else if ok {
					body["expiresIn"] = n
				}
				return callJSON(ctx, gw, "POST", path("v1", "mailboxes", mailboxID, "mails", mailID, "attachments"), body, bearer)
			},
		},
		{
			Name:        "mm_get_attachment",
			Description: "Download an attachment. The file contents are returned base64-encoded in the JSON \"data\" field.",
			InputSchema: schema(map[string]any{
				"mailboxId":    str("Mailbox ID"),
				"mailId":       str("Mail ID"),
				"attachmentId": str("Attachment ID"),
			}, "mailboxId", "mailId", "attachmentId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				mailID, err := argString(args, "mailId")
				if err != nil {
					return nil, err
				}
				attID, err := argString(args, "attachmentId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "GET", path("v1", "mailboxes", mailboxID, "mails", mailID, "attachments", attID), nil, bearer)
			},
		},
		{
			Name:        "mm_delete_attachment",
			Description: "Delete a single attachment.",
			InputSchema: schema(map[string]any{
				"mailboxId":    str("Mailbox ID"),
				"mailId":       str("Mail ID"),
				"attachmentId": str("Attachment ID"),
			}, "mailboxId", "mailId", "attachmentId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, mailID, err := twoStrings(args, "mailboxId", "mailId")
				if err != nil {
					return nil, err
				}
				attID, err := argString(args, "attachmentId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "DELETE", path("v1", "mailboxes", mailboxID, "mails", mailID, "attachments", attID), nil, bearer)
			},
		},
		{
			Name:        "mm_bulk_delete_attachments",
			Description: "Delete several attachments of one email at once.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Mailbox ID"),
				"mailId":    str("Mail ID"),
				"ids":       arr("Attachment IDs to delete"),
			}, "mailboxId", "mailId", "ids"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				mailboxID, mailID, err := twoStrings(args, "mailboxId", "mailId")
				if err != nil {
					return nil, err
				}
				ids, err := argStringsOpt(args, "ids")
				if err != nil {
					return nil, err
				}
				if len(ids) == 0 {
					return nil, fmt.Errorf("argument %q must contain at least one id", "ids")
				}
				return callJSON(ctx, gw, "DELETE", path("v1", "mailboxes", mailboxID, "mails", mailID, "attachments"), map[string]any{"ids": ids}, bearer)
			},
		},
	}
}

func twoStrings(args map[string]any, first, second string) (string, string, error) {
	a, err := argString(args, first)
	if err != nil {
		return "", "", err
	}
	b, err := argString(args, second)
	if err != nil {
		return "", "", err
	}
	return a, b, nil
}

func queryParams(v string) string {
	return url.QueryEscape(v)
}

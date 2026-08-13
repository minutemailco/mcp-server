package tools

import (
	"context"
	"fmt"

	"mm-mcp-server/internal/gateway"
)

// archivedTools covers /v1/archived-mailboxes (proxied to mm-mailbox-service).
func archivedTools() []Tool {
	return []Tool{
		{
			Name:        "mm_list_archived_mailboxes",
			Description: "List the tenant's archived (expired but recoverable) mailboxes.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/archived-mailboxes", nil, bearer)
			},
		},
		{
			Name:        "mm_get_archived_mailbox",
			Description: "Fetch a single archived mailbox by ID.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Archived mailbox ID"),
			}, "mailboxId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "GET", path("v1", "archived-mailboxes", id), nil, bearer)
			},
		},
		{
			Name:        "mm_delete_archived_mailbox",
			Description: "Permanently delete an archived mailbox.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Archived mailbox ID"),
			}, "mailboxId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "DELETE", path("v1", "archived-mailboxes", id), nil, bearer)
			},
		},
		{
			Name:        "mm_reactivate_archived_mailbox",
			Description: "Reactivate an archived mailbox back to active state.",
			InputSchema: schema(map[string]any{
				"mailboxId": str("Archived mailbox ID"),
				"expiresIn": integer("New lifetime in minutes, 1-60. Optional."),
			}, "mailboxId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "mailboxId")
				if err != nil {
					return nil, err
				}
				var body map[string]any
				if n, ok, err := argIntOpt(args, "expiresIn"); err != nil {
					return nil, err
				} else if ok {
					body = map[string]any{"expiresIn": n}
				}
				return callJSON(ctx, gw, "POST", path("v1", "archived-mailboxes", id, "reactivate"), body, bearer)
			},
		},
	}
}

// domainTools covers /v1/domains (proxied to mm-domains-service).
func domainTools() []Tool {
	return []Tool{
		{
			Name:        "mm_list_domains",
			Description: "List the tenant's custom domains with their DNS verification status.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/domains", nil, bearer)
			},
		},
		{
			Name:        "mm_register_domain",
			Description: "Register a new custom domain. DNS records (TXT token + MX) must then be added before verification can succeed.",
			InputSchema: schema(map[string]any{
				"name": str("Domain name, e.g. mail.example.com"),
			}, "name"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				name, err := argString(args, "name")
				if err != nil {
					return nil, err
				}
				return callJSON(ctx, gw, "POST", "/v1/domains", map[string]any{"name": name}, bearer)
			},
		},
		{
			Name:        "mm_verify_domain",
			Description: "Trigger DNS verification (TXT + MX) of a registered domain.",
			InputSchema: schema(map[string]any{
				"domainId": str("Domain ID"),
			}, "domainId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "domainId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "POST", path("v1", "domains", id, "verify"), nil, bearer)
			},
		},
		{
			Name:        "mm_delete_domain",
			Description: "Delete a registered custom domain.",
			InputSchema: schema(map[string]any{
				"domainId": str("Domain ID"),
			}, "domainId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "domainId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "DELETE", path("v1", "domains", id), nil, bearer)
			},
		},
	}
}

// teamTools covers /v1/members and /v1/invitations (proxied to mm-team-service).
func teamTools() []Tool {
	member := func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client, method string) (*Result, error) {
		id, err := argString(args, "memberId")
		if err != nil {
			return nil, err
		}
		return callRaw(ctx, gw, method, path("v1", "members", id), nil, bearer)
	}
	return []Tool{
		{
			Name:        "mm_list_members",
			Description: "List the tenant's team members.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/members", nil, bearer)
			},
		},
		{
			Name:        "mm_add_member",
			Description: "Add a team member directly (no invitation flow).",
			InputSchema: schema(map[string]any{
				"user_id":  str("Member user ID"),
				"username": str("Member username"),
				"email":    str("Member email address"),
				"status":   str("Member status, conventionally ACTIVE"),
			}, "user_id", "username", "email", "status"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				body := map[string]any{}
				for _, name := range []string{"user_id", "username", "email", "status"} {
					v, err := argString(args, name)
					if err != nil {
						return nil, err
					}
					body[name] = v
				}
				return callJSON(ctx, gw, "POST", "/v1/members", body, bearer)
			},
		},
		{
			Name:        "mm_get_member",
			Description: "Fetch a single team member by ID.",
			InputSchema: schema(map[string]any{
				"memberId": str("Member ID"),
			}, "memberId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return member(ctx, args, bearer, gw, "GET")
			},
		},
		{
			Name:        "mm_delete_member",
			Description: "Remove a team member.",
			InputSchema: schema(map[string]any{
				"memberId": str("Member ID"),
			}, "memberId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return member(ctx, args, bearer, gw, "DELETE")
			},
		},
		{
			Name:        "mm_create_invitation",
			Description: "Create a team invitation for an email address (SMTP invite is sent by the team service).",
			InputSchema: schema(map[string]any{
				"email": str("Invitee email address"),
			}, "email"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				email, err := argString(args, "email")
				if err != nil {
					return nil, err
				}
				return callJSON(ctx, gw, "POST", "/v1/invitations", map[string]any{"email": email}, bearer)
			},
		},
		{
			Name:        "mm_list_invitations",
			Description: "List the tenant's team invitations.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/invitations", nil, bearer)
			},
		},
		{
			Name:        "mm_delete_invitation",
			Description: "Revoke a team invitation.",
			InputSchema: schema(map[string]any{
				"invitationId": str("Invitation ID"),
			}, "invitationId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "invitationId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "DELETE", path("v1", "invitations", id), nil, bearer)
			},
		},
	}
}

// identityTools covers /v1/identities (proxied to mm-idp-service /web/identities).
func identityTools() []Tool {
	return []Tool{
		{
			Name:        "mm_list_identities",
			Description: "List the tenant's mock identities, optionally filtered by OAuth client or mailbox address.",
			InputSchema: schema(map[string]any{
				"clientId":       str("Filter by OAuth client ID. Optional."),
				"mailboxAddress": str("Filter by mailbox address. Optional."),
			}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				p := "/v1/identities"
				q := ""
				if v := argStringOpt(args, "clientId"); v != "" {
					q = "clientId=" + queryParams(v)
				}
				if v := argStringOpt(args, "mailboxAddress"); v != "" {
					if q != "" {
						q += "&"
					}
					q += "mailboxAddress=" + queryParams(v)
				}
				if q != "" {
					p += "?" + q
				}
				return callRaw(ctx, gw, "GET", p, nil, bearer)
			},
		},
		{
			Name:        "mm_create_identity",
			Description: "Create a mock identity bound to a mailbox for OAuth flow testing.",
			InputSchema: schema(map[string]any{
				"clientId":       str("OAuth client ID the identity belongs to"),
				"mailboxAddress": str("Mailbox address the identity is linked to (must exist)"),
				"username":       str("Identity username. Optional."),
				"name":           str("Display name. Optional."),
				"avatarUrl":      str("Avatar URL. Optional."),
			}, "clientId", "mailboxAddress"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				body := map[string]any{}
				for _, name := range []string{"clientId", "mailboxAddress"} {
					v, err := argString(args, name)
					if err != nil {
						return nil, err
					}
					body[name] = v
				}
				for _, name := range []string{"username", "name", "avatarUrl"} {
					if v := argStringOpt(args, name); v != "" {
						body[name] = v
					}
				}
				return callJSON(ctx, gw, "POST", "/v1/identities", body, bearer)
			},
		},
		{
			Name:        "mm_get_identity",
			Description: "Fetch a single mock identity by ID.",
			InputSchema: schema(map[string]any{
				"identityId": str("Identity ID"),
			}, "identityId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "identityId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "GET", path("v1", "identities", id), nil, bearer)
			},
		},
		{
			Name:        "mm_delete_identity",
			Description: "Delete a mock identity.",
			InputSchema: schema(map[string]any{
				"identityId": str("Identity ID"),
			}, "identityId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "identityId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "DELETE", path("v1", "identities", id), nil, bearer)
			},
		},
		{
			Name:        "mm_list_oauth_clients",
			Description: "List the tenant's OAuth clients for mock identity testing.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/identities/clients", nil, bearer)
			},
		},
		{
			Name:        "mm_create_oauth_client",
			Description: "Register an OAuth client for mock identity flows. The plaintext clientSecret is returned once at creation.",
			InputSchema: schema(map[string]any{
				"name":          str("Client display name"),
				"providerType":  strEnum("Identity provider type", "google", "github", "apple", "facebook", "custom"),
				"providerLabel": str("Custom provider label. Required when providerType is custom."),
				"redirectUris":  arr("Allowed redirect URIs (at least one)"),
			}, "name", "redirectUris"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				name, err := argString(args, "name")
				if err != nil {
					return nil, err
				}
				uris, err := argStringsOpt(args, "redirectUris")
				if err != nil {
					return nil, err
				}
				if len(uris) == 0 {
					return nil, fmt.Errorf("argument %q must contain at least one redirect URI", "redirectUris")
				}
				body := map[string]any{"name": name, "redirectUris": uris}
				if v := argStringOpt(args, "providerType"); v != "" {
					body["providerType"] = v
				}
				if v := argStringOpt(args, "providerLabel"); v != "" {
					body["providerLabel"] = v
				}
				return callJSON(ctx, gw, "POST", "/v1/identities/clients", body, bearer)
			},
		},
		{
			Name:        "mm_get_oauth_client",
			Description: "Fetch a single OAuth client by its public client ID.",
			InputSchema: schema(map[string]any{
				"clientId": str("Public OAuth client ID"),
			}, "clientId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "clientId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "GET", path("v1", "identities", "clients", id), nil, bearer)
			},
		},
		{
			Name:        "mm_delete_oauth_client",
			Description: "Delete an OAuth client by its public client ID.",
			InputSchema: schema(map[string]any{
				"clientId": str("Public OAuth client ID"),
			}, "clientId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "clientId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "DELETE", path("v1", "identities", "clients", id), nil, bearer)
			},
		},
		{
			Name:        "mm_rotate_client_secret",
			Description: "Rotate an OAuth client's secret. The new plaintext secret is returned once.",
			InputSchema: schema(map[string]any{
				"clientId": str("Public OAuth client ID"),
			}, "clientId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "clientId")
				if err != nil {
					return nil, err
				}
				return callRaw(ctx, gw, "POST", path("v1", "identities", "clients", id, "rotate-secret"), nil, bearer)
			},
		},
	}
}

// buildTools returns the full ordered tool list.
func buildTools() []Tool {
	all := make([]Tool, 0, 40)
	all = append(all, mailboxTools()...)
	all = append(all, archivedTools()...)
	all = append(all, domainTools()...)
	all = append(all, teamTools()...)
	all = append(all, identityTools()...)
	return all
}

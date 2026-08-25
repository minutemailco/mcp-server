package tools

import (
	"context"
	"fmt"

	"mcp-server/internal/gateway"
)

// archivedTools covers /v1/archived-mailboxes (proxied to mm-mailbox-service).
func archivedTools() []Tool {
	return []Tool{
		{
			Name:        "archived.list",
			Description: "List the tenant's archived (expired but recoverable) mailboxes.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/archived-mailboxes", nil, bearer)
			},
		},
		{
			Name:        "archived.get",
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
			Name:        "archived.delete",
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
			Name:        "archived.reactivate",
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
			Name:        "domains.list",
			Description: "List the tenant's custom domains with their DNS verification status.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/domains", nil, bearer)
			},
		},
		{
			Name:        "domains.register",
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
			Name:        "domains.verify",
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
			Name:        "domains.delete",
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
			Name:        "team.members.list",
			Description: "List the tenant's team members.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/members", nil, bearer)
			},
		},
		{
			Name:        "team.members.add",
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
			Name:        "team.members.get",
			Description: "Fetch a single team member by ID.",
			InputSchema: schema(map[string]any{
				"memberId": str("Member ID"),
			}, "memberId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return member(ctx, args, bearer, gw, "GET")
			},
		},
		{
			Name:        "team.members.delete",
			Description: "Remove a team member.",
			InputSchema: schema(map[string]any{
				"memberId": str("Member ID"),
			}, "memberId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return member(ctx, args, bearer, gw, "DELETE")
			},
		},
		{
			Name:        "team.invitations.create",
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
			Name:        "team.invitations.list",
			Description: "List the tenant's team invitations.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/invitations", nil, bearer)
			},
		},
		{
			Name:        "team.invitations.delete",
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
			Name:        "identities.list",
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
			Name:        "identities.create",
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
			Name:        "identities.get",
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
			Name:        "identities.update",
			Description: "Update a mock identity: profile fields, isActive (activate/deactivate), emailVerified, and custom claims merged into the ID token and userinfo.",
			InputSchema: schema(map[string]any{
				"identityId":    str("Identity ID"),
				"username":      str("Identity username. Optional."),
				"name":          str("Display name. Optional."),
				"avatarUrl":     str("Avatar URL. Optional."),
				"isActive":      boolean("Whether the identity can be used in OAuth flows. Optional."),
				"emailVerified": boolean("Value of the email_verified claim issued for this identity. Optional."),
				"claims": map[string]any{
					"type":                 "object",
					"description":          "Custom claims (e.g. role, plan) merged into the ID token and userinfo. Replaces existing claims. Reserved claim names are rejected. Optional.",
					"additionalProperties": map[string]any{"type": "string"},
				},
			}, "identityId"),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				id, err := argString(args, "identityId")
				if err != nil {
					return nil, err
				}
				body := map[string]any{}
				for _, name := range []string{"username", "name", "avatarUrl"} {
					if v := argStringOpt(args, name); v != "" {
						body[name] = v
					}
				}
				for _, name := range []string{"isActive", "emailVerified"} {
					if v, ok := argBoolOpt(args, name); ok {
						body[name] = v
					}
				}
				if v, ok := args["claims"]; ok && v != nil {
					raw, ok := v.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("argument %q must be an object of string values", "claims")
					}
					claims := make(map[string]string, len(raw))
					for k, item := range raw {
						s, ok := item.(string)
						if !ok {
							return nil, fmt.Errorf("argument %q must contain only string values (key %q is not a string)", "claims", k)
						}
						claims[k] = s
					}
					body["claims"] = claims
				}
				if len(body) == 0 {
					return nil, fmt.Errorf("provide at least one field to update")
				}
				return callJSON(ctx, gw, "PATCH", path("v1", "identities", id), body, bearer)
			},
		},
		{
			Name:        "identities.delete",
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
			Name:        "oauth.clients.list",
			Description: "List the tenant's OAuth clients for mock identity testing.",
			InputSchema: schema(map[string]any{}),
			Handler: func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error) {
				return callRaw(ctx, gw, "GET", "/v1/identities/clients", nil, bearer)
			},
		},
		{
			Name:        "oauth.clients.create",
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
			Name:        "oauth.clients.get",
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
			Name:        "oauth.clients.delete",
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
			Name:        "oauth.clients.rotate_secret",
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

package resellersite

import (
	"context"
	"net/url"
	"strings"
)

// Site is a pair of entry points for the same application and customer ledger.
// Site origins are server-side configuration; the primary origin is not sent to
// the customer frontend as a domain mapping.
type Site struct {
	Enabled        bool
	PrimaryOrigin  string
	CustomerOrigin string
}

type requestSite struct {
	site     Site
	customer bool
}

type contextKey struct{}

func WithHost(ctx context.Context, host string) context.Context {
	for _, site := range configuredSites {
		primary, _ := url.Parse(site.PrimaryOrigin)
		if strings.EqualFold(host, primary.Host) {
			return context.WithValue(ctx, contextKey{}, requestSite{site: site})
		}
		if site.Enabled {
			customer, _ := url.Parse(site.CustomerOrigin)
			if strings.EqualFold(host, customer.Host) {
				return context.WithValue(ctx, contextKey{}, requestSite{site: site, customer: true})
			}
		}
	}
	return ctx
}

func IsCustomer(ctx context.Context) bool {
	entry, _ := ctx.Value(contextKey{}).(requestSite)
	return entry.customer
}

func CustomerOrigin(ctx context.Context) string {
	entry, _ := ctx.Value(contextKey{}).(requestSite)
	if entry.customer {
		return entry.site.CustomerOrigin
	}
	return ""
}

// Rewrite keeps configured first-party links on the customer's entry point.
// Relative links and links to other services retain their existing meaning.
func Rewrite(ctx context.Context, value string) string {
	entry, _ := ctx.Value(contextKey{}).(requestSite)
	if !entry.customer {
		return value
	}
	return strings.ReplaceAll(value, entry.site.PrimaryOrigin, entry.site.CustomerOrigin)
}

func InvitationURL(ctx context.Context, code string) string {
	entry, ok := ctx.Value(contextKey{}).(requestSite)
	if !ok {
		return ""
	}
	origin := entry.site.PrimaryOrigin
	if entry.site.Enabled {
		origin = entry.site.CustomerOrigin
	}
	return origin + "/register?" + url.Values{"reseller": {code}}.Encode()
}

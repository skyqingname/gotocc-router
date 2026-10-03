package service

import (
	"context"
	"net/url"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/resellersite"
)

func applyResellerSitePublicSettings(ctx context.Context, settings *PublicSettings) (*PublicSettings, error) {
	if !resellersite.IsCustomer(ctx) {
		return settings, nil
	}
	apiURL, err := customerSiteAPIURL(ctx, settings.APIBaseURL)
	if err != nil {
		return nil, err
	}
	settings.SiteLogo = resellersite.Rewrite(ctx, settings.SiteLogo)
	settings.APIBaseURL = apiURL
	settings.DocURL = resellersite.Rewrite(ctx, settings.DocURL)
	settings.ContactInfo = resellersite.Rewrite(ctx, settings.ContactInfo)
	settings.HomeContent = resellersite.Rewrite(ctx, settings.HomeContent)
	settings.PurchaseSubscriptionURL = resellersite.Rewrite(ctx, settings.PurchaseSubscriptionURL)
	settings.CustomMenuItems = resellersite.Rewrite(ctx, settings.CustomMenuItems)
	settings.CustomEndpoints = resellersite.Rewrite(ctx, settings.CustomEndpoints)
	settings.BalanceLowNotifyRechargeURL = resellersite.Rewrite(ctx, settings.BalanceLowNotifyRechargeURL)
	return settings, nil
}

func customerSiteAPIURL(ctx context.Context, configured string) (string, error) {
	origin := resellersite.CustomerOrigin(ctx)
	base, _ := url.Parse(origin)
	api, err := url.Parse(configured)
	if err != nil {
		return "", err
	}
	api.Scheme = base.Scheme
	api.Host = base.Host
	return api.String(), nil
}

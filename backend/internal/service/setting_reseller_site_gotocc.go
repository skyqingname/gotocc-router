package service

import (
	"context"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/resellersite"
)

func applyResellerSitePublicSettings(ctx context.Context, settings *PublicSettings) (*PublicSettings, error) {
	if !resellersite.IsCustomer(ctx) {
		return settings, nil
	}
	settings.ResellerInvitationRequired = true
	settings.InvitationCodeEnabled = true
	settings.PromoCodeEnabled = false
	settings.SiteLogo = resellersite.Rewrite(ctx, settings.SiteLogo)
	settings.APIBaseURL = resellersite.CustomerOrigin(ctx)
	settings.DocURL = resellersite.Rewrite(ctx, settings.DocURL)
	settings.ContactInfo = ""
	settings.HomeContent = resellersite.Rewrite(ctx, settings.HomeContent)
	settings.PurchaseSubscriptionURL = resellersite.Rewrite(ctx, settings.PurchaseSubscriptionURL)
	settings.CustomMenuItems = resellersite.Rewrite(ctx, settings.CustomMenuItems)
	settings.CustomEndpoints = "[]"
	settings.BalanceLowNotifyRechargeURL = resellersite.Rewrite(ctx, settings.BalanceLowNotifyRechargeURL)
	return settings, nil
}

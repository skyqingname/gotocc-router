package service

import (
	"context"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
)

type openAIImagesDirectModelRoutingCtxKey struct{}

func withOpenAIImagesDirectModelRouting(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAIImagesDirectModelRoutingCtxKey{}, true)
}

func openAIImagesDirectModelRoutingFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(openAIImagesDirectModelRoutingCtxKey{}).(bool)
	return enabled
}

func openAIAccountSupportsRequestedModel(ctx context.Context, account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	if openAIImagesDirectModelRoutingFromContext(ctx) {
		return account.IsModelDirectlySupported(requestedModel)
	}
	return account.IsModelSupported(requestedModel)
}

func openAIAccountTransportCompatible(cfg *config.Config, resolver OpenAIWSProtocolResolver, account *Account, requiredTransport OpenAIUpstreamTransport) bool {
	if requiredTransport == OpenAIUpstreamTransportAny || requiredTransport == OpenAIUpstreamTransportHTTPSSE {
		return true
	}
	if account == nil || resolver == nil {
		return false
	}
	if requiredTransport == OpenAIUpstreamTransportResponsesWebsocketV2Ingress {
		if cfg == nil || !cfg.Gateway.OpenAIWS.ModeRouterV2Enabled {
			return resolver.Resolve(account).Transport == OpenAIUpstreamTransportResponsesWebsocketV2
		}
		mode := account.ResolveOpenAIResponsesWebSocketV2Mode(cfg.Gateway.OpenAIWS.IngressModeDefault)
		switch mode {
		case OpenAIWSIngressModeCtxPool, OpenAIWSIngressModePassthrough, OpenAIWSIngressModeHTTPBridge, OpenAIWSIngressModeShared, OpenAIWSIngressModeDedicated:
			return true
		default:
			return false
		}
	}
	return resolver.Resolve(account).Transport == requiredTransport
}

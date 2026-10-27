package apicompat

// ChatResponseRoute describes this gateway's actual forwarding choice, not
// the supplier's internal region, deployment or model implementation.
type ChatResponseRoute struct {
	InboundEndpoint  string `json:"inbound_endpoint"`
	UpstreamEndpoint string `json:"upstream_endpoint"`
	UpstreamModel    string `json:"upstream_model"`
}

package service

func resolveImageRateMultiplier(apiKey *APIKey, effectiveGroupMultiplier float64) float64 {
	if quote := apiKey.ResellerPrice(); quote != nil {
		return quote.ImageRate
	}
	return resolveBaseImageRateMultiplier(apiKey, effectiveGroupMultiplier)
}

func resolveBaseImageRateMultiplier(apiKey *APIKey, effectiveGroupMultiplier float64) float64 {
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.ImageRateIndependent {
		if apiKey.Group.ImageRateMultiplier < 0 {
			return 0
		}
		return apiKey.Group.ImageRateMultiplier
	}
	return effectiveGroupMultiplier
}

func resolveVideoRateMultiplier(apiKey *APIKey, effectiveGroupMultiplier float64) float64 {
	if quote := apiKey.ResellerPrice(); quote != nil {
		return quote.VideoRate
	}
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.VideoRateIndependent {
		if apiKey.Group.VideoRateMultiplier < 0 {
			return 0
		}
		return apiKey.Group.VideoRateMultiplier
	}
	return effectiveGroupMultiplier
}

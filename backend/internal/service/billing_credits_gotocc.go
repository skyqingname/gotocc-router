package service

import "strings"

// Official face-value cards are used only by default pricing. Explicit group
// and channel prices already represent credits and retain their precedence.
func officialCreditPricing(model string) (*ModelPricing, bool) {
	model = strings.ToLower(strings.TrimSpace(model))
	if separator := strings.LastIndex(model, "/"); separator >= 0 {
		model = model[separator+1:]
	}
	if price, ok := officialCreditPrices[model]; ok {
		return &price, true
	}
	for prefix, name := range officialCreditPricePrefixes {
		if strings.HasPrefix(model, prefix) {
			price := officialCreditPrices[name]
			return &price, true
		}
	}
	return nil, false
}

func hasOfficialCreditPricing(model string) bool {
	_, ok := officialCreditPricing(model)
	return isDeepSeekModel(model) || ok
}

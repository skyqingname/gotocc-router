package service

import "math"

var contentModerationCategoryOrder = func() []string {
	categories := make([]string, 0, len(contentAuditJev.Risks))
	for _, risk := range contentAuditJev.Risks {
		categories = append(categories, risk.Category)
	}
	return categories
}()

func ContentModerationDefaultThresholds() map[string]float64 {
	thresholds := make(map[string]float64, len(contentAuditJev.Risks))
	for _, risk := range contentAuditJev.Risks {
		thresholds[risk.Category] = risk.Threshold
	}
	return thresholds
}

func ContentModerationCategories() []string {
	return append([]string(nil), contentModerationCategoryOrder...)
}

// OpenAI-compatible providers retain their detailed category keys. Merge these
// scores before threshold evaluation and persistence; historical rows stay intact.
func mergeContentModerationCategoryScores(scores map[string]float64) map[string]float64 {
	merged := cloneFloatMap(scores)
	for _, risk := range contentAuditJev.Risks {
		for _, category := range risk.ScoreCategories {
			if score, ok := scores[category]; ok {
				merged[risk.Category] = math.Max(merged[risk.Category], score)
			}
			if category != risk.Category {
				delete(merged, category)
			}
		}
	}
	for _, category := range contentAuditJev.RetiredCategories {
		delete(merged, category)
	}
	return merged
}

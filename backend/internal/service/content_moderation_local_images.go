package service

import (
	"context"
	"log/slog"

	"github.com/LuckyKuang/sub2api-plus/internal/auditcontent"
)

// CheckLocalAndImages is the gateway's non-Jev audit lane. It preserves the
// existing session, keyword, hash, scope, sampling, image-audit and side-effect
// rules, but never sends prompt text to the legacy moderation model, including
// in observe/shadow mode. Jev owns semantic text decisions in every gateway mode.
// Historical decoder and explicit administrative legacy probes are separate.
func (s *ContentModerationService) CheckLocalAndImages(ctx context.Context, input ContentModerationCheckInput) (*ContentModerationDecision, error) {
	allow := &ContentModerationDecision{Allowed: true, Action: ContentModerationActionAllow}
	logFailure := func(code string) {
		slog.Warn("content_moderation.local_images_failure",
			"request_id", input.RequestID, "endpoint", input.Endpoint,
			"protocol", input.Protocol, "stage", contentModerationAuditStage(input.Stage),
			"body_bytes", len(input.Body), "error_code", code)
	}
	if s == nil || s.settingRepo == nil || s.repo == nil {
		logFailure("service_unavailable")
		return allow, nil
	}
	runtimeSnapshot, err := s.loadRuntimeSnapshot(ctx)
	if err != nil {
		logFailure("config_load_failed")
		return allow, nil
	}
	cfg := runtimeSnapshot.config
	if !runtimeSnapshot.riskControlEnabled || cfg == nil || !cfg.Enabled || cfg.Mode == ContentModerationModeOff {
		return allow, nil
	}
	if blocked := s.lookupBlockedSession(ctx, cfg, input); blocked != nil {
		return blocked, nil
	}
	if !cfg.includesGroup(input.GroupID) || !cfg.includesModel(input.Model) {
		return allow, nil
	}
	s.extractionAttempted.Add(1)
	content, _, reasons, extractErr := extractContentModerationInput(input.Protocol, input.Body)
	incomplete := extractErr != nil || len(reasons) > 0
	if incomplete {
		s.extractionFailed.Add(1)
		slog.Warn("content_moderation.extraction_failed",
			"request_id", input.RequestID, "endpoint", input.Endpoint,
			"protocol", input.Protocol, "stage", contentModerationAuditStage(input.Stage),
			"body_bytes", len(input.Body), "error_code", "incomplete_content",
			"incomplete_reasons", auditcontent.SanitizeIncompleteReasons(reasons))
	}
	if content.IsEmpty() {
		if !incomplete { s.extractionEmpty.Add(1) }
		return allow, nil
	}
	if !incomplete { s.extractionSucceeded.Add(1) }
	content.Normalize()
	inputHash := content.Hash()
	if cfg.Mode == ContentModerationModePreBlock {
		if cfg.KeywordBlockingMode != ContentModerationKeywordModeAPIOnly && len(cfg.BlockedKeywords) > 0 {
			if keyword, hit := runtimeSnapshot.matchBlockedKeyword(content.Text); hit {
				s.recordPreBlockSyncMetric(0, ContentModerationActionKeywordBlock)
				scores := map[string]float64{contentModerationKeywordCategory: 1}
				record := s.buildLog(input, cfg, ContentModerationActionKeywordBlock, true, contentModerationKeywordCategory, 1, scores, content.ExcerptText(), nil, nil, "")
				record.MatchedKeyword = keyword
				s.enqueueRecord(input, cfg, record, inputHash, false, true)
				return &ContentModerationDecision{
					Blocked: true, Flagged: true, Message: cfg.BlockMessage, StatusCode: cfg.BlockStatus,
					HighestCategory: contentModerationKeywordCategory, HighestScore: 1,
					CategoryScores: scores, Action: ContentModerationActionKeywordBlock,
				}, nil
			}
		}
		if cfg.KeywordBlockingMode == ContentModerationKeywordModeKeywordOnly && len(content.Images) == 0 {
			s.recordPreBlockSyncMetric(0, ContentModerationActionAllow)
			return allow, nil
		}
	}
	if cfg.PreHashCheckEnabled && s.hashCache != nil {
		matched, lookupErr := s.hashCache.HasFlaggedInputHash(ctx, inputHash)
		if lookupErr != nil { logFailure("hash_check_failed") }
		if matched {
			if cfg.Mode == ContentModerationModePreBlock { s.recordPreBlockSyncMetric(0, ContentModerationActionHashBlock) }
			scores := map[string]float64{"hash": 1}
			record := s.buildLog(input, cfg, ContentModerationActionHashBlock, true, "hash", 1, scores, content.ExcerptText(), nil, nil, "")
			s.enqueueRecord(input, cfg, record, inputHash, false, false)
			return &ContentModerationDecision{
				Blocked: true, Flagged: true, Message: cfg.BlockMessage, StatusCode: cfg.BlockStatus,
				InputHash: inputHash, Action: ContentModerationActionHashBlock,
			}, nil
		}
	}
	// In particular, do not enqueue a text shadow task. The images-only content
	// object is what the existing worker/HTTP client receives, not input.Body.
	images := ContentModerationInput{Images: append([]string(nil), content.Images...)}
	images.Normalize()
	if images.IsEmpty() { return allow, nil }
	if cfg.Mode != ContentModerationModePreBlock && !cfg.shouldSample(inputHash) { return allow, nil }
	if len(cfg.apiKeys()) == 0 {
		logFailure("no_image_audit_api_keys")
		return allow, nil
	}
	if cfg.Mode == ContentModerationModeObserve {
		s.enqueueAsync(input, cfg, images, images.Hash(), false)
		return allow, nil
	}
	return s.checkSync(ctx, input, cfg, images, images.Hash(), nil, true, true), nil
}

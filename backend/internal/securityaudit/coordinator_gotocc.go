package securityaudit

// NewContentAuditCoordinator wires GoToCC's unified content audit: every
// ingress request crosses Content Moderation only, and Prompt Audit no longer
// joins the request path. Its admin data and history remain readable.
func NewContentAuditCoordinator(legacy LegacyEngine) *Coordinator {
	return &Coordinator{legacy: legacy}
}

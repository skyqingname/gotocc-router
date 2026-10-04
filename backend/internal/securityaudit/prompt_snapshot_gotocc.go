package securityaudit

import (
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/auditcontent"
)

type promptSegment struct {
	source auditcontent.Source
	text   string
	user   bool
	role   string
}

// ExtractBlockingPromptSnapshot builds the synchronous guard input.
// When latestTurnOnly is true, the scan window is the latest user turn plus the
// nearest preceding assistant/model turn. When it is false, blocking uses the
// same client-controlled transcript as async review. A request without user
// content cannot be narrowed safely and falls back to that full transcript.
func ExtractBlockingPromptSnapshot(req Request, latestTurnOnly bool) (PromptSnapshot, error) {
	snapshot, _, err := extractPromptSnapshotWithDiagnostics(req, latestTurnOnly)
	return snapshot, err
}

func promptAuditScanSegments(values []promptSegment, protocol string, latestTurnOnly bool) []string {
	if isSystemOnePromptProtocol(protocol) {
		// System One is a single evaluation, not a chat turn. Keep its state
		// first in both audit modes, then retain the other text in canonical order.
		segments := promptSegmentTexts(normalizedPromptSegments(values))
		if len(segments) > 1 {
			segments = append([]string{segments[len(segments)-1]}, segments[:len(segments)-1]...)
		}
		return segments
	}
	if latestTurnOnly {
		return blockingSegmentsLatestUserAndPreviousOutput(values)
	}
	return normalizeSegmentsLatestUserFirst(values)
}

func promptSegmentsFromAuditContent(document auditcontent.Document, protocol string) []promptSegment {
	allowRolelessMessage := promptAuditAllowsRolelessMessage(protocol)
	systemOne := isSystemOnePromptProtocol(protocol)
	segments := make([]promptSegment, 0, len(document.Segments))
	for _, segment := range document.Segments {
		if !isPromptAuditClientControlledSegment(segment, allowRolelessMessage) {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(segment.Role))
		user := role == "user" && segment.Source != auditcontent.SourceToolOutput
		if role == "" && ((segment.Source == auditcontent.SourceMessage && allowRolelessMessage) ||
			segment.Source == auditcontent.SourceSearchQuery ||
			segment.Source == auditcontent.SourceEmbeddingInput ||
			segment.Source == auditcontent.SourceMediaPrompt) {
			user = true
			role = "user"
		}
		if role == "" {
			switch segment.Source {
			case auditcontent.SourceInstruction, auditcontent.SourcePromptVariable:
				role = "system"
			case auditcontent.SourceToolCall, auditcontent.SourceToolDefinition, auditcontent.SourceToolOutput:
				role = "tool"
			case auditcontent.SourceReasoning:
				role = "assistant"
			}
		}
		if segment.Source == auditcontent.SourceToolOutput {
			role = "tool"
		}
		segText := segment.Text
		if user {
			if !systemOne {
				segText = stripPromptAuditClientWrapperBlocks(segText)
			}
			if segText == "" {
				continue
			}
		}
		segments = append(segments, promptSegment{
			text: segText, source: segment.Source,
			user: user,
			role: role,
		})
	}
	return segments
}

func isPromptAuditClientControlledSegment(segment auditcontent.Segment, allowRolelessMessage bool) bool {
	switch segment.Source {
	case auditcontent.SourceSearchQuery, auditcontent.SourceEmbeddingInput, auditcontent.SourceMediaPrompt,
		auditcontent.SourceInstruction, auditcontent.SourcePromptVariable,
		auditcontent.SourceToolCall, auditcontent.SourceToolDefinition, auditcontent.SourceToolOutput,
		auditcontent.SourceReasoning:
		return true
	case auditcontent.SourceMessage:
		role := strings.ToLower(strings.TrimSpace(segment.Role))
		switch role {
		case "user", "system", "developer", "assistant", "tool", "model":
			return true
		case "":
			return allowRolelessMessage
		default:
			return false
		}
	default:
		return false
	}
}

func promptAuditAllowsRolelessMessage(protocol string) bool {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "openai_responses", "openai_live", "gemini":
		return true
	default:
		return false
	}
}

func normalizeSegmentsLatestUserFirst(values []promptSegment) []string {
	normalized := normalizedPromptSegments(values)
	if len(normalized) == 0 {
		return nil
	}
	result := make([]string, 0, len(normalized))
	for index := len(normalized) - 1; index >= 0; index-- {
		result = append(result, normalized[index].text)
	}
	return result
}

// blockingSegmentsLatestUserAndPreviousOutput limits synchronous guard input
// to the current user turn and the nearest preceding assistant/model turn.
// A request without user content cannot be narrowed safely and falls back to
// the complete client-controlled transcript.
func blockingSegmentsLatestUserAndPreviousOutput(values []promptSegment) []string {
	normalized := normalizedPromptSegments(values)
	latestUserStart := latestUserSegmentStart(normalized)
	if latestUserStart < 0 {
		return normalizeSegmentsLatestUserFirst(values)
	}
	latestUserEnd := latestUserStart
	for latestUserEnd < len(normalized) && isUserSegment(normalized[latestUserEnd]) {
		latestUserEnd++
	}
	currentUserText := make([]string, 0, latestUserEnd-latestUserStart)
	for _, segment := range normalized[latestUserStart:latestUserEnd] {
		currentUserText = append(currentUserText, segment.text)
	}
	selected := []promptSegment{{text: strings.Join(currentUserText, "\n\n"), user: true, role: "user"}}
	for _, segment := range normalized[latestUserEnd:] {
		if segment.source == auditcontent.SourceToolOutput {
			selected = append(selected, segment)
		}
	}
	for index := latestUserStart - 1; index >= 0; index-- {
		if !isAssistantOutputSegment(normalized[index]) {
			continue
		}
		start := index
		for start > 0 && isAssistantOutputSegment(normalized[start-1]) {
			start--
		}
		selected = append(selected, normalized[start:index+1]...)
		break
	}
	return promptSegmentTexts(selected)
}

func normalizedPromptSegments(values []promptSegment) []promptSegment {
	normalized := make([]promptSegment, 0, len(values))
	for _, value := range values {
		value.text = strings.TrimSpace(value.text)
		if value.text != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func latestUserSegmentStart(values []promptSegment) int {
	latest := -1
	for index := len(values) - 1; index >= 0; index-- {
		if isUserSegment(values[index]) {
			latest = index
			break
		}
	}
	for latest > 0 && isUserSegment(values[latest-1]) {
		latest--
	}
	return latest
}

func isUserSegment(segment promptSegment) bool {
	return segment.user || segment.role == "user"
}

func isAssistantOutputSegment(segment promptSegment) bool {
	return segment.role == "assistant" || segment.role == "model"
}

func promptSegmentTexts(values []promptSegment) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.text)
	}
	return result
}

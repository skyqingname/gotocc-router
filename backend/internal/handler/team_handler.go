package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// TeamHandler 处理当前用户所在团队的生命周期和成员管理请求。
type TeamHandler struct {
	service *service.TeamService
}

func NewTeamHandler(teamService *service.TeamService) *TeamHandler {
	return &TeamHandler{service: teamService}
}

func teamSubject(c *gin.Context) (middleware.AuthSubject, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
	}
	return subject, ok
}

func (h *TeamHandler) GetCurrent(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	teamCtx, err := h.service.GetCurrent(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, teamCtx)
}

func (h *TeamHandler) Create(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	teamCtx, err := h.service.Create(c.Request.Context(), subject.UserID, req.Name)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, teamCtx)
}

func (h *TeamHandler) Update(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	teamCtx, err := h.service.UpdateName(c.Request.Context(), subject.UserID, req.Name)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, teamCtx)
}

// UpdateDefaultMemberLimits 保存后续新成员继承的团队级默认限额。
func (h *TeamHandler) UpdateDefaultMemberLimits(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		Daily   float64 `json:"default_daily_limit_usd"`
		Weekly  float64 `json:"default_weekly_limit_usd"`
		Monthly float64 `json:"default_monthly_limit_usd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	teamCtx, err := h.service.UpdateDefaultMemberLimits(c.Request.Context(), subject.UserID, req.Daily, req.Weekly, req.Monthly)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, teamCtx)
}

func (h *TeamHandler) SetStatus(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=active suspended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	teamCtx, err := h.service.SetStatus(c.Request.Context(), subject.UserID, req.Status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, teamCtx)
}

func (h *TeamHandler) ListMembers(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	members, err := h.service.ListMembers(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, members)
}

func parseTeamUsageQuery(c *gin.Context) (service.TeamUsageQuery, error) {
	query := service.TeamUsageQuery{}
	parseTime := func(value string, end bool) (time.Time, error) {
		value = strings.TrimSpace(value)
		if value == "" {
			return time.Time{}, nil
		}
		if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return parsed, nil
		}
		for _, candidate := range []struct {
			layout   string
			dateOnly bool
		}{
			{layout: "2006-01-02", dateOnly: true},
			{layout: "2006-01-02T15:04:05"},
			{layout: "2006-01-02T15:04"},
			{layout: "2006-01-02 15:04:05"},
			{layout: "2006-01-02 15:04"},
		} {
			parsed, err := timezone.ParseInUserLocation(candidate.layout, value, "")
			if err != nil {
				continue
			}
			if end && candidate.dateOnly {
				parsed = parsed.AddDate(0, 0, 1)
			}
			return parsed, nil
		}
		return time.Time{}, fmt.Errorf("invalid datetime %q", value)
	}
	var err error
	if query.From, err = parseTime(c.Query("from"), false); err != nil {
		return query, err
	}
	if query.To, err = parseTime(c.Query("to"), true); err != nil {
		return query, err
	}
	parseID := func(name string) (*int64, error) {
		value := strings.TrimSpace(c.Query(name))
		if value == "" {
			return nil, nil
		}
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return nil, strconv.ErrSyntax
		}
		return &id, nil
	}
	if query.ActorUserID, err = parseID("member_id"); err != nil {
		return query, err
	}
	if query.APIKeyID, err = parseID("api_key_id"); err != nil {
		return query, err
	}
	query.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	query.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	return query, nil
}

// GetUsageSummary 返回团队作用域的汇总和趋势。
func (h *TeamHandler) GetUsageSummary(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	query, err := parseTeamUsageQuery(c)
	if err != nil {
		response.BadRequest(c, "Invalid team usage query")
		return
	}
	result, err := h.service.GetUsageSummary(c.Request.Context(), subject.UserID, query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ListMemberUsageSeries 返回当前和历史成员的一次性趋势汇总。
func (h *TeamHandler) ListMemberUsageSeries(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	query, err := parseTeamUsageQuery(c)
	if err != nil {
		response.BadRequest(c, "Invalid team usage query")
		return
	}
	result, err := h.service.ListMemberUsageSeries(c.Request.Context(), subject.UserID, query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ListUsageLogs 返回团队作用域的分页用量明细。
func (h *TeamHandler) ListUsageLogs(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	query, err := parseTeamUsageQuery(c)
	if err != nil {
		response.BadRequest(c, "Invalid team usage query")
		return
	}
	result, err := h.service.ListUsageLogs(c.Request.Context(), subject.UserID, query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ListTeamKeys 返回当前成员可见的团队 Key 元数据。
func (h *TeamHandler) ListTeamKeys(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	items, err := h.service.ListTeamKeys(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *TeamHandler) DisableTeamKey(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || keyID <= 0 {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	if err := h.service.DisableTeamKey(c.Request.Context(), subject.UserID, keyID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"disabled": true})
}

func (h *TeamHandler) EnableTeamKey(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || keyID <= 0 {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	if err := h.service.EnableTeamKey(c.Request.Context(), subject.UserID, keyID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"enabled": true})
}

func (h *TeamHandler) DeleteTeamKey(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || keyID <= 0 {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	if err := h.service.DeleteTeamKey(c.Request.Context(), subject.UserID, keyID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *TeamHandler) Invite(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	ctx := service.WithTeamFrontendRequest(c.Request.Context(), c.GetHeader("Origin"), c.Request.Host)
	invitation, err := h.service.Invite(ctx, subject.UserID, req.Email)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, invitation)
}

func (h *TeamHandler) ListInvitations(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	items, err := h.service.ListInvitations(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *TeamHandler) ReissueInvitation(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invitation ID")
		return
	}
	ctx := service.WithTeamFrontendRequest(c.Request.Context(), c.GetHeader("Origin"), c.Request.Host)
	item, err := h.service.ReissueInvitation(ctx, subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *TeamHandler) RevokeInvitation(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invitation ID")
		return
	}
	if err := h.service.RevokeInvitation(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"revoked": true})
}

// PreviewInvitation 返回当前登录用户有权确认的邀请摘要。
func (h *TeamHandler) PreviewInvitation(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	preview, err := h.service.PreviewInvitation(c.Request.Context(), subject.UserID, strings.TrimSpace(req.Token))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, preview)
}

func (h *TeamHandler) ResolveInvitation(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		Token      string `json:"token" binding:"required"`
		Resolution string `json:"resolution" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	teamCtx, err := h.service.ResolveInvitation(c.Request.Context(), subject.UserID, req.Token, req.Resolution)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, teamCtx)
}

func (h *TeamHandler) RemoveMember(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	memberID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid member ID")
		return
	}
	if err := h.service.RemoveMember(c.Request.Context(), subject.UserID, memberID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"removed": true})
}

func (h *TeamHandler) Leave(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	if err := h.service.Leave(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"left": true})
}

func (h *TeamHandler) UpdateMemberLimits(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	memberID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid member ID")
		return
	}
	var req struct {
		Daily   float64 `json:"daily_limit_usd"`
		Weekly  float64 `json:"weekly_limit_usd"`
		Monthly float64 `json:"monthly_limit_usd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.service.UpdateMemberLimits(c.Request.Context(), subject.UserID, memberID, req.Daily, req.Weekly, req.Monthly); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}

func (h *TeamHandler) ResetMemberUsage(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	memberID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid member ID")
		return
	}
	var req struct {
		Daily   bool `json:"daily"`
		Weekly  bool `json:"weekly"`
		Monthly bool `json:"monthly"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.service.ResetMemberUsage(c.Request.Context(), subject.UserID, memberID, req.Daily, req.Weekly, req.Monthly); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"reset": true})
}

func (h *TeamHandler) StartOwnershipTransfer(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		TargetUserID int64 `json:"target_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	ctx := service.WithTeamFrontendRequest(c.Request.Context(), c.GetHeader("Origin"), c.Request.Host)
	transfer, err := h.service.StartOwnershipTransfer(ctx, subject.UserID, req.TargetUserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, transfer)
}

func (h *TeamHandler) ResolveOwnershipTransfer(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		Token      string `json:"token" binding:"required"`
		Resolution string `json:"resolution" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	teamCtx, err := h.service.ResolveOwnershipTransfer(c.Request.Context(), subject.UserID, req.Token, req.Resolution)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, teamCtx)
}

func (h *TeamHandler) CancelOwnershipTransfer(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	if err := h.service.CancelOwnershipTransfer(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"cancelled": true})
}

func (h *TeamHandler) Dissolve(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	if err := h.service.Dissolve(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"dissolved": true})
}

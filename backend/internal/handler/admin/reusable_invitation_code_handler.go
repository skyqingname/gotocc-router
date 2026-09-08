package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

type ReusableInvitationCodeHandler struct {
	repo service.ReusableInvitationCodeRepository
}

func NewReusableInvitationCodeHandler(repo service.ReusableInvitationCodeRepository) *ReusableInvitationCodeHandler {
	return &ReusableInvitationCodeHandler{repo: repo}
}

type CreateReusableInvitationCodeRequest struct {
	Code      string     `json:"code" binding:"required,min=3,max=64"`
	MaxUses   int        `json:"max_uses" binding:"omitempty,min=0"`
	ExpiresAt *time.Time `json:"expires_at"`
	Notes     string     `json:"notes"`
}

type ReusableInvitationCodeResponse struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	Status    string     `json:"status"`
	MaxUses   int        `json:"max_uses"`
	UsedCount int        `json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Notes     string     `json:"notes"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ReusableInvitationCodeUseResponse struct {
	ID         int64     `json:"id"`
	CodeID     int64     `json:"code_id"`
	UserID     int64     `json:"user_id"`
	Email      string    `json:"email"`
	AuthSource string    `json:"auth_source"`
	UsedAt     time.Time `json:"used_at"`
}

func (h *ReusableInvitationCodeHandler) Create(c *gin.Context) {
	if h == nil || h.repo == nil {
		response.InternalError(c, "reusable invitation code repository not configured")
		return
	}
	var req CreateReusableInvitationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if len(req.Code) < 3 || len(req.Code) > 64 {
		response.BadRequest(c, "code length must be between 3 and 64")
		return
	}
	if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt.UTC()
		if !expiresAt.After(time.Now().UTC()) {
			response.BadRequest(c, "expires_at must be in the future")
			return
		}
		req.ExpiresAt = &expiresAt
	}
	code := &service.ReusableInvitationCode{
		Code: req.Code, Status: service.ReusableInvitationCodeStatusActive,
		MaxUses: req.MaxUses, ExpiresAt: req.ExpiresAt, Notes: strings.TrimSpace(req.Notes),
	}
	if err := h.repo.Create(c.Request.Context(), code); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, reusableInvitationCodeToResponse(code))
}

func (h *ReusableInvitationCodeHandler) List(c *gin.Context) {
	if h == nil || h.repo == nil {
		response.InternalError(c, "reusable invitation code repository not configured")
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page: page, PageSize: pageSize,
		SortBy: c.DefaultQuery("sort_by", "id"), SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
	codes, result, err := h.repo.List(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]ReusableInvitationCodeResponse, 0, len(codes))
	for i := range codes {
		out = append(out, reusableInvitationCodeToResponse(&codes[i]))
	}
	if result == nil {
		response.Paginated(c, out, int64(len(out)), page, pageSize)
		return
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

func (h *ReusableInvitationCodeHandler) Disable(c *gin.Context) {
	if h == nil || h.repo == nil {
		response.InternalError(c, "reusable invitation code repository not configured")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid reusable invitation code ID")
		return
	}
	code, err := h.repo.Disable(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, reusableInvitationCodeToResponse(code))
}

func (h *ReusableInvitationCodeHandler) ListUses(c *gin.Context) {
	if h == nil || h.repo == nil {
		response.InternalError(c, "reusable invitation code repository not configured")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid reusable invitation code ID")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			response.BadRequest(c, "Invalid limit")
			return
		}
	}
	uses, err := h.repo.ListUsesByCodeID(c.Request.Context(), id, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]ReusableInvitationCodeUseResponse, 0, len(uses))
	for i := range uses {
		out = append(out, reusableInvitationCodeUseToResponse(&uses[i]))
	}
	response.Success(c, out)
}

func reusableInvitationCodeToResponse(code *service.ReusableInvitationCode) ReusableInvitationCodeResponse {
	if code == nil {
		return ReusableInvitationCodeResponse{}
	}
	return ReusableInvitationCodeResponse{
		ID: code.ID, Code: code.Code, Status: code.Status, MaxUses: code.MaxUses,
		UsedCount: code.UsedCount, ExpiresAt: code.ExpiresAt, Notes: code.Notes,
		CreatedAt: code.CreatedAt, UpdatedAt: code.UpdatedAt,
	}
}

func reusableInvitationCodeUseToResponse(use *service.ReusableInvitationCodeUse) ReusableInvitationCodeUseResponse {
	if use == nil {
		return ReusableInvitationCodeUseResponse{}
	}
	return ReusableInvitationCodeUseResponse{
		ID: use.ID, CodeID: use.CodeID, UserID: use.UserID, Email: use.Email,
		AuthSource: use.AuthSource, UsedAt: use.UsedAt,
	}
}

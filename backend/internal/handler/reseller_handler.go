package handler

import (
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/reseller"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	middleware2 "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

type ResellerHandler struct{ service *service.ResellerService }

func NewResellerHandler(s *service.ResellerService) *ResellerHandler { return &ResellerHandler{s} }
func (h *ResellerHandler) owner(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	if _, err := h.service.RequireEnabled(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return 0, false
	}
	return subject.UserID, true
}
func (h *ResellerHandler) Access(c *gin.Context) {
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	p, err := h.service.Repo.Profile(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"enabled": p.Enabled})
}
func (h *ResellerHandler) Overview(c *gin.Context) {
	id, ok := h.owner(c)
	if !ok {
		return
	}
	p, e := h.service.AdminProfile(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	summary, e := h.service.Repo.Summary(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"profile": p, "summary": summary})
}
func (h *ResellerHandler) Customers(c *gin.Context) {
	id, ok := h.owner(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	rows, total, e := h.service.Repo.Customers(c.Request.Context(), id, c.Query("search"), page, reseller.PageSize)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"items": rows, "total": total, "page": page, "page_size": reseller.PageSize})
}
func (h *ResellerHandler) CustomerNotes(c *gin.Context) {
	id, ok := h.owner(c)
	if !ok {
		return
	}
	customerID, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil {
		response.BadRequest(c, "Invalid customer")
		return
	}
	var input struct {
		Notes string `json:"notes"`
	}
	if c.ShouldBindJSON(&input) != nil {
		response.BadRequest(c, "Invalid notes")
		return
	}
	if e = h.service.Repo.UpdateNotes(c.Request.Context(), id, customerID, input.Notes); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"updated": true})
}
func (h *ResellerHandler) Prices(c *gin.Context) {
	id, ok := h.owner(c)
	if !ok {
		return
	}
	prices, e := h.service.Repo.Prices(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	groups, e := h.service.Groups(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	profile, e := h.service.Repo.Profile(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	out := []gin.H{}
	for _, g := range groups {
		out = append(out, gin.H{"id": g.ID, "name": g.Name, "platform": g.Platform, "base_multiplier": g.RateMultiplier, "image_multiplier": g.ImageRateMultiplier, "image_independent": g.ImageRateIndependent, "video_multiplier": g.VideoRateMultiplier, "video_independent": g.VideoRateIndependent})
	}
	response.Success(c, gin.H{"prices": prices, "groups": out, "default_multiplier": profile.DefaultMultiplier})
}
func (h *ResellerHandler) SavePrices(c *gin.Context) {
	id, ok := h.owner(c)
	if !ok {
		return
	}
	var in struct {
		CustomerID *int64                  `json:"customer_id"`
		Multiplier *float64                `json:"multiplier"`
		Groups     []service.ResellerPrice `json:"groups"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, "Invalid prices")
		return
	}
	if e := h.service.SetPrices(c.Request.Context(), id, in.CustomerID, in.Multiplier, in.Groups); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"updated": true})
}
func (h *ResellerHandler) Earnings(c *gin.Context) {
	id, ok := h.owner(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	rows, total, e := h.service.Repo.Earnings(c.Request.Context(), id, page, reseller.PageSize)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"items": rows, "total": total, "page": page, "page_size": reseller.PageSize})
}
func (h *ResellerHandler) AdminProfile(c *gin.Context) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "Invalid user")
		return
	}
	p, e := h.service.AdminProfile(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, p)
}
func (h *ResellerHandler) AdminSave(c *gin.Context) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "Invalid user")
		return
	}
	var in struct {
		Enabled     bool       `json:"enabled"`
		RebateRates []*float64 `json:"rebate_rates"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, "Invalid reseller settings")
		return
	}
	p, e := h.service.SaveProfile(c.Request.Context(), id, in.Enabled, in.RebateRates)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, p)
}

// A snapshot is attached after authentication and before request-specific routing.
// No price or ownership is read again during settlement.
func (h *ResellerHandler) PricingContext(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		c.Next()
		return
	}
	prices, e := h.service.Repo.Pricing(c.Request.Context(), subject.UserID)
	if e != nil {
		response.Error(c, 503, "无法读取当前客户价格，请稍后重试")
		c.Abort()
		return
	}
	c.Request = c.Request.WithContext(service.WithResellerPrices(c.Request.Context(), subject.UserID, prices))
	if key, ok := middleware2.GetAPIKeyFromContext(c); ok {
		key.ResellerPrices = prices
	}
	c.Next()
}

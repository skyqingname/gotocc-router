package handler

import (
	"strconv"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/handler/dto"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

type resellerAnnouncementDTO struct {
	*dto.Announcement
	SourceAnnouncementID *int64 `json:"source_announcement_id"`
}

type resellerAnnouncementReviewDTO struct {
	*dto.Announcement
	ReviewStatus string     `json:"review_status"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
}

func (h *ResellerHandler) CommunicationSettings(c *gin.Context) {
	ownerID, ok := h.owner(c)
	if !ok {
		return
	}
	var input service.ResellerCommunicationSettings
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请填写有效的客户展示设置")
		return
	}
	profile, err := h.service.SaveCommunicationSettings(c.Request.Context(), ownerID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, profile)
}

func (h *ResellerHandler) Announcements(c *gin.Context) {
	ownerID, ok := h.owner(c)
	if !ok {
		return
	}
	items, err := h.service.Announcements(c.Request.Context(), ownerID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]resellerAnnouncementDTO, 0, len(items))
	for i := range items {
		out = append(out, resellerAnnouncementDTO{dto.AnnouncementFromService(&items[i].Announcement), items[i].SourceAnnouncementID})
	}
	response.Success(c, out)
}

func (h *ResellerHandler) SaveAnnouncement(c *gin.Context) {
	ownerID, ok := h.owner(c)
	if !ok {
		return
	}
	id := int64(0)
	if raw := c.Param("id"); raw != "" {
		var err error
		id, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "公告编号无效")
			return
		}
	}
	var input service.ResellerAnnouncementInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请填写标题、正文和公告状态")
		return
	}
	item, err := h.service.SaveAnnouncement(c.Request.Context(), ownerID, id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, resellerAnnouncementDTO{dto.AnnouncementFromService(&item.Announcement), item.SourceAnnouncementID})
}

func (h *ResellerHandler) AnnouncementStatus(c *gin.Context) {
	ownerID, ok := h.owner(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "公告编号无效")
		return
	}
	var input struct {
		Status string `json:"status" binding:"required,oneof=draft active archived"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "公告状态无效")
		return
	}
	if err := h.service.SetAnnouncementStatus(c.Request.Context(), ownerID, id, input.Status); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}

func (h *ResellerHandler) MainAnnouncementReviews(c *gin.Context) {
	ownerID, ok := h.owner(c)
	if !ok {
		return
	}
	items, err := h.service.MainAnnouncementReviews(c.Request.Context(), ownerID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]resellerAnnouncementReviewDTO, 0, len(items))
	for i := range items {
		out = append(out, resellerAnnouncementReviewDTO{dto.AnnouncementFromService(&items[i].Announcement), items[i].ReviewStatus, items[i].ReviewedAt})
	}
	response.Success(c, out)
}

func (h *ResellerHandler) ReviewMainAnnouncement(c *gin.Context) {
	ownerID, ok := h.owner(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "公告编号无效")
		return
	}
	var input struct {
		SourceUpdatedAt time.Time `json:"source_updated_at" binding:"required"`
		Status          string    `json:"status" binding:"required,oneof=approved rejected"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请重新查看公告后选择审核结果")
		return
	}
	if err := h.service.ReviewMainAnnouncement(c.Request.Context(), ownerID, id, input.SourceUpdatedAt, input.Status); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"reviewed": true})
}

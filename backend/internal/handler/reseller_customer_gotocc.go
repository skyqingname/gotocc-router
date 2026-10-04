package handler

import (
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/handler/dto"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/reseller"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	middleware2 "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *ResellerHandler) ownedCustomer(c *gin.Context) (int64, int64, bool) {
	ownerID, ok := h.owner(c)
	if !ok {
		return 0, 0, false
	}
	customerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || customerID <= 0 {
		response.BadRequest(c, "Invalid customer")
		return 0, 0, false
	}
	if err = h.service.RequireCustomer(c.Request.Context(), ownerID, customerID); err != nil {
		response.ErrorFrom(c, err)
		return 0, 0, false
	}
	return ownerID, customerID, true
}

func resellerPagination(c *gin.Context) pagination.PaginationParams {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	return pagination.PaginationParams{Page: page, PageSize: reseller.PageSize, SortOrder: pagination.SortOrderDesc}
}

func (h *ResellerHandler) Customer(c *gin.Context) {
	ownerID, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	user, err := h.service.Customer(c.Request.Context(), ownerID, customerID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UserFromService(user))
}

func (h *ResellerHandler) CreateCustomer(c *gin.Context) {
	ownerID, ok := h.owner(c)
	if !ok {
		return
	}
	h.saveCustomer(c, ownerID, 0)
}

func (h *ResellerHandler) UpdateCustomer(c *gin.Context) {
	ownerID, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	h.saveCustomer(c, ownerID, customerID)
}

func (h *ResellerHandler) saveCustomer(c *gin.Context, ownerID, customerID int64) {
	var input service.ResellerCustomerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请填写有效的客户资料、状态和使用限额")
		return
	}
	user, err := h.service.SaveCustomer(c.Request.Context(), ownerID, customerID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UserFromService(user))
}

func (h *ResellerHandler) CustomerCredit(c *gin.Context) {
	ownerID, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	var input service.ResellerCreditInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请填写有效的额度操作")
		return
	}
	entry, err := h.service.ChangeCredit(c.Request.Context(), ownerID, customerID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, entry)
}

func (h *ResellerHandler) CustomerCreditEntries(c *gin.Context) {
	ownerID, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	p := resellerPagination(c)
	entries, total, err := h.service.Repo.CustomerCreditEntries(c.Request.Context(), ownerID, customerID, p.Page, p.PageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": entries, "total": total, "page": p.Page, "page_size": p.PageSize})
}

func (h *ResellerHandler) MyCredits(c *gin.Context) {
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	account, err := h.service.Repo.CustomerAccount(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if account == nil {
		response.NotFound(c, "当前用户没有站长客户额度")
		return
	}
	p := resellerPagination(c)
	entries, total, err := h.service.Repo.CustomerCreditEntries(c.Request.Context(), account.OwnerID, subject.UserID, p.Page, p.PageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		items = append(items, gin.H{"id": entry.ID, "kind": entry.Kind, "amount": entry.Amount, "frozen_amount": entry.FrozenAmount, "balance_after": entry.BalanceAfter, "frozen_after": entry.FrozenAfter, "model": entry.Model, "created_at": entry.CreatedAt})
	}
	response.Success(c, gin.H{"account": account, "items": items, "total": total, "page": p.Page, "page_size": p.PageSize})
}

func (h *ResellerHandler) CustomerKeys(c *gin.Context) {
	ownerID, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	keys, total, err := h.service.CustomerKeys(c.Request.Context(), ownerID, customerID, resellerPagination(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]*dto.APIKey, 0, len(keys))
	for i := range keys {
		item := dto.APIKeyFromService(&keys[i])
		item.Key = ""
		items = append(items, item)
	}
	p := resellerPagination(c)
	response.Success(c, gin.H{"items": items, "total": total, "page": p.Page, "page_size": p.PageSize})
}

func (h *ResellerHandler) CreateCustomerKey(c *gin.Context) {
	ownerID, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	var input service.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid API key")
		return
	}
	key, err := h.service.CreateCustomerKey(c.Request.Context(), ownerID, customerID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.APIKeyFromService(key))
}

func (h *ResellerHandler) UpdateCustomerKey(c *gin.Context) {
	ownerID, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	keyID, err := strconv.ParseInt(c.Param("key_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key")
		return
	}
	var input service.UpdateAPIKeyRequest
	if err = c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid API key")
		return
	}
	key, err := h.service.UpdateCustomerKey(c.Request.Context(), ownerID, customerID, keyID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	item := dto.APIKeyFromService(key)
	item.Key = ""
	response.Success(c, item)
}

func (h *ResellerHandler) DeleteCustomerKey(c *gin.Context) {
	ownerID, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	keyID, err := strconv.ParseInt(c.Param("key_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key")
		return
	}
	if err = h.service.DeleteCustomerKey(c.Request.Context(), ownerID, customerID, keyID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *ResellerHandler) CustomerUsage(c *gin.Context) {
	_, customerID, ok := h.ownedCustomer(c)
	if !ok {
		return
	}
	p := resellerPagination(c)
	logs, result, err := h.usage.ListByUser(c.Request.Context(), customerID, p)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]*dto.UsageLog, 0, len(logs))
	for i := range logs {
		items = append(items, dto.UsageLogFromService(&logs[i]))
	}
	response.Success(c, gin.H{"items": items, "total": result.Total, "page": p.Page, "page_size": p.PageSize})
}

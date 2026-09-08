package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type reusableInvitationCodeAdminRepoStub struct {
	created  *service.ReusableInvitationCode
	list     []service.ReusableInvitationCode
	disabled *service.ReusableInvitationCode
	uses     []service.ReusableInvitationCodeUse
}

func (s *reusableInvitationCodeAdminRepoStub) Create(_ context.Context, code *service.ReusableInvitationCode) error {
	now := time.Now().UTC()
	code.ID, code.CreatedAt, code.UpdatedAt = 7, now, now
	cloned := *code
	s.created = &cloned
	return nil
}
func (*reusableInvitationCodeAdminRepoStub) GetByID(context.Context, int64) (*service.ReusableInvitationCode, error) {
	panic("unexpected GetByID call")
}
func (*reusableInvitationCodeAdminRepoStub) GetByCode(context.Context, string) (*service.ReusableInvitationCode, error) {
	panic("unexpected GetByCode call")
}
func (*reusableInvitationCodeAdminRepoStub) GetUsableByCode(context.Context, string) (*service.ReusableInvitationCode, error) {
	panic("unexpected GetUsableByCode call")
}
func (s *reusableInvitationCodeAdminRepoStub) List(_ context.Context, params pagination.PaginationParams) ([]service.ReusableInvitationCode, *pagination.PaginationResult, error) {
	return s.list, &pagination.PaginationResult{
		Total: int64(len(s.list)), Page: params.Page, PageSize: params.PageSize, Pages: 1,
	}, nil
}
func (s *reusableInvitationCodeAdminRepoStub) Disable(_ context.Context, id int64) (*service.ReusableInvitationCode, error) {
	now := time.Now().UTC()
	s.disabled = &service.ReusableInvitationCode{
		ID: id, Code: "FOREVER", Status: service.ReusableInvitationCodeStatusDisabled,
		CreatedAt: now, UpdatedAt: now,
	}
	return s.disabled, nil
}
func (*reusableInvitationCodeAdminRepoStub) Use(context.Context, int64, int64, string, string) error {
	panic("unexpected Use call")
}
func (*reusableInvitationCodeAdminRepoStub) Release(context.Context, int64, int64) error {
	panic("unexpected Release call")
}
func (s *reusableInvitationCodeAdminRepoStub) ListUsesByCodeID(context.Context, int64, int) ([]service.ReusableInvitationCodeUse, error) {
	return s.uses, nil
}

func TestReusableInvitationCodeHandlerCreateListDisableAndUses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	repo := &reusableInvitationCodeAdminRepoStub{
		list: []service.ReusableInvitationCode{{
			ID: 7, Code: "FOREVER", Status: service.ReusableInvitationCodeStatusActive,
			UsedCount: 2, CreatedAt: now, UpdatedAt: now,
		}},
		uses: []service.ReusableInvitationCodeUse{{
			ID: 11, CodeID: 7, UserID: 42, Email: "user@example.com",
			AuthSource: "email", UsedAt: now,
		}},
	}
	handler := NewReusableInvitationCodeHandler(repo)
	router := gin.New()
	router.POST("/codes", handler.Create)
	router.GET("/codes", handler.List)
	router.POST("/codes/:id/disable", handler.Disable)
	router.GET("/codes/:id/uses", handler.ListUses)

	createReq := httptest.NewRequest(http.MethodPost, "/codes", bytes.NewBufferString(`{"code":"  FOREVER  ","notes":" permanent "}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)
	require.NotNil(t, repo.created)
	require.Equal(t, "FOREVER", repo.created.Code)
	require.Equal(t, "permanent", repo.created.Notes)
	require.Zero(t, repo.created.MaxUses)
	require.Nil(t, repo.created.ExpiresAt)

	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/codes", nil))
	require.Equal(t, http.StatusOK, listRec.Code)
	var listBody struct {
		Data struct {
			Items []ReusableInvitationCodeResponse `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listBody))
	require.Len(t, listBody.Data.Items, 1)
	require.Equal(t, "FOREVER", listBody.Data.Items[0].Code)

	disableRec := httptest.NewRecorder()
	router.ServeHTTP(disableRec, httptest.NewRequest(http.MethodPost, "/codes/7/disable", nil))
	require.Equal(t, http.StatusOK, disableRec.Code)
	require.Equal(t, service.ReusableInvitationCodeStatusDisabled, repo.disabled.Status)

	usesRec := httptest.NewRecorder()
	router.ServeHTTP(usesRec, httptest.NewRequest(http.MethodGet, "/codes/7/uses", nil))
	require.Equal(t, http.StatusOK, usesRec.Code)
	var usesBody struct {
		Data []ReusableInvitationCodeUseResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(usesRec.Body.Bytes(), &usesBody))
	require.Len(t, usesBody.Data, 1)
	require.Equal(t, int64(42), usesBody.Data[0].UserID)
}

func TestReusableInvitationCodeHandlerRejectsPastExpiry(t *testing.T) {
	repo := &reusableInvitationCodeAdminRepoStub{}
	handler := NewReusableInvitationCodeHandler(repo)
	router := gin.New()
	router.POST("/codes", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/codes", bytes.NewBufferString(`{"code":"PAST","expires_at":"2020-01-01T00:00:00Z"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Nil(t, repo.created)
}

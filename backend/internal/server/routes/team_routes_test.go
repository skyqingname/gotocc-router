package routes

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/handler"
	adminhandler "github.com/LuckyKuang/sub2api-plus/internal/handler/admin"
	servermiddleware "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserTeamRoutesExposeCompleteContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Team: &handler.TeamHandler{}}
	registerUserTeamRoutes(
		router.Group("/api/v1"),
		handlers,
		servermiddleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() }),
		nil,
	)

	require.Equal(t, []string{
		"DELETE /api/v1/team",
		"DELETE /api/v1/team/invitations/:id",
		"DELETE /api/v1/team/keys/:id",
		"DELETE /api/v1/team/members/:user_id",
		"DELETE /api/v1/team/ownership-transfer",
		"GET /api/v1/team",
		"GET /api/v1/team/invitations",
		"GET /api/v1/team/keys",
		"GET /api/v1/team/members",
		"GET /api/v1/team/usage",
		"GET /api/v1/team/usage/logs",
		"GET /api/v1/team/usage/members",
		"PATCH /api/v1/team",
		"PATCH /api/v1/team/default-member-limits",
		"PATCH /api/v1/team/members/:user_id/limits",
		"POST /api/v1/team",
		"POST /api/v1/team/invitations",
		"POST /api/v1/team/invitations/:id/reissue",
		"POST /api/v1/team/invitations/preview",
		"POST /api/v1/team/invitations/resolve",
		"POST /api/v1/team/keys/:id/disable",
		"POST /api/v1/team/keys/:id/enable",
		"POST /api/v1/team/leave",
		"POST /api/v1/team/members/:user_id/usage/reset",
		"POST /api/v1/team/ownership-transfer",
		"POST /api/v1/team/ownership-transfer/resolve",
		"POST /api/v1/team/status",
	}, registeredRoutes(router))
}

func TestTeamSensitiveRoutesRequireStepUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stepUp := servermiddleware.StepUpAuthMiddleware(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusPreconditionRequired)
	})

	userRouter := gin.New()
	registerUserTeamRoutes(userRouter.Group("/api/v1"), &handler.Handlers{Team: &handler.TeamHandler{}}, stepUp, nil)
	for _, request := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/team/status"},
		{http.MethodDelete, "/api/v1/team"},
		{http.MethodPost, "/api/v1/team/ownership-transfer"},
		{http.MethodPost, "/api/v1/team/ownership-transfer/resolve"},
		{http.MethodDelete, "/api/v1/team/ownership-transfer"},
	} {
		recorder := httptest.NewRecorder()
		userRouter.ServeHTTP(recorder, httptest.NewRequest(request.method, request.path, nil))
		require.Equal(t, http.StatusPreconditionRequired, recorder.Code, "%s %s", request.method, request.path)
	}

	adminRouter := gin.New()
	registerTeamRoutes(adminRouter.Group("/api/v1/admin"), &handler.Handlers{Admin: &handler.AdminHandlers{Team: &adminhandler.TeamHandler{}}}, stepUp)
	for _, request := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/admin/teams/1/force-transfer"},
		{http.MethodDelete, "/api/v1/admin/teams/1"},
	} {
		recorder := httptest.NewRecorder()
		adminRouter.ServeHTTP(recorder, httptest.NewRequest(request.method, request.path, nil))
		require.Equal(t, http.StatusPreconditionRequired, recorder.Code, "%s %s", request.method, request.path)
	}
}

func TestAdminTeamRoutesExposeCompleteContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerTeamRoutes(
		router.Group("/api/v1/admin"),
		&handler.Handlers{Admin: &handler.AdminHandlers{Team: &adminhandler.TeamHandler{}}},
		servermiddleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() }),
	)

	require.Equal(t, []string{
		"DELETE /api/v1/admin/teams/:id",
		"GET /api/v1/admin/teams",
		"GET /api/v1/admin/teams/:id",
		"GET /api/v1/admin/teams/:id/members",
		"GET /api/v1/admin/teams/:id/usage",
		"PATCH /api/v1/admin/teams/:id",
		"POST /api/v1/admin/teams",
		"POST /api/v1/admin/teams/:id/force-transfer",
	}, registeredRoutes(router))
}

func registeredRoutes(router *gin.Engine) []string {
	routes := router.Routes()
	items := make([]string, 0, len(routes))
	for _, route := range routes {
		items = append(items, route.Method+" "+route.Path)
	}
	sort.Strings(items)
	return items
}

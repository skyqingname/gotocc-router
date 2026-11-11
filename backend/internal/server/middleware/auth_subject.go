package middleware

import (
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthSubject is the minimal authenticated identity stored in gin context.
// UserID remains the billing/authorization identity; TeamID scopes request capacity.
type AuthSubject struct {
	UserID      int64
	Concurrency int
	TeamID      *int64
}

func (s AuthSubject) RequestLimitScopeID() int64 {
	return service.RequestLimitScopeID(s.UserID, s.TeamID)
}

func GetAuthSubjectFromContext(c *gin.Context) (AuthSubject, bool) {
	value, exists := c.Get(string(ContextKeyUser))
	if !exists {
		return AuthSubject{}, false
	}
	subject, ok := value.(AuthSubject)
	return subject, ok
}

func GetUserRoleFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(string(ContextKeyUserRole))
	if !exists {
		return "", false
	}
	role, ok := value.(string)
	return role, ok
}

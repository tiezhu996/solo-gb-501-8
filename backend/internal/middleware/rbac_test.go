package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"sterile-packaging-release-control/backend/internal/constants"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/util"
)

type rbacUserRepository struct{ user *model.User }

func (r rbacUserRepository) FindByUsername(context.Context, string) (*model.User, error) {
	return r.user, nil
}
func (r rbacUserRepository) Find(context.Context, uint) (*model.User, error) { return r.user, nil }
func (r rbacUserRepository) Create(context.Context, *model.User) error       { return nil }
func (r rbacUserRepository) Count(context.Context) (int64, error)            { return 1, nil }

// 模拟 POST /api/inspections/:id/complete 的权限保护：携带真实 JWT 的用户必须
// 具备 inspection:write 权限，否则 403。
func inspectionWriteResponse(t *testing.T, role constants.Role) *httptest.ResponseRecorder {
	t.Helper()
	const secret = "test-secret-that-is-long-enough"
	user := &model.User{Base: model.Base{ID: 5}, Username: "tester", DisplayName: "测试员", Role: role, Active: true}
	token, _, err := util.SignToken(secret, time.Hour, user.ID, user.Username, user.DisplayName, role)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(RequestContext(), Auth(secret, rbacUserRepository{user: user}), RequirePermission("inspection:write"))
	r.POST("/inspections/:id/complete", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodPost, "/inspections/1/complete", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	return response
}

func TestInspectionWriteRequiresPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []constants.Role{constants.RoleOperator, constants.RoleViewer, constants.RoleApprover} {
		if response := inspectionWriteResponse(t, role); response.Code != http.StatusForbidden {
			t.Errorf("role %s: got %d, want 403", role, response.Code)
		}
	}
	for _, role := range []constants.Role{constants.RoleInspector, constants.RoleAdmin} {
		if response := inspectionWriteResponse(t, role); response.Code != http.StatusNoContent {
			t.Errorf("role %s: got %d, want 204", role, response.Code)
		}
	}
}

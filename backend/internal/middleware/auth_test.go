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

type authUserRepository struct{ user *model.User }

func (r authUserRepository) FindByUsername(context.Context, string) (*model.User, error) {
	return r.user, nil
}
func (r authUserRepository) Find(context.Context, uint) (*model.User, error) { return r.user, nil }
func (r authUserRepository) Create(context.Context, *model.User) error       { return nil }
func (r authUserRepository) Count(context.Context) (int64, error)            { return 1, nil }

func signedRequest(t *testing.T, role constants.Role, user *model.User) *httptest.ResponseRecorder {
	t.Helper()
	const secret = "test-secret-that-is-long-enough"
	token, _, err := util.SignToken(secret, time.Hour, user.ID, user.Username, user.DisplayName, role)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(RequestContext(), Auth(secret, authUserRepository{user: user}), RequirePermission("release:write"))
	r.GET("/secured", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/secured", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	return response
}

func TestAuthUsesCurrentDatabaseRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := &model.User{Base: model.Base{ID: 7}, Username: "reviewer", DisplayName: "复核员", Role: constants.RoleViewer, Active: true}
	response := signedRequest(t, constants.RoleApprover, user)
	if response.Code != http.StatusForbidden {
		t.Fatalf("stale privileged token returned %d, want 403", response.Code)
	}
}

func TestAuthRejectsDisabledUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := &model.User{Base: model.Base{ID: 7}, Username: "reviewer", DisplayName: "复核员", Role: constants.RoleApprover, Active: false}
	response := signedRequest(t, constants.RoleApprover, user)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("disabled user returned %d, want 401", response.Code)
	}
}

func TestRequestContextReplacesUnsafeRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestContext())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "invalid request id")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	if got := response.Header().Get("X-Request-ID"); got == "" || got == req.Header.Get("X-Request-ID") {
		t.Fatalf("unsafe request ID was not replaced: %q", got)
	}
}

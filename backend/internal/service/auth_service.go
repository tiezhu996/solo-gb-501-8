package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/constants"
	"sterile-packaging-release-control/backend/internal/dto"
	"sterile-packaging-release-control/backend/internal/model"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/util"
)

type AuthService interface {
	Login(context.Context, dto.LoginRequest) (*dto.LoginResponse, error)
	CurrentUser(context.Context, uint) (*dto.UserView, error)
	Seed(context.Context) error
}

type authService struct {
	repo   repository.UserRepository
	secret string
	ttl    time.Duration
}

func NewAuthService(repo repository.UserRepository, secret string, ttl time.Duration) AuthService {
	return &authService{repo: repo, secret: secret, ttl: ttl}
}

func (s *authService) Login(ctx context.Context, input dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.repo.FindByUsername(ctx, input.Username)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !util.ComparePassword(user.PasswordHash, input.Password)) {
		return nil, &util.APIError{Status: 401, Code: "INVALID_CREDENTIALS", Message: "用户名或密码错误"}
	}
	if err != nil {
		return nil, err
	}
	if !user.Active {
		return nil, &util.APIError{Status: 403, Code: "USER_DISABLED", Message: "用户已停用"}
	}
	token, expires, err := util.SignToken(s.secret, s.ttl, user.ID, user.Username, user.DisplayName, user.Role)
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}
	return &dto.LoginResponse{Token: token, ExpiresAt: expires.Unix(), User: userView(user)}, nil
}

func (s *authService) CurrentUser(ctx context.Context, id uint) (*dto.UserView, error) {
	user, err := s.repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	view := userView(user)
	return &view, nil
}

func (s *authService) Seed(ctx context.Context) error {
	count, err := s.repo.Count(ctx)
	if err != nil || count > 0 {
		return err
	}
	seeds := []struct {
		username, name, password string
		role                     constants.Role
	}{
		{"admin", "质量管理员", "admin123", constants.RoleAdmin},
		{"inspector", "检验员", "inspect123", constants.RoleInspector},
		{"approver", "放行审批员", "approve123", constants.RoleApprover},
		{"operator", "产线操作员", "operate123", constants.RoleOperator},
	}
	for _, seed := range seeds {
		hash, hashErr := util.HashPassword(seed.password)
		if hashErr != nil {
			return hashErr
		}
		if createErr := s.repo.Create(ctx, &model.User{Username: seed.username, DisplayName: seed.name, PasswordHash: hash, Role: seed.role, Active: true}); createErr != nil {
			return createErr
		}
	}
	return nil
}

func userView(user *model.User) dto.UserView {
	all := []string{"line:write", "batch:write", "inspection:write", "release:write", "audit:read"}
	permissions := make([]string, 0, len(all))
	for _, permission := range all {
		if user.Role.Can(permission) {
			permissions = append(permissions, permission)
		}
	}
	return dto.UserView{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role, Permissions: permissions}
}

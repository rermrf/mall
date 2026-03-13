package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/user/domain"
	"github.com/rermrf/mall/user/events"
	"github.com/rermrf/mall/user/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateUser        = errors.New("用户已存在")
	ErrUserNotFound         = errors.New("用户不存在")
	ErrInvalidCredentials   = errors.New("用户名或密码错误")
	ErrSmsCodeVerifyFailed  = errors.New("验证码错误或已过期")
	ErrUserFrozen           = errors.New("用户已被冻结")
)

type UserService interface {
	// 认证
	Signup(ctx context.Context, u domain.User) (domain.User, error)
	Login(ctx context.Context, tenantId int64, phone, password string) (domain.User, error)
	SendSmsCode(ctx context.Context, tenantId int64, phone string) error
	LoginByPhone(ctx context.Context, tenantId int64, phone, code string) (domain.User, error)
	OAuthLogin(ctx context.Context, tenantId int64, provider, code string) (domain.User, bool, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, accessToken string) error

	// 用户管理
	FindById(ctx context.Context, id int64) (domain.User, error)
	FindByPhone(ctx context.Context, tenantId int64, phone string) (domain.User, error)
	ListUsers(ctx context.Context, tenantId int64, page, pageSize int32, status int32, keyword string) ([]domain.User, int64, error)
	UpdateProfile(ctx context.Context, id int64, nickname, avatar string) error
	UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error

	// RBAC
	GetPermissions(ctx context.Context, uid, tenantId int64) ([]domain.Permission, error)
	AssignRole(ctx context.Context, userId, tenantId, roleId int64) error
	CreateRole(ctx context.Context, role domain.Role) (domain.Role, error)
	UpdateRole(ctx context.Context, role domain.Role) error
	ListRoles(ctx context.Context, tenantId int64) ([]domain.Role, error)

	// 地址
	CreateAddress(ctx context.Context, a domain.UserAddress) (domain.UserAddress, error)
	ListAddresses(ctx context.Context, userId int64) ([]domain.UserAddress, error)
	UpdateAddress(ctx context.Context, a domain.UserAddress) error
	DeleteAddress(ctx context.Context, id, userId int64) error
}

type userService struct {
	repo     repository.UserRepository
	producer events.Producer
	l        logger.Logger
}

func NewUserService(repo repository.UserRepository, producer events.Producer, l logger.Logger) UserService {
	return &userService{
		repo:     repo,
		producer: producer,
		l:        l,
	}
}

// ==================== 认证 ====================

func (s *userService) Signup(ctx context.Context, u domain.User) (domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	u.Password = string(hash)
	u.Status = domain.UserStatusNormal

	created, err := s.repo.Create(ctx, u)
	if err != nil {
		return domain.User{}, err
	}

	// 异步发送注册事件
	go func() {
		er := s.producer.ProduceUserRegistered(context.Background(), events.UserRegisteredEvent{
			UserId:   created.ID,
			TenantId: created.TenantID,
			Phone:    created.Phone,
		})
		if er != nil {
			s.l.Error("发送用户注册事件失败",
				logger.Error(er),
				logger.Int64("uid", created.ID),
			)
		}
	}()

	return created, nil
}

func (s *userService) Login(ctx context.Context, tenantId int64, phone, password string) (domain.User, error) {
	u, err := s.repo.FindByTenantAndPhone(ctx, tenantId, phone)
	if err != nil {
		return domain.User{}, ErrInvalidCredentials
	}
	if u.Status == domain.UserStatusFrozen {
		return domain.User{}, ErrUserFrozen
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return domain.User{}, ErrInvalidCredentials
	}
	return u, nil
}

func (s *userService) SendSmsCode(ctx context.Context, tenantId int64, phone string) error {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	return s.repo.SendSmsCode(ctx, tenantId, phone, code)
}

func (s *userService) LoginByPhone(ctx context.Context, tenantId int64, phone, code string) (domain.User, error) {
	ok, err := s.repo.VerifySmsCode(ctx, tenantId, phone, code)
	if err != nil || !ok {
		return domain.User{}, ErrSmsCodeVerifyFailed
	}
	u, err := s.repo.FindByTenantAndPhone(ctx, tenantId, phone)
	if err != nil {
		// 手机号登录，如果用户不存在则自动注册
		u = domain.User{
			TenantID: tenantId,
			Phone:    phone,
			Status:   domain.UserStatusNormal,
		}
		u, err = s.repo.Create(ctx, u)
		if err != nil {
			return domain.User{}, err
		}
	}
	if u.Status == domain.UserStatusFrozen {
		return domain.User{}, ErrUserFrozen
	}
	return u, nil
}

func (s *userService) OAuthLogin(ctx context.Context, tenantId int64, provider, code string) (domain.User, bool, error) {
	// MVP: 简化 OAuth 实现，用 code 作为 provider_uid
	// 生产环境需对接真实 OAuth provider（Google/GitHub）获取用户信息
	oauth := domain.OAuthAccount{
		TenantID:    tenantId,
		Provider:    provider,
		ProviderUID: code,
	}
	newUser := domain.User{
		TenantID: tenantId,
		Phone:    "",
		Status:   domain.UserStatusNormal,
	}
	u, err := s.repo.FindOrCreateByOAuth(ctx, oauth, newUser)
	if err != nil {
		return domain.User{}, false, err
	}
	isNew := u.Phone == ""
	return u, isNew, nil
}

func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// JWT 刷新由 BFF 层本地处理，user-svc 不参与 token 管理
	return "", "", nil
}

func (s *userService) Logout(ctx context.Context, accessToken string) error {
	// JWT 黑名单由 BFF 层 Redis 管理，user-svc 不参与 token 管理
	return nil
}

// ==================== 用户管理 ====================

func (s *userService) FindById(ctx context.Context, id int64) (domain.User, error) {
	return s.repo.FindById(ctx, id)
}

func (s *userService) FindByPhone(ctx context.Context, tenantId int64, phone string) (domain.User, error) {
	return s.repo.FindByTenantAndPhone(ctx, tenantId, phone)
}

func (s *userService) ListUsers(ctx context.Context, tenantId int64, page, pageSize int32, status int32, keyword string) ([]domain.User, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := int((page - 1) * pageSize)
	return s.repo.ListByTenant(ctx, tenantId, offset, int(pageSize), uint8(status), keyword)
}

func (s *userService) UpdateProfile(ctx context.Context, id int64, nickname, avatar string) error {
	return s.repo.Update(ctx, domain.User{
		ID:       id,
		Nickname: nickname,
		Avatar:   avatar,
	})
}

func (s *userService) UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

// ==================== RBAC ====================

func (s *userService) GetPermissions(ctx context.Context, uid, tenantId int64) ([]domain.Permission, error) {
	return s.repo.GetPermissions(ctx, uid, tenantId)
}

func (s *userService) AssignRole(ctx context.Context, userId, tenantId, roleId int64) error {
	return s.repo.AssignRole(ctx, userId, tenantId, roleId)
}

func (s *userService) CreateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	return s.repo.CreateRole(ctx, role)
}

func (s *userService) UpdateRole(ctx context.Context, role domain.Role) error {
	return s.repo.UpdateRole(ctx, role)
}

func (s *userService) ListRoles(ctx context.Context, tenantId int64) ([]domain.Role, error) {
	return s.repo.ListRoles(ctx, tenantId)
}

// ==================== 地址 ====================

func (s *userService) CreateAddress(ctx context.Context, a domain.UserAddress) (domain.UserAddress, error) {
	return s.repo.CreateAddress(ctx, a)
}

func (s *userService) ListAddresses(ctx context.Context, userId int64) ([]domain.UserAddress, error) {
	return s.repo.ListAddresses(ctx, userId)
}

func (s *userService) UpdateAddress(ctx context.Context, a domain.UserAddress) error {
	return s.repo.UpdateAddress(ctx, a)
}

func (s *userService) DeleteAddress(ctx context.Context, id, userId int64) error {
	return s.repo.DeleteAddress(ctx, id, userId)
}

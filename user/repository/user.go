package repository

import (
	"context"
	"time"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/user/domain"
	"github.com/rermrf/mall/user/repository/cache"
	"github.com/rermrf/mall/user/repository/dao"
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	FindByTenantAndPhone(ctx context.Context, tenantId int64, phone string) (domain.User, error)
	FindById(ctx context.Context, id int64) (domain.User, error)
	Update(ctx context.Context, u domain.User) error
	UpdateStatus(ctx context.Context, id int64, status domain.UserStatus) error
	FindOrCreateByOAuth(ctx context.Context, oauth domain.OAuthAccount, u domain.User) (domain.User, error)
	ListByTenant(ctx context.Context, tenantId int64, offset, limit int, status uint8, keyword string) ([]domain.User, int64, error)

	SendSmsCode(ctx context.Context, tenantId int64, phone, code string) error
	VerifySmsCode(ctx context.Context, tenantId int64, phone, code string) (bool, error)

	GetPermissions(ctx context.Context, uid, tenantId int64) ([]domain.Permission, error)
	AssignRole(ctx context.Context, userId, tenantId, roleId int64) error
	CreateRole(ctx context.Context, r domain.Role) (domain.Role, error)
	UpdateRole(ctx context.Context, r domain.Role) error
	ListRoles(ctx context.Context, tenantId int64) ([]domain.Role, error)

	CreateAddress(ctx context.Context, a domain.UserAddress) (domain.UserAddress, error)
	ListAddresses(ctx context.Context, userId int64) ([]domain.UserAddress, error)
	UpdateAddress(ctx context.Context, a domain.UserAddress) error
	DeleteAddress(ctx context.Context, id, userId int64) error
}

type CachedUserRepository struct {
	userDAO dao.UserDAO
	roleDAO dao.RoleDAO
	addrDAO dao.AddressDAO
	cache   cache.UserCache
	l       logger.Logger
}

func NewUserRepository(
	userDAO dao.UserDAO,
	roleDAO dao.RoleDAO,
	addrDAO dao.AddressDAO,
	cache cache.UserCache,
	l logger.Logger,
) UserRepository {
	return &CachedUserRepository{
		userDAO: userDAO,
		roleDAO: roleDAO,
		addrDAO: addrDAO,
		cache:   cache,
		l:       l,
	}
}

// ==================== User CRUD ====================

func (r *CachedUserRepository) Create(ctx context.Context, u domain.User) (domain.User, error) {
	entity := r.domainToEntity(u)
	res, err := r.userDAO.Insert(ctx, entity)
	if err != nil {
		return domain.User{}, err
	}
	return r.entityToDomain(res), nil
}

func (r *CachedUserRepository) FindByTenantAndPhone(ctx context.Context, tenantId int64, phone string) (domain.User, error) {
	entity, err := r.userDAO.FindByTenantAndPhone(ctx, tenantId, phone)
	if err != nil {
		return domain.User{}, err
	}
	return r.entityToDomain(entity), nil
}

func (r *CachedUserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
	// Cache-Aside: try cache first
	u, err := r.cache.Get(ctx, id)
	if err == nil {
		return u, nil
	}
	entity, err := r.userDAO.FindById(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	u = r.entityToDomain(entity)
	// async set cache, don't block
	go func() {
		if er := r.cache.Set(context.Background(), u); er != nil {
			r.l.Error("设置用户缓存失败", logger.Error(er), logger.Int64("uid", id))
		}
	}()
	return u, nil
}

func (r *CachedUserRepository) Update(ctx context.Context, u domain.User) error {
	entity := r.domainToEntity(u)
	err := r.userDAO.UpdateNonZeroFields(ctx, entity)
	if err != nil {
		return err
	}
	return r.cache.Delete(ctx, u.ID)
}

func (r *CachedUserRepository) UpdateStatus(ctx context.Context, id int64, status domain.UserStatus) error {
	err := r.userDAO.UpdateStatus(ctx, id, uint8(status))
	if err != nil {
		return err
	}
	return r.cache.Delete(ctx, id)
}

func (r *CachedUserRepository) FindOrCreateByOAuth(ctx context.Context, oauth domain.OAuthAccount, u domain.User) (domain.User, error) {
	oaEntity := dao.OAuthAccount{
		TenantId:    oauth.TenantID,
		Provider:    oauth.Provider,
		ProviderUid: oauth.ProviderUID,
		AccessToken: oauth.AccessToken,
	}
	uEntity := r.domainToEntity(u)
	res, err := r.userDAO.FindOrCreateByOAuth(ctx, oaEntity, uEntity)
	if err != nil {
		return domain.User{}, err
	}
	return r.entityToDomain(res), nil
}

func (r *CachedUserRepository) ListByTenant(ctx context.Context, tenantId int64, offset, limit int, status uint8, keyword string) ([]domain.User, int64, error) {
	entities, total, err := r.userDAO.ListByTenant(ctx, tenantId, offset, limit, status, keyword)
	if err != nil {
		return nil, 0, err
	}
	users := make([]domain.User, 0, len(entities))
	for _, e := range entities {
		users = append(users, r.entityToDomain(e))
	}
	return users, total, nil
}

// ==================== SMS Code ====================

func (r *CachedUserRepository) SendSmsCode(ctx context.Context, tenantId int64, phone, code string) error {
	return r.cache.SetSmsCode(ctx, tenantId, phone, code)
}

func (r *CachedUserRepository) VerifySmsCode(ctx context.Context, tenantId int64, phone, code string) (bool, error) {
	saved, err := r.cache.GetSmsCode(ctx, tenantId, phone)
	if err != nil {
		return false, err
	}
	return saved == code, nil
}

// ==================== RBAC ====================

func (r *CachedUserRepository) GetPermissions(ctx context.Context, uid, tenantId int64) ([]domain.Permission, error) {
	// Cache-Aside
	perms, err := r.cache.GetPermissions(ctx, uid, tenantId)
	if err == nil {
		return perms, nil
	}
	daoPerms, err := r.roleDAO.GetPermissionsByUserAndTenant(ctx, uid, tenantId)
	if err != nil {
		return nil, err
	}
	perms = make([]domain.Permission, 0, len(daoPerms))
	for _, p := range daoPerms {
		perms = append(perms, domain.Permission{
			ID:       p.ID,
			Code:     p.Code,
			Name:     p.Name,
			Type:     domain.PermissionType(p.Type),
			Resource: p.Resource,
			Ctime:    p.Ctime,
			Utime:    p.Utime,
		})
	}
	go func() {
		if er := r.cache.SetPermissions(context.Background(), uid, tenantId, perms); er != nil {
			r.l.Error("设置权限缓存失败", logger.Error(er))
		}
	}()
	return perms, nil
}

func (r *CachedUserRepository) AssignRole(ctx context.Context, userId, tenantId, roleId int64) error {
	return r.roleDAO.InsertUserRole(ctx, dao.UserRole{
		UserId:   userId,
		TenantId: tenantId,
		RoleId:   roleId,
	})
}

func (r *CachedUserRepository) CreateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	entity, err := r.roleDAO.InsertRole(ctx, dao.Role{
		TenantId:    role.TenantID,
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
	})
	if err != nil {
		return domain.Role{}, err
	}
	role.ID = entity.ID
	role.Ctime = entity.Ctime
	role.Utime = entity.Utime
	return role, nil
}

func (r *CachedUserRepository) UpdateRole(ctx context.Context, role domain.Role) error {
	return r.roleDAO.UpdateRole(ctx, dao.Role{
		ID:          role.ID,
		TenantId:    role.TenantID,
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
	})
}

func (r *CachedUserRepository) ListRoles(ctx context.Context, tenantId int64) ([]domain.Role, error) {
	entities, err := r.roleDAO.ListByTenant(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	roles := make([]domain.Role, 0, len(entities))
	for _, e := range entities {
		roles = append(roles, domain.Role{
			ID:          e.ID,
			TenantID:    e.TenantId,
			Name:        e.Name,
			Code:        e.Code,
			Description: e.Description,
			Ctime:       e.Ctime,
			Utime:       e.Utime,
		})
	}
	return roles, nil
}

// ==================== Address ====================

func (r *CachedUserRepository) CreateAddress(ctx context.Context, a domain.UserAddress) (domain.UserAddress, error) {
	entity, err := r.addrDAO.Insert(ctx, dao.UserAddress{
		UserId:    a.UserID,
		Name:      a.Name,
		Phone:     a.Phone,
		Province:  a.Province,
		City:      a.City,
		District:  a.District,
		Detail:    a.Detail,
		IsDefault: a.IsDefault,
	})
	if err != nil {
		return domain.UserAddress{}, err
	}
	a.Id = entity.ID
	return a, nil
}

func (r *CachedUserRepository) ListAddresses(ctx context.Context, userId int64) ([]domain.UserAddress, error) {
	entities, err := r.addrDAO.ListByUser(ctx, userId)
	if err != nil {
		return nil, err
	}
	addrs := make([]domain.UserAddress, 0, len(entities))
	for _, e := range entities {
		addrs = append(addrs, domain.UserAddress{
			Id:        e.ID,
			UserID:    e.UserId,
			Name:      e.Name,
			Phone:     e.Phone,
			Province:  e.Province,
			City:      e.City,
			District:  e.District,
			Detail:    e.Detail,
			IsDefault: e.IsDefault,
		})
	}
	return addrs, nil
}

func (r *CachedUserRepository) UpdateAddress(ctx context.Context, a domain.UserAddress) error {
	return r.addrDAO.Update(ctx, dao.UserAddress{
		ID:        a.Id,
		UserId:    a.UserID,
		Name:      a.Name,
		Phone:     a.Phone,
		Province:  a.Province,
		City:      a.City,
		District:  a.District,
		Detail:    a.Detail,
		IsDefault: a.IsDefault,
	})
}

func (r *CachedUserRepository) DeleteAddress(ctx context.Context, id, userId int64) error {
	return r.addrDAO.Delete(ctx, id, userId)
}

// ==================== Converters ====================

func (r *CachedUserRepository) entityToDomain(e dao.User) domain.User {
	return domain.User{
		ID:       e.ID,
		TenantID: e.TenantId,
		Phone:    e.Phone,
		Email:    e.Email,
		Password: e.Password,
		Nickname: e.Nickname,
		Avatar:   e.Avatar,
		Status:   domain.UserStatus(e.Status),
		Ctime:    time.UnixMilli(e.Ctime),
		Utime:    time.UnixMilli(e.Utime),
	}
}

func (r *CachedUserRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{
		ID:       u.ID,
		TenantId: u.TenantID,
		Phone:    u.Phone,
		Email:    u.Email,
		Password: u.Password,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
		Status:   uint8(u.Status),
	}
}

package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/user/domain"
	"github.com/rermrf/mall/user/service"
)

type UserGRPCServer struct {
	userv1.UnimplementedUserServiceServer
	svc service.UserService
}

func NewUserGRPCServer(svc service.UserService) *UserGRPCServer {
	return &UserGRPCServer{svc: svc}
}

func (s *UserGRPCServer) Register(server *grpc.Server) {
	userv1.RegisterUserServiceServer(server, s)
}

// ==================== 认证 ====================

func (s *UserGRPCServer) Signup(ctx context.Context, req *userv1.SignupRequest) (*userv1.SignupResponse, error) {
	u, err := s.svc.Signup(ctx, domain.User{
		TenantID: req.GetTenantId(),
		Phone:    req.GetPhone(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "注册失败: %v", err)
	}
	return &userv1.SignupResponse{Id: u.ID}, nil
}

func (s *UserGRPCServer) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	u, err := s.svc.Login(ctx, req.GetTenantId(), req.GetPhone(), req.GetPassword())
	if err != nil {
		return nil, handleErr(err)
	}
	return &userv1.LoginResponse{User: s.toDTO(u)}, nil
}

func (s *UserGRPCServer) SendSmsCode(ctx context.Context, req *userv1.SendSmsCodeRequest) (*userv1.SendSmsCodeResponse, error) {
	err := s.svc.SendSmsCode(ctx, req.GetTenantId(), req.GetPhone())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "发送验证码失败: %v", err)
	}
	return &userv1.SendSmsCodeResponse{}, nil
}

func (s *UserGRPCServer) LoginByPhone(ctx context.Context, req *userv1.LoginByPhoneRequest) (*userv1.LoginByPhoneResponse, error) {
	u, err := s.svc.LoginByPhone(ctx, req.GetTenantId(), req.GetPhone(), req.GetCode())
	if err != nil {
		return nil, handleErr(err)
	}
	return &userv1.LoginByPhoneResponse{User: s.toDTO(u)}, nil
}

func (s *UserGRPCServer) OAuthLogin(ctx context.Context, req *userv1.OAuthLoginRequest) (*userv1.OAuthLoginResponse, error) {
	u, isNew, err := s.svc.OAuthLogin(ctx, req.GetTenantId(), req.GetProvider(), req.GetCode())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "OAuth 登录失败: %v", err)
	}
	return &userv1.OAuthLoginResponse{User: s.toDTO(u), IsNew: isNew}, nil
}

func (s *UserGRPCServer) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	accessToken, refreshToken, err := s.svc.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "刷新 token 失败: %v", err)
	}
	return &userv1.RefreshTokenResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *UserGRPCServer) Logout(ctx context.Context, req *userv1.LogoutRequest) (*userv1.LogoutResponse, error) {
	err := s.svc.Logout(ctx, req.GetAccessToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "登出失败: %v", err)
	}
	return &userv1.LogoutResponse{}, nil
}

// ==================== 用户管理 ====================

func (s *UserGRPCServer) FindById(ctx context.Context, req *userv1.FindByIdRequest) (*userv1.FindByIdResponse, error) {
	u, err := s.svc.FindById(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "用户不存在: %v", err)
	}
	return &userv1.FindByIdResponse{User: s.toDTO(u)}, nil
}

func (s *UserGRPCServer) FindByPhone(ctx context.Context, req *userv1.FindByPhoneRequest) (*userv1.FindByPhoneResponse, error) {
	u, err := s.svc.FindByPhone(ctx, req.GetTenantId(), req.GetPhone())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "用户不存在: %v", err)
	}
	return &userv1.FindByPhoneResponse{User: s.toDTO(u)}, nil
}

func (s *UserGRPCServer) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	users, total, err := s.svc.ListUsers(ctx, req.GetTenantId(), req.GetPage(), req.GetPageSize(), req.GetStatus(), req.GetKeyword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询用户列表失败: %v", err)
	}
	dtos := make([]*userv1.User, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, s.toDTO(u))
	}
	return &userv1.ListUsersResponse{Users: dtos, Total: total}, nil
}

func (s *UserGRPCServer) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UpdateProfileResponse, error) {
	err := s.svc.UpdateProfile(ctx, req.GetId(), req.GetNickname(), req.GetAvatar())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "更新资料失败: %v", err)
	}
	return &userv1.UpdateProfileResponse{}, nil
}

func (s *UserGRPCServer) UpdateUserStatus(ctx context.Context, req *userv1.UpdateUserStatusRequest) (*userv1.UpdateUserStatusResponse, error) {
	err := s.svc.UpdateUserStatus(ctx, req.GetId(), domain.UserStatus(req.GetStatus()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "更新状态失败: %v", err)
	}
	return &userv1.UpdateUserStatusResponse{}, nil
}

// ==================== RBAC ====================

func (s *UserGRPCServer) GetPermissions(ctx context.Context, req *userv1.GetPermissionsRequest) (*userv1.GetPermissionsResponse, error) {
	perms, err := s.svc.GetPermissions(ctx, req.GetUserId(), req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "获取权限失败: %v", err)
	}
	dtos := make([]*userv1.Permission, 0, len(perms))
	for _, p := range perms {
		dtos = append(dtos, s.toPermissionDTO(p))
	}
	return &userv1.GetPermissionsResponse{Permissions: dtos}, nil
}

func (s *UserGRPCServer) AssignRole(ctx context.Context, req *userv1.AssignRoleRequest) (*userv1.AssignRoleResponse, error) {
	err := s.svc.AssignRole(ctx, req.GetUserId(), req.GetTenantId(), req.GetRoleId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "分配角色失败: %v", err)
	}
	return &userv1.AssignRoleResponse{}, nil
}

func (s *UserGRPCServer) CreateRole(ctx context.Context, req *userv1.CreateRoleRequest) (*userv1.CreateRoleResponse, error) {
	r := req.GetRole()
	role, err := s.svc.CreateRole(ctx, domain.Role{
		TenantID:    r.GetTenantId(),
		Name:        r.GetName(),
		Code:        r.GetCode(),
		Description: r.GetDescription(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建角色失败: %v", err)
	}
	return &userv1.CreateRoleResponse{Id: role.ID}, nil
}

func (s *UserGRPCServer) UpdateRole(ctx context.Context, req *userv1.UpdateRoleRequest) (*userv1.UpdateRoleResponse, error) {
	r := req.GetRole()
	err := s.svc.UpdateRole(ctx, domain.Role{
		ID:          r.GetId(),
		TenantID:    r.GetTenantId(),
		Name:        r.GetName(),
		Code:        r.GetCode(),
		Description: r.GetDescription(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "更新角色失败: %v", err)
	}
	return &userv1.UpdateRoleResponse{}, nil
}

func (s *UserGRPCServer) ListRoles(ctx context.Context, req *userv1.ListRolesRequest) (*userv1.ListRolesResponse, error) {
	roles, err := s.svc.ListRoles(ctx, req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询角色列表失败: %v", err)
	}
	dtos := make([]*userv1.Role, 0, len(roles))
	for _, r := range roles {
		dtos = append(dtos, s.toRoleDTO(r))
	}
	return &userv1.ListRolesResponse{Roles: dtos}, nil
}

// ==================== 收货地址 ====================

func (s *UserGRPCServer) CreateAddress(ctx context.Context, req *userv1.CreateAddressRequest) (*userv1.CreateAddressResponse, error) {
	a := req.GetAddress()
	addr, err := s.svc.CreateAddress(ctx, domain.UserAddress{
		UserID:    a.GetUserId(),
		Name:      a.GetName(),
		Phone:     a.GetPhone(),
		Province:  a.GetProvince(),
		City:      a.GetCity(),
		District:  a.GetDistrict(),
		Detail:    a.GetDetail(),
		IsDefault: a.GetIsDefault(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建地址失败: %v", err)
	}
	return &userv1.CreateAddressResponse{Id: addr.Id}, nil
}

func (s *UserGRPCServer) ListAddresses(ctx context.Context, req *userv1.ListAddressesRequest) (*userv1.ListAddressesResponse, error) {
	addrs, err := s.svc.ListAddresses(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询地址列表失败: %v", err)
	}
	dtos := make([]*userv1.UserAddress, 0, len(addrs))
	for _, a := range addrs {
		dtos = append(dtos, s.toAddressDTO(a))
	}
	return &userv1.ListAddressesResponse{Addresses: dtos}, nil
}

func (s *UserGRPCServer) UpdateAddress(ctx context.Context, req *userv1.UpdateAddressRequest) (*userv1.UpdateAddressResponse, error) {
	a := req.GetAddress()
	err := s.svc.UpdateAddress(ctx, domain.UserAddress{
		Id:        a.GetId(),
		UserID:    a.GetUserId(),
		Name:      a.GetName(),
		Phone:     a.GetPhone(),
		Province:  a.GetProvince(),
		City:      a.GetCity(),
		District:  a.GetDistrict(),
		Detail:    a.GetDetail(),
		IsDefault: a.GetIsDefault(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "更新地址失败: %v", err)
	}
	return &userv1.UpdateAddressResponse{}, nil
}

func (s *UserGRPCServer) DeleteAddress(ctx context.Context, req *userv1.DeleteAddressRequest) (*userv1.DeleteAddressResponse, error) {
	err := s.svc.DeleteAddress(ctx, req.GetId(), req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "删除地址失败: %v", err)
	}
	return &userv1.DeleteAddressResponse{}, nil
}

// ==================== DTO 转换 ====================

func (s *UserGRPCServer) toDTO(u domain.User) *userv1.User {
	return &userv1.User{
		Id:       u.ID,
		TenantId: u.TenantID,
		Phone:    u.Phone,
		Email:    u.Email,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
		Status:   int32(u.Status),
		Ctime:    timestamppb.New(u.Ctime),
		Utime:    timestamppb.New(u.Utime),
	}
}

func (s *UserGRPCServer) toRoleDTO(r domain.Role) *userv1.Role {
	return &userv1.Role{
		Id:          r.ID,
		TenantId:    r.TenantID,
		Name:        r.Name,
		Code:        r.Code,
		Description: r.Description,
	}
}

func (s *UserGRPCServer) toPermissionDTO(p domain.Permission) *userv1.Permission {
	return &userv1.Permission{
		Id:       p.ID,
		Code:     p.Code,
		Name:     p.Name,
		Type:     int32(p.Type),
		Resource: p.Resource,
	}
}

func (s *UserGRPCServer) toAddressDTO(a domain.UserAddress) *userv1.UserAddress {
	return &userv1.UserAddress{
		Id:        a.Id,
		UserId:    a.UserID,
		Name:      a.Name,
		Phone:     a.Phone,
		Province:  a.Province,
		City:      a.City,
		District:  a.District,
		Detail:    a.Detail,
		IsDefault: a.IsDefault,
	}
}

// ==================== 错误处理 ====================

func handleErr(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, service.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrUserFrozen):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, service.ErrSmsCodeVerifyFailed):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Errorf(codes.Internal, "内部错误: %v", err)
	}
}

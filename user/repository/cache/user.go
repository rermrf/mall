package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rermrf/mall/user/domain"
)

var ErrKeyNotExist = redis.Nil

type UserCache interface {
	Get(ctx context.Context, id int64) (domain.User, error)
	Set(ctx context.Context, u domain.User) error
	Delete(ctx context.Context, id int64) error

	GetSmsCode(ctx context.Context, tenantId int64, phone string) (string, error)
	SetSmsCode(ctx context.Context, tenantId int64, phone, code string) error

	GetPermissions(ctx context.Context, uid, tenantId int64) ([]domain.Permission, error)
	SetPermissions(ctx context.Context, uid, tenantId int64, perms []domain.Permission) error
}

type RedisUserCache struct {
	cmd redis.Cmdable
}

func NewUserCache(cmd redis.Cmdable) UserCache {
	return &RedisUserCache{cmd: cmd}
}

func (c *RedisUserCache) userKey(id int64) string {
	return fmt.Sprintf("user:info:%d", id)
}

func (c *RedisUserCache) smsCodeKey(tenantId int64, phone string) string {
	return fmt.Sprintf("user:code:%d:%s", tenantId, phone)
}

func (c *RedisUserCache) smsSentKey(tenantId int64, phone string) string {
	return fmt.Sprintf("user:code:sent:%d:%s", tenantId, phone)
}

func (c *RedisUserCache) permKey(uid, tenantId int64) string {
	return fmt.Sprintf("user:perm:%d:%d", uid, tenantId)
}

// Get 获取用户缓存
func (c *RedisUserCache) Get(ctx context.Context, id int64) (domain.User, error) {
	val, err := c.cmd.Get(ctx, c.userKey(id)).Result()
	if err != nil {
		return domain.User{}, err
	}
	var u domain.User
	err = json.Unmarshal([]byte(val), &u)
	return u, err
}

// Set 设置用户缓存，过期时间 15 分钟
func (c *RedisUserCache) Set(ctx context.Context, u domain.User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return c.cmd.Set(ctx, c.userKey(u.ID), data, 15*time.Minute).Err()
}

// Delete 删除用户缓存
func (c *RedisUserCache) Delete(ctx context.Context, id int64) error {
	return c.cmd.Del(ctx, c.userKey(id)).Err()
}

// SetSmsCode 设置短信验证码（10 分钟过期），同时设置 60s 防重发标记
func (c *RedisUserCache) SetSmsCode(ctx context.Context, tenantId int64, phone, code string) error {
	// 检查 60s 内是否已发送
	sent, _ := c.cmd.Exists(ctx, c.smsSentKey(tenantId, phone)).Result()
	if sent > 0 {
		return fmt.Errorf("验证码发送过于频繁")
	}
	pipe := c.cmd.Pipeline()
	pipe.Set(ctx, c.smsCodeKey(tenantId, phone), code, 10*time.Minute)
	pipe.Set(ctx, c.smsSentKey(tenantId, phone), "1", 60*time.Second)
	_, err := pipe.Exec(ctx)
	return err
}

// GetSmsCode 获取短信验证码
func (c *RedisUserCache) GetSmsCode(ctx context.Context, tenantId int64, phone string) (string, error) {
	return c.cmd.GetDel(ctx, c.smsCodeKey(tenantId, phone)).Result()
}

// GetPermissions 获取权限缓存
func (c *RedisUserCache) GetPermissions(ctx context.Context, uid, tenantId int64) ([]domain.Permission, error) {
	val, err := c.cmd.Get(ctx, c.permKey(uid, tenantId)).Result()
	if err != nil {
		return nil, err
	}
	var perms []domain.Permission
	err = json.Unmarshal([]byte(val), &perms)
	return perms, err
}

// SetPermissions 设置权限缓存，过期时间 10 分钟
func (c *RedisUserCache) SetPermissions(ctx context.Context, uid, tenantId int64, perms []domain.Permission) error {
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	return c.cmd.Set(ctx, c.permKey(uid, tenantId), data, 10*time.Minute).Err()
}

# Handler 错误码分层设计

## 日期
2026-03-13

## 问题
所有 BFF handler 的业务错误统一返回 `(ginx.Result{}, err)`，wrapper 兜底为 `{code: 5, msg: "系统错误"}`。前端无法区分"用户已存在"和"库存不足"等业务错误，只能显示泛化的"系统错误"。

## 解决方案

### 错误码体系

```
0        → 成功
4        → 参数绑定错误 (wrapper 自动处理)
5        → 系统错误 (wrapper 兜底，不暴露内部细节)

401001   → 未登录 / Token 过期
401002   → 用户名或密码错误
401003   → 验证码错误或已过期

403001   → 无权限
403002   → 仅平台管理员可访问
403003   → 需要商家身份

404001   → 用户不存在
404002   → 订单不存在
404003   → 商品不存在
404004   → 地址不存在

409001   → 用户已存在
409002   → 库存不足
409003   → 订单状态不允许此操作
409004   → 优惠券已领取/已过期
409005   → 秒杀已结束/已抢光
409006   → 配额不足

422001   → 用户已被冻结
```

### 实现策略

1. `pkg/ginx/errcode.go` - 定义错误码常量 + `HandleGRPCError` 辅助函数
2. 辅助函数通过 gRPC status message 匹配已知业务错误 → 返回 `(Result{Code, Msg}, nil)`
3. 未匹配的错误 → 返回 `(Result{}, err)` 让 wrapper 兜底为 "系统错误"
4. 修改所有 3 个 BFF 的 handler 使用辅助函数

### HandleGRPCError 设计

```go
type ErrMapping struct {
    Contains string  // gRPC error message 包含此字符串
    Code     int     // 返回给前端的错误码
    Msg      string  // 返回给前端的消息（可选，默认用 Contains）
}

func HandleGRPCError(err error, context string, mappings ...ErrMapping) (Result, error) {
    msg := err.Error()
    for _, m := range mappings {
        if strings.Contains(msg, m.Contains) {
            display := m.Msg
            if display == "" { display = m.Contains }
            return Result{Code: m.Code, Msg: display}, nil
        }
    }
    return Result{}, fmt.Errorf("%s: %w", context, err)
}
```

### Handler 改动模式

```go
// BEFORE:
if err != nil {
    return ginx.Result{}, fmt.Errorf("调用用户服务登录失败: %w", err)
}

// AFTER:
if err != nil {
    return ginx.HandleGRPCError(err, "登录失败",
        ginx.ErrMapping{Contains: "用户名或密码错误", Code: ginx.CodeInvalidCredentials},
        ginx.ErrMapping{Contains: "用户已被冻结", Code: ginx.CodeUserFrozen},
    )
}
```

### 影响范围
- `pkg/ginx/errcode.go` - 新增
- consumer-bff: ~40 个 handler 函数
- merchant-bff: ~50 个 handler 函数
- admin-bff: ~30 个 handler 函数
- 前端无需改动（已在 request() 中展示 body.msg）

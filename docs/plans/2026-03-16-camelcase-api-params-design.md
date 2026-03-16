# API 参数命名统一为 camelCase

## 背景

前后端参数命名风格不一致，部分使用 snake_case，部分使用 camelCase，导致请求绑定失败（如 `pageSize` vs `page_size`）。

## 规则

- BFF handler 所有 `json:` 和 `form:` tag 统一为 camelCase
- BFF response 中的 JSON key 统一为 camelCase
- 前端 TypeScript interface 字段名统一为 camelCase
- gRPC proto、数据库列名、Go struct 字段名不变

## 改动范围

### 后端
1. merchant-bff handler — request struct tag + response map key
2. consumer-bff handler — request struct tag + response map key

### 前端
3. merchant-frontend — types/ interface 字段 + pages/ 引用
4. consumer-frontend (frontend/) — types/ interface 字段 + pages/ 引用

## 执行顺序

1. merchant-bff request tag
2. merchant-bff response 构建
3. merchant-frontend types + pages
4. consumer-bff request tag
5. consumer-bff response 构建
6. consumer-frontend types + pages
7. Go build + tsc --noEmit 验证

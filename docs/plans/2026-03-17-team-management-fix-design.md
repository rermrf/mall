# Fix: Team Management roles.map / rawData.some Errors

**Date:** 2026-03-17
**Status:** Approved

## Problem

Merchant frontend team management shows two errors:
1. Employee list: `roles.map is not a function`
2. Role list: `rawData.some is not a function`

## Root Cause

BFF `ListRoles` handler (`merchant-bff/handler/user.go:100`) returns the raw protobuf response object:

```go
return ginx.Result{Code: 0, Msg: "success", Data: resp}, nil
```

`resp` is `*ListRolesResponse` which serializes to `{ "roles": [...] }` — an object with a `roles` property. The frontend expects a flat `Role[]` array.

When the frontend receives `{ roles: [...] }`:
- `StaffList.tsx:14` — `setRoles(r ?? [])` stores the object (truthy, no fallback to `[]`), then `roles.map()` fails
- `RoleList.tsx:52` — `{ data: data ?? [], success: true }` passes the object to ProTable, which calls `.some()` on it

## Fix

Change `user.go:100`:

```go
// Before
return ginx.Result{Code: 0, Msg: "success", Data: resp}, nil

// After
return ginx.Result{Code: 0, Msg: "success", Data: resp.GetRoles()}, nil
```

**Scope:** 1 file, 1 line. No frontend changes needed.

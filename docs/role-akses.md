# Modul: Role & Akses (RBAC + Audit)

Route FE: `http://localhost:5173/role-akses`
File FE: `src/pages/RoleAksesPage.tsx`, `src/components/RAHeader.tsx`, `src/components/RAFilterBars.tsx`, `src/components/RAKpi.tsx`, `src/components/RACharts.tsx`, `src/components/RATables.tsx`, `src/components/RAModals.tsx`, `src/components/RAPermissionEditor.tsx`
Type/domain: `src/lib/role-access.types.ts` (Role, User, AuditEntry), data: `src/lib/mockRoleAccess.ts`

> **Catatan SaaS**: `users` adalah sumber kebenaran (lihat `auth.md`). Tabel `role_outlets` & `user_outlets` redundan secara desain (role scope + user override) - FE membaca union-nya.

## Ringkasan Fitur
- Role dengan scope `company` (semua outlet) atau `store` (per outlet, daftar `storeIds`)
- Role sistem (`isSystem=true`, mis. `Owner`) tidak bisa dihapus
- Permission matrix: 13 modul x 11 aksi = 143 sel
- Modul: dashboard, kasir, produk, kategori, stok, pelanggan, supplier, outlet, laporan_penjualan, laporan_stok, laporan_keuangan, role_akses, integrasi
- Aksi: view, create, edit, delete, approve, export, print, void, refund, assign, toggle (void/refund/approve = critical)
- User: 1 user = 1 role + N outlet (`User.storeIds[]`)
- Audit log: role_create, role_update, role_delete, role_toggle, permission_change, user_assign, user_unassign, store_assign, store_unassign (disimpan ke `audit_logs` umum, lihat `audit-logs.md`)

## Skema Tabel Backend (Production)

### `tenants` (sumber kebenaran)
- `id` UUID PK
- lihat `auth.md` untuk kolom lengkap

### `roles`
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `name` varchar
- `description` text nullable
- `scope` enum `('company','store')`
- `is_system` bool default false (Owner = true; auto-seed saat tenant dibuat)
- `status` enum `('aktif','nonaktif')` default `aktif`
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`
- `unique (tenant_id, name)`

### `role_outlets` (scope 'store' - outlet yang role ini boleh akses)
- `role_id` UUID FK `roles`
- `outlet_id` UUID FK `outlets`
- PK komposit `(role_id, outlet_id)`
- `tenant_id` UUID (denormalized untuk RLS)

### `permissions` (katalog global, shared lintas tenant)
- `id` UUID PK
- `code` varchar unique (e.g. `product.read`, `product.write`, `stock.adjust`, `kasir.void`)
- `module` varchar (e.g. `product`, `kasir`, `stok`)
- `action` enum `('view','create','edit','delete','approve','export','print','void','refund','assign','toggle')`
- `critical` bool default false (untuk aksi `void|approve|refund` -> true)
- `description` text nullable
- Seed awal: 13 modul x 11 aksi = 143 baris

### `role_permissions` (override per role; absent = false)
- `role_id` UUID FK `roles`
- `permission_id` UUID FK `permissions`
- `granted` bool default true
- `tenant_id` UUID (denormalized untuk RLS + audit)
- PK komposit `(role_id, permission_id)`

### `users` (sumber kebenaran)
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `name` varchar
- `username` varchar
- `email` varchar
- `password_hash` text (jangan pernah balikan plaintext)
- `role_id` UUID FK `roles`
- `status` enum `('aktif','nonaktif')` default `aktif`
- `last_login_at` timestamp nullable
- `last_login_ip` inet nullable
- `failed_login_count` int default 0
- `locked_until` timestamp nullable
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable
- lihat `auth.md` untuk kolom auth lainnya

### `user_outlets` (scoping user ke outlet; override per-user)
- `user_id` UUID FK `users`
- `outlet_id` UUID FK `outlets`
- PK komposit `(user_id, outlet_id)`
- `tenant_id` UUID
- **Union rule**: outlet yang user boleh akses = `role_outlets` (jika role scope='store') union `user_outlets` (override eksplisit)

### `audit_logs` (umum, lihat `audit-logs.md`)
- Modul ini tulis entry dengan `entity_type='role'|'user'|'permission'`, `action` dari enum di bawah
- `action` values untuk role/user: `('role_create','role_update','role_delete','role_toggle','permission_change','user_create','user_update','user_delete','user_toggle','user_assign','user_unassign','store_assign','store_unassign','login','logout','password_change')`
- `diff` JSONB menyimpan before/after untuk perubahan role/permission
- `target_type` enum `('role','user','permission','session')`
- `target_id` UUID
- `target_name` varchar (snapshot)

## Field yang Diturunkan (FE)
| Field FE | Sumber di DB |
|---|---|
| `Role.id/name/description` | `roles.*` |
| `Role.companyId` | `roles.tenant_id` |
| `Role.scope` | `roles.scope` |
| `Role.status` | `roles.status` |
| `Role.isSystem` | `roles.is_system` |
| `Role.permissions` | `role_permissions` join `permissions` -> `Record<module, Record<action, bool>>` (hanya yang `granted=true`) |
| `Role.userCount` | `count(users where role_id = r.id and deleted_at is null)` (denormalized di `roles.user_count` opsional) |
| `Role.storeIds` | `role_outlets.outlet_id[]` (kosong jika scope company) |
| `User.outlets` | `user_outlets.outlet_id[]` |
| `User.roleId` | `users.role_id` |
| `AuditEntry.*` | `audit_logs` where `entity_type IN ('role','user','permission')` order by `at desc` |

## Index Penting
- `unique (tenant_id, name)` di `roles`
- `unique (tenant_id, username)` di `users`
- `users(tenant_id, role_id, status)`
- `role_outlets(tenant_id, outlet_id)`
- `user_outlets(tenant_id, outlet_id)`
- `role_permissions(tenant_id, role_id)`
- `permissions(module, action)` - untuk permission editor

## API Minimal

### Roles
- `GET /api/roles?q=&status=&scope=`
- `GET /api/roles/:id` (ikut `permissions[]`, `userCount`, `storeIds[]`)
- `POST /api/roles` (body: `{name, description?, scope, storeIds[], permissions}` - `permissions` berupa list `{module, action, granted}[]`)
- `PATCH /api/roles/:id` (update name/description/scope/storeIds)
- `PATCH /api/roles/:id/permissions` (replace full matrix)
- `POST /api/roles/:id/toggle-status` (untuk role non-system)
- `DELETE /api/roles/:id` (tolak jika `is_system=true` atau ada user aktif)

### Users
- `GET /api/users?q=&roleId=&outletId=&status=&page=&pageSize=`
- `GET /api/users/:id` (ikut `outlets[]`, `roleName`, `permissions[]`)
- `POST /api/users` (body: `{name, username, email, password, roleId, storeIds[]}`)
- `PATCH /api/users/:id` (termasuk `roleId`)
- `POST /api/users/:id/reset-password` (admin only)
- `POST /api/users/:id/toggle-status`
- `POST /api/users/:id/assign-outlets` (body: `outletIds[]` - replace)
- `DELETE /api/users/:id` (soft delete)

### Permissions & Audit
- `GET /api/permissions/catalog` (balik daftar `module` x `action` untuk permission editor)
- `GET /api/audit?entity=role&entityId=&actor=&action=&from=&to=&page=&pageSize=` (lihat `audit-logs.md`)

## Catatan SaaS
- `tenant_id` wajib di `roles`/`role_outlets`/`user_outlets`/`role_permissions` + RLS
- **Single source of truth**: tidak ada `employees` table di production; `Employee` di FE = read-only alias dari `users` (lihat `auth.md`)
- Role sistem `Owner` auto-seed saat tenant dibuat; tidak bisa dihapus/diubah `is_system`/`name`
- Role scope `store` + `storeIds` kosong saat create: tolak (wajib pilih minimal 1 outlet)
- Perubahan permission harus invalidasi cache/JWT user terkait (event-driven)
- Tiap perubahan tulis ke `audit_logs` (immutable) - lihat `audit-logs.md`. Termasuk aksi-aksi dari modul `karyawan.md` (user_create, user_assign, store_assign, dll).
- Password: bcrypt/argon2 + salt; throttle login (rate limit, lihat `auth.md`)


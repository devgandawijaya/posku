# Modul: Auth & Tenant

Route FE: `http://localhost:5173/login`
File FE: `src/pages/LoginPage.tsx`, `src/components/LoginView.tsx`
Type/domain: `src/models/auth.model.ts` (Company, Store, Employee, LoginPayload, LoginResponse), controller: `src/controllers/auth.controller.ts`, service: `src/services/auth.service.ts`, storage: `src/services/storage.service.ts`, api: `src/services/api.ts`

> **Catatan SaaS**: production pakai tabel `tenants` + `users` (lihat `role-akses.md`). `Employee`/`Company`/`Store` di FE adalah alias read-only untuk response login; bukan tabel produksi.

## Ringkasan Fitur
- Login via `username + password`
- Response: `{ employee, token, refreshToken, expiresIn }`
- `employee` membawa `company`, `store`, `role` (nama) -> di BE = join `tenants` + `outlets` + `roles`
- Token disimpan di `localStorage.token`; auto-attach via axios interceptor (`Authorization: Bearer <token>`)
- Multi-tenant: setiap user terkait `tenant_id` (UUID)
- Multi-outlet: user terkait 0..N outlet via `user_outlets` (kasir = 1 outlet, admin/owner = 0 = semua)

## Skema Tabel Backend (Production)

### `tenants` (sumber kebenaran; sebelumnya dinamai `companies` di mock FE)
- `id` UUID PK
- `name` varchar
- `slug` varchar unique
- `address` text nullable
- `plan` enum `('free','pro','enterprise')` default `free`
- `status` enum `('active','suspended','trial')` default `trial`
- `trial_ends_at` timestamp nullable
- `owner_user_id` UUID nullable FK `users` (set setelah user owner dibuat)
- `created_at`, `updated_at`, `deleted_at` nullable
- lihat juga: `subscription-billing.md` (`plans`, `subscriptions`, `invoices`)

### `outlets` (sumber kebenaran; sebelumnya `stores` di mock FE)
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `code` varchar unique per tenant
- `name` varchar
- `address` text nullable
- `phone` varchar nullable
- `is_active` bool default true
- lihat detail di `outlet.md`

### `users` (sumber kebenaran; sebelumnya `employees` di mock FE)
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `name` varchar
- `username` varchar
- `email` varchar
- `password_hash` text (bcrypt/argon2)
- `role_id` UUID FK `roles` (lihat `role-akses.md`; **bukan enum hardcoded**)
- `status` enum `('aktif','nonaktif')` default `aktif`
- `last_login_at` timestamp nullable
- `last_login_ip` inet nullable
- `failed_login_count` int default 0
- `locked_until` timestamp nullable
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable
- lihat detail di `role-akses.md`

### `user_outlets` (multi-outlet; sebelumnya tidak ada di `employees`)
- `user_id` UUID FK `users`
- `outlet_id` UUID FK `outlets`
- PK komposit `(user_id, outlet_id)`
- `tenant_id` UUID (denormalized untuk RLS)
- lihat juga `role-akses.md` (table identik)

## Tabel Auth

### `refresh_tokens`
- `id` UUID PK
- `user_id` UUID FK `users`
- `tenant_id` UUID (denormalized)
- `token_hash` text (sha256 dari refresh token asli)
- `device_info` text nullable
- `ip` inet nullable
- `user_agent` text nullable
- `issued_at` timestamp
- `expires_at` timestamp
- `revoked_at` timestamp nullable
- `replaced_by` UUID nullable FK `refresh_tokens`

### `password_resets`
- `id` UUID PK
- `user_id` UUID FK `users`
- `token_hash` text
- `expires_at` timestamp
- `used_at` timestamp nullable
- `requested_ip` inet nullable
- `created_at` timestamp

### `login_audit`
- `id` UUID PK
- `tenant_id` UUID nullable
- `user_id` UUID nullable
- `username_input` varchar
- `result` enum `('success','invalid_credentials','locked','disabled','not_found','rate_limited','two_factor_required')`
- `ip` inet
- `user_agent` text
- `at` timestamp
- `index (tenant_id, at desc)` - partition by month

## Field yang Diturunkan (FE `LoginResponse.employee`)
> FE `Employee` adalah alias read-only. Backend memetakan dari `users` join `tenants`/`outlets`/`roles`.

| Field FE | Sumber di DB |
|---|---|
| `employee.id` | `users.id` |
| `employee.company_id` | `users.tenant_id` |
| `employee.company` | `tenants.*` (join) |
| `employee.store_id` | outlet pertama dari `user_outlets` (single default; null untuk admin) |
| `employee.store` | `outlets.*` (join) |
| `employee.name` | `users.name` |
| `employee.username` | `users.username` |
| `employee.email` | `users.email` |
| `employee.role` | `roles.name` (string, diturunkan dari `users.role_id`) |
| `employee.outlets[]` (extensi opsional) | `user_outlets.outlet_id[]` |
| `token` | JWT signed (access); refresh via `refresh_tokens` |

## Index Penting
- `unique (tenant_id, username)` di `users`
- `users(tenant_id, status)`, `users(tenant_id, role_id)`
- `refresh_tokens(user_id, revoked_at, expires_at)`
- `login_audit(tenant_id, at desc)` - partition by month

## API Minimal

### Public
- `POST /api/auth/login` (body: `{username, password, tenantSlug?}`) -> `{employee, token, refreshToken, expiresIn, permissions[], outletIds[]}`
- `POST /api/auth/refresh` (body: `{refreshToken}`) -> `{token, refreshToken}`
- `POST /api/auth/logout` (body: `{refreshToken}`) - revoke refresh token
- `POST /api/auth/forgot-password` (body: `{email|username, tenantSlug}`)
- `POST /api/auth/reset-password` (body: `{token, newPassword}`)
- `POST /api/auth/change-password` (auth required; body: `{oldPassword, newPassword}`)

### Authenticated
- `GET /api/auth/me` -> `{employee, role, permissions[], outlets[]}` (untuk refresh FE state)

## Keamanan
- `password_hash`: bcrypt cost 12 / argon2id
- JWT access token: 15-60 menit, klaim: `sub (user_id)`, `tenant_id`, `role_id`, `outlet_ids[]`, `permissions[]`
- Refresh token: 7-30 hari, rotation (revoke lama saat issue baru); simpan hanya `token_hash` di DB
- Rate limit login: 5 attempt / 15 menit / IP; lock `locked_until` setelah 10x gagal
- Audit log login wajib (success/failure) -> `login_audit`
- HTTPS only; cookie httpOnly untuk refresh (opsional)
- 2FA TOTP (opsional) untuk owner/admin: tambah tabel `user_2fa`

## Catatan SaaS
- **Single source of truth**: `tenants` + `users` (lihat juga `role-akses.md`); hapus duplikat `companies`/`employees` di production
- `tenants.id` UUID (FE mock pakai `number` 1; production migrasi ke UUID)
- Saat `users` dibuat, auto-assign ke `user_outlets` jika role-nya store-scoped (lihat `role-akses.md.roles.scope`)
- `tenants.owner_user_id` di-set setelah user owner pertama dibuat
- Logout harus revoke refresh token (tidak bisa dipakai ulang)
- FE yang sudah pakai `Employee.role: string` aman selama backend isi dengan `roles.name` (string)
- Modul lain yang FK ke `users`: `karyawan.md` (definisi & multi-outlet placement), `outlet.md` (manager), `supplier.md`, `pelanggan.md`, `transaksi.md` (cashier), `stok.md` (created_by), `retur.md`, `laporan-keuangan.md` (expenses/payroll), `subscription-billing.md`, `integrasi.md`, `role-akses.md`. Semua sudah benar mengarah ke `users`.


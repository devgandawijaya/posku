# Modul: Karyawan (Employees / Staff per Outlet)

Route FE: `http://localhost:5173/role-akses` (tab "User" di `RoleAksesPage`)
File FE: `src/pages/RoleAksesPage.tsx`, `src/components/RATables.tsx`, `src/components/RAFilterBars.tsx`, `src/components/RAKpi.tsx`, `src/components/RACharts.tsx`
Type/domain: `src/lib/role-access.types.ts` (`User`), data: `src/lib/mockRoleAccess.ts` (`mockUsers`), hook: `src/lib/useRoleAccess.ts`

> **Konsep SaaS**: `karyawan` = `users` (sumber kebenaran, lihat `auth.md`). Setiap karyawan **milik satu `tenant`** (perusahaan) dan **bisa ditempatkan di banyak `outlet`** lewat tabel pivot `user_outlets`. Penempatan menentukan outlet mana yang boleh diakses saat login.

## Ringkasan Fitur
- CRUD karyawan per tenant
- Field: `id`, `name`, `username`, `email`, `roleId`, `storeIds[]`, `status` (`aktif`/`nonaktif`)
- Placeholder field: `lastLoginAt`, `createdAt`
- Filter: search (nama/username/email), role, store, status
- Aksi: assign role, assign outlet (replace), toggle status, soft delete
- KPI: total user, user tanpa role, distribusi per role, distribusi per outlet
- Hak akses granular: `view` semua, `create/delete/assign/toggle` owner/admin, `edit` owner/admin
- Audit log ke `audit_logs` umum (lihat `audit-logs.md`) dengan `action` `user_create|user_update|user_delete|user_toggle|user_assign|user_unassign|store_assign|store_unassign`

## Skema Tabel Backend

### `users` (sumber kebenaran, lihat juga `auth.md`)
- `id` UUID PK
- `tenant_id` UUID FK `tenants` (**wajib**)
- `name` varchar
- `username` varchar
- `email` varchar
- `password_hash` text
- `role_id` UUID FK `roles` (lihat `role-akses.md`; **nullable** untuk user orphan, harus ditolak di service)
- `status` enum `('aktif','nonaktif')` default `aktif`
- `last_login_at` timestamp nullable
- `last_login_ip` inet nullable
- `failed_login_count` int default 0
- `locked_until` timestamp nullable
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable
- `unique (tenant_id, username)`
- `unique (tenant_id, email)` (opsional, untuk cegah duplikat)

### `user_outlets` (penempatan karyawan ke outlet, **multi-outlet**)
- `user_id` UUID FK `users`
- `outlet_id` UUID FK `outlets`
- PK komposit `(user_id, outlet_id)`
- `tenant_id` UUID (denormalized untuk RLS)
- `assigned_at` timestamp default now()
- `assigned_by` UUID FK `users` nullable

### `user_metrics_cache` (denormalized untuk KPI tabel)
- `user_id` UUID PK
- `tenant_id` UUID
- `outlet_count` int
- `last_active_at` timestamp
- `transaction_count_30d` int
- `updated_at` timestamp

### `employee_invitations` (opsional, onboarding via email)
- `id` UUID PK
- `tenant_id` UUID
- `email` varchar
- `role_id` UUID FK `roles` nullable
- `outlet_ids` UUID[] (snapshot saat invite)
- `token_hash` text
- `invited_by` UUID FK `users`
- `expires_at` timestamp
- `accepted_at` timestamp nullable
- `created_at` timestamp

## Relasi ke Modul Lain
- `outlets` (lihat `outlet.md`): karyawan ditempatkan di 0..N outlet
- `roles` (lihat `role-akses.md`): setiap karyawan punya 1 role
- `shifts` (lihat `kasir.md`): shift kasir diikat ke `user_id` + `outlet_id`
- `sales` (lihat `transaksi.md`): `cashier_user_id` FK ke `users.id`
- `stock_movements` (lihat `stok.md`): `created_by` FK ke `users.id`
- `audit_logs` (lihat `audit-logs.md`): `actor_user_id` FK ke `users.id`

## Aturan Akses (Union Rule)
Outlet yang boleh diakses karyawan = `(role.scope = 'store' ? role_outlets.outlet_id[] : semua outlet di tenant)` **union** `user_outlets.outlet_id[]`.

Contoh:
- Role `Manager Toko` scope `store` dengan `storeIds=[t1, t2]`, user override `storeIds=[t3]` -> akses = `[t1, t2, t3]`
- Role `Owner` scope `company` -> akses semua outlet; user_outlets boleh kosong atau partial

Service saat query harus: `outlet_id IN (effective_outlets)`.

## Field yang Diturunkan (dari `User` FE)
| Field FE | Sumber di DB |
|---|---|
| `id` | `users.id` |
| `companyId` | `users.tenant_id` |
| `name` | `users.name` |
| `username` | `users.username` |
| `email` | `users.email` |
| `roleId` | `users.role_id` (string kosong "" untuk orphan) |
| `storeIds` | `user_outlets.outlet_id[]` (replace; UI pakai `Set<string>`) |
| `status` | `users.status` |
| `lastLoginAt` | `users.last_login_at` |
| `createdAt` | `users.created_at` |

## Index Penting
- `unique (tenant_id, username)` di `users`
- `unique (tenant_id, email)` di `users` (opsional)
- `users(tenant_id, role_id, status)`
- `users(tenant_id, status, name)` - untuk search
- `user_outlets(tenant_id, outlet_id)` - untuk filter "karyawan per outlet"
- `user_outlets(tenant_id, user_id)` - untuk fetch "outlet user"

## API Minimal

### CRUD Karyawan
- `GET /api/users?q=&roleId=&outletId=&status=&page=&pageSize=` (balik `User` + `roleName` + `outlets[]`)
- `GET /api/users/:id` (ikut `role`, `permissions[]`, `outlets[]`, `metrics`)
- `POST /api/users` (body: `{name, username, email, password, roleId, storeIds[]}`)
  - Validasi: `roleId` harus ada di tenant; `storeIds` harus subset dari role scope `store` (atau semua outlet jika role scope `company`)
  - Default `status='aktif'`
  - Auto-hash password, audit `user_create`
- `PATCH /api/users/:id` (body partial: `name?, email?, roleId?, status?`)
- `POST /api/users/:id/reset-password` (admin/owner only; kirim password baru via email/in-app notif)
- `POST /api/users/:id/toggle-status` (active <-> inactive)
- `DELETE /api/users/:id` (soft delete: set `deleted_at`; tolak jika ada shift/sales open yang reference)

### Penempatan Outlet
- `GET /api/users/:id/outlets`
- `POST /api/users/:id/assign-outlets` (body: `outletIds[]` - **replace** sesuai pola FE `assignUserStores`)
  - Validasi: setiap outlet harus milik tenant yang sama
  - Audit `store_assign` (atau `store_unassign` jika shrinkage)
- `POST /api/users/:id/add-outlet` / `POST /api/users/:id/remove-outlet` (alternatif incremental, opsional)

### Import & Invitasi
- `POST /api/users/invite` (body: `{email, roleId, outletIds[]}` -> create `employee_invitations` + kirim email)
- `POST /api/users/accept-invite` (public; body: `{token, password, name, username}`)
- `POST /api/users/bulk-import` (multipart CSV; validasi + create many)

### Lookup
- `GET /api/users/options?role=&q=&outletId=` (dropdown `{id, name, username, roleName}`)
- `GET /api/users/by-outlet/:outletId` (untuk `outlet_metrics_cache.employee_count` dan outlet detail)
- `GET /api/users/me/outlets` (balik `effective_outlets` untuk user saat ini)

## Hak Akses Granular
- `user.view` - semua role internal
- `user.create` - owner
- `user.edit` - owner, admin
- `user.delete` - owner
- `user.toggle` - owner, admin
- `user.assign` (role + outlet) - owner, admin
- `user.reset_password` - owner, admin
- Tiap permission di-check di service (bukan hanya di FE) -> lihat `role-akses.md` untuk matrix

## Catatan SaaS
- **Single source of truth**: `users` (lihat `auth.md`); tidak ada tabel `employees`/`staff` terpisah di production
- `tenant_id` wajib di `users` & `user_outlets` + RLS
- Penempatan outlet adalah data operasional, bukan struktur tetap: bisa berubah sewaktu (promosi, mutasi)
- Saat hapus user, pertimbangkan: shift terbuka, sales open, return pending -> set `status='nonaktif'` lebih aman daripada hard delete
- Audit log ke `audit_logs` (lihat `audit-logs.md`) dengan `entity_type='user'`, `action` sesuai enum
- Password reset: token sekali pakai, expire 24 jam, hash di DB
- Invitasi email: throttle (max 5/jam per tenant) untuk cegah spam
- Untuk superadmin posku (internal): bisa bypass `tenant_id` via scope khusus (lihat `audit-logs.md`)

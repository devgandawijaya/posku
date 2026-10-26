# Modul: Outlet (Multi-Store)

Route FE: `http://localhost:5173/outlet`
File FE: `src/pages/OutletPage.tsx`, `src/components/OutletHeader.tsx`, `src/components/OutletTable.tsx`, `src/components/OutletFormModal.tsx`, `src/components/OutletKpi.tsx`, `src/components/OutletCharts.tsx`
Type/domain: `src/lib/outlet.types.ts`, hook: `src/lib/useOutlet.ts`

## Ringkasan Fitur
- CRUD outlet dalam 1 tenant
- Field: nama, kode (auto `OUT-0001`), alamat, telepon, nama manager, tanggal buka, status (aktif/nonaktif), notes
- Filter: status, dengan/tanpa transaksi, pencarian
- KPI: total, aktif, nonaktif, total revenue, total trx, rata-rata trx/outlet, total karyawan
- Denormalized metrics: `totalSales`, `totalTransactions`, `lastTransactionAt`, `employees`

## Skema Tabel Backend

### `outlets`
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `code` varchar unique per tenant (auto-generated `OUT-XXXX`)
- `name` varchar
- `address` text
- `phone` varchar
- `manager_user_id` UUID nullable FK `users` (link ke employee/manager)
- `manager_name` varchar (snapshot untuk tampilan)
- `status` enum `('aktif', 'nonaktif')`
- `opened_at` date
- `notes` text nullable
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable

### `outlet_metrics_cache` (denormalized, dihitung via scheduled job / trigger)
- `outlet_id` UUID PK
- `tenant_id` UUID
- `total_sales` numeric(18,2) (lifetime)
- `total_transactions` int
- `last_transaction_at` timestamp nullable
- `employee_count` int
- `updated_at` timestamp

## Field yang Diturunkan (dari `Outlet` FE)
| Field FE | Sumber di DB |
|---|---|
| `id`, `code`, `name` | `outlets.*` |
| `companyId` | `outlets.tenant_id` |
| `address`, `phone` | `outlets.address/phone` |
| `managerName` | `outlets.manager_name` (snapshot) |
| `managerUserId` | `outlets.manager_user_id` |
| `status` | `outlets.status` |
| `openedAt` | `outlets.opened_at` (ISO) |
| `totalSales` | `outlet_metrics_cache.total_sales` |
| `totalTransactions` | `outlet_metrics_cache.total_transactions` |
| `lastTransactionAt` | `outlet_metrics_cache.last_transaction_at` |
| `employees` | `outlet_metrics_cache.employee_count` |
| `notes` | `outlets.notes` |

## Index Penting
- `unique (tenant_id, code)`
- `index (tenant_id, status)`
- `index (tenant_id, manager_user_id)`

## API Minimal
- `GET /api/outlets?status=&hasTrx=&q=&page=&pageSize=`
- `GET /api/outlets/:id`
- `POST /api/outlets` (body: `name, address, phone, managerUserId, openedAt, notes?`)
- `PATCH /api/outlets/:id`
- `DELETE /api/outlets/:id` (soft delete; tolak bila `totalTransactions > 0`)
- `POST /api/outlets/:id/toggle-status` (aktif <-> nonaktif)
- `GET /api/outlets/:id/metrics?from=&to=`
- `GET /api/outlets/options` (dropdown: `{id, name}` untuk modul lain)

## Hak Akses
- `outlet.view` semua role
- `outlet.create` / `outlet.edit` / `outlet.toggle` / `outlet.delete` -> modul role akses (lihat `role-akses.md`)
- Hanya owner yang boleh delete outlet

## Catatan SaaS
- Relasi: setiap outlet memiliki `manager_user_id` FK ke `users` (lihat `karyawan.md`). Manager adalah karyawan biasa yang di-flag sebagai manager outlet - bisa multi-outlet via `user_outlets` (lihat `karyawan.md.employee_count`).

- `tenant_id` wajib + RLS
- `manager_user_id` harus milik tenant yang sama (validasi service)
- Audit log create/update/delete/toggle
- Setiap kali `sales` / `users.outlets` berubah, refresh `outlet_metrics_cache` (event-driven)

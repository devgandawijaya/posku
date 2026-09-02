# Modul: Pelanggan (Customer / Member)

Route FE: `http://localhost:5173/pelanggan`
File FE: `src/pages/PelangganPage.tsx`, `src/components/PelangganHeader.tsx`, `src/components/PelangganTable.tsx`, `src/components/PelangganKpi.tsx`, `src/components/PelangganCharts.tsx`
Type/domain: `src/lib/pelanggan.types.ts`, hook: `src/lib/usePelanggan.ts`

## Ringkasan Fitur
- CRUD pelanggan (member) + tier (bronze/silver/gold/vip)
- Multi-outlet registration: `registeredStoreIds[]`
- `favoriteStoreId` (toko paling sering dikunjungi)
- Poin member (`pointsBalance`)
- Total transaksi + total spent
- KPI: total member, per-tier, avg spent, poin beredar, top spender

## Skema Tabel Backend

### `customers`
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `member_code` varchar unique per tenant (auto `MBR00001`)
- `name` varchar
- `email` varchar
- `phone` varchar
- `tier` enum `('bronze','silver','gold','vip')` default `'bronze'`
- `points_balance` int default 0
- `joined_at` timestamp
- `last_visit_at` timestamp nullable
- `favorite_outlet_id` UUID nullable FK `outlets`
- `status` enum `('aktif', 'nonaktif')` default `aktif`
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable

### `customer_outlets` (mapping toko tempat member terdaftar)
- `customer_id` UUID FK
- `outlet_id` UUID FK `outlets`
- `registered_at` timestamp default now()
- PK komposit `(customer_id, outlet_id)`
- `tenant_id` UUID

### `customer_metrics_cache`
- `customer_id` UUID PK
- `tenant_id` UUID
- `total_transactions` int
- `total_spent` numeric(18,2)
- `last_visit_at` timestamp
- `favorite_outlet_id` UUID nullable
- `updated_at` timestamp

### `points_ledger` (audit pergerakan poin)
- `id` UUID PK
- `tenant_id` UUID
- `customer_id` UUID FK
- `delta` int (signed: + earn, - redeem)
- `reason` enum `('earn','redeem','adjust','expire')`
- `ref_type` (`sale`, `manual`, `voucher`), `ref_id` UUID
- `note` text nullable
- `created_by` UUID FK `users`
- `created_at` timestamp

## Field yang Diturunkan (dari `Customer` FE)
| Field FE | Sumber di DB |
|---|---|
| `id`, `name`, `email`, `phone` | `customers.*` |
| `memberCode` | `customers.member_code` |
| `tier` | `customers.tier` (auto recompute di service dari `total_spent` jika rule berubah) |
| `registeredStoreIds` | `customer_outlets.outlet_id[]` |
| `favoriteStoreId` | `customer_metrics_cache.favorite_outlet_id` |
| `totalTransactions` | `customer_metrics_cache.total_transactions` |
| `totalSpent` | `customer_metrics_cache.total_spent` |
| `pointsBalance` | `customers.points_balance` (denormalized dari `points_ledger` sum) |
| `joinedAt` | `customers.joined_at` |
| `lastVisit` | `customers.last_visit_at` |

## Index Penting
- `unique (tenant_id, member_code)`
- `index (tenant_id, tier)`
- `index (tenant_id, email)`, `index (tenant_id, phone)` (untuk search)
- `customer_outlets(tenant_id, outlet_id)`

## API Minimal
- `GET /api/customers?storeId=&q=&tier=&page=&pageSize=`
- `GET /api/customers/:id`
- `POST /api/customers` (body: `name, email, phone, outletIds[]`)
- `PATCH /api/customers/:id`
- `DELETE /api/customers/:id` (soft delete; gabungkan ke `deleted_at` agar histori trx tetap)
- `POST /api/customers/:id/points` (adjust poin,catat di `points_ledger`)
- `GET /api/customers/:id/transactions?from=&to=&page=`
- `GET /api/customers/options?q=` (dropdown kasir: `{id, memberCode, name}`)

## Catatan SaaS
- `tenant_id` wajib + RLS
- Email/phone sebaiknya unik per tenant (mencegah duplikat global bila perlu, tambahkan `unique (tenant_id, email)` setelah validasi)
- PII sensitif: hash/log akses di audit log
- Penentuan tier otomatis (cron) berdasarkan `total_spent` rolling 12 bulan

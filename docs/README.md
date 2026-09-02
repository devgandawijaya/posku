# Backend Spec - POSKu (SaaS Multi-Tenant)

Spesifikasi kebutuhan tabel & API backend untuk aplikasi POSKu. Setiap modul punya file `.md` terpisah dengan pola yang konsisten: ringkasan fitur FE -> skema tabel -> pemetaan field -> API -> catatan SaaS.

## Status Implementasi (Go/Gin/GORM, `internal/`)
Implementasi saat ini memakai konvensi existing codebase (ID `uint` auto-increment, `Company`=tenant, `Store`=outlet, `Employee`=user) alih-alih migrasi penuh ke UUID/`tenants`/`outlets`/`users` seperti draft skema di bawah, supaya tidak menghancurkan data & kode yang sudah ada.

Seluruh 19 modul dokumentasi sudah memiliki model, controller & endpoint dasar. Detail checklist
per modul (fitur mana yang sudah vs belum, mis. workflow berjenjang, cache/materialized view,
webhook eksternal, enkripsi, cron job) ada di [CHECKLIST.md](CHECKLIST.md).

## Prinsip Arsitektur
- **Multi-tenant**: `tenant_id` di semua tabel bisnis + Row-Level Security (RLS)
- **Shared DB, shared schema** + filter `tenant_id` di service / RLS
- **UUID PK** untuk skalabilitas horizontal (hindari sequence per-tenant)
- **Soft delete** (`deleted_at`) di tabel yang boleh restore
- **Audit log** append-only untuk semua perubahan (lihat `audit-logs.md`)
- **Snapshot field** (e.g. `sku_snapshot`, `name_snapshot`, `api_key_masked`) untuk kestabilan histori
- **Materialized view / cache** untuk agregat laporan (refresh terjadwal)
- **Idempotency** di payment & checkout (cegah duplikat saat retry)

## Index Modul

### Core
- `auth.md` - Login, refresh token, JWT, login_audit, source-of-truth `tenants` & `users`
- `role-akses.md` - RBAC (roles, permissions, role_permissions, role_outlets, role_audit)
- `karyawan.md` - Definisi user SaaS + penempatan multi-outlet (`user_outlets`), KPI per role/outlet, reset password, invitation
- `tenant` lihat `auth.md` (sumber kebenaran `tenants` UUID) + `subscription-billing.md`
### Master Data
- `product.md` - Produk, SKU, barcode, gambar, multi-barcode, price history
- `kategori.md` - Kategori + subkategori + scope per outlet
- `outlet.md` - Outlet + metrics cache
- `supplier.md` - Supplier + assign outlet
- `pelanggan.md` - Customer/member + tier + poin

### Operasional
- `kasir.md` - Shift, cart, voucher, promo, payment, checkout
- `transaksi.md` - Sales header + items + refund + audit
- `retur.md` - Retur/refund + approval workflow + fate
- `stok.md` - Stocks per outlet + movements + transfer + adjustment

### Reporting
- `laporan-penjualan.md` - Sales summary, timeseries, by payment, by store
- `laporan-stok.md` - Stock summary, movement, top low
- `laporan-keuangan.md` - P&L, cashflow, expense, payroll

### Integration & Platform
- `integrasi.md` - Payment/delivery/accounting/marketplace/pos/api
- `dashboard.md` - KPI, alerts, shift panel, tenant store monitor, SaaS metrics
- `subscription-billing.md` - Plans, subscriptions, invoices, payments
- `audit-logs.md` - Cross-module audit trail

## ER Diagram (Ringkas)

```
                          +------------------+
                          |    tenants       |
                          | (companies)      |
                          +------------------+
                                  |
            +---------------------+----------------------+
            |                     |                      |
   +----------------+    +-----------------+    +------------------+
   |   outlets      |    |    users        |    | subscriptions    |
   |  (stores)      |    |  (employees)    |    |  -> plans        |
   +----------------+    +-----------------+    +------------------+
            |                     |                      |
   +----------------+    +-----------------+    +------------------+
   |  user_outlets  |    | roles           |    | invoices         |
   |  (karyawan.md) |    |  -> permissions |    |  -> payments     |
   +----------------+    | role_outlets    |   +------------------+
                         | role_permissions|
                         +-----------------+

   MASTER DATA
   +-------------+   +-----------+   +-----------+   +-----------+
   | categories  |<--| products  |-->| units     |   | suppliers |
   |  (self-ref) |   |  (sku,    |   +-----------+   +-----------+
   +-------------+   |   barcode)|
                     | price/cost|
                     +-----------+
                           |
            +--------------+---------------+
            |                              |
   +----------------+              +----------------+
   | product_       |              | product_       |
   | images         |              | barcodes       |
   +----------------+              +----------------+
            |
   +----------------+   +-----------+   +-----------+
   | stocks         |-->| outlets   |   | customers |
   | (per outlet)   |   +-----------+   +-----------+
   +----------------+                      |
            |                             |
   +----------------+               +-----------+
   | stock_movements|               | points_   |
   | stock_transfers|               | ledger    |
   | stock_adjusts  |               +-----------+
   +----------------+

   SALES & RETUR
   +----------------+   +-----------+   +-----------+
   | shifts         |-->| sales     |-->| sale_items|
   +----------------+   | (invoice) |   +-----------+
            |          +-----------+
            |                |
            |          +-----------+   +-----------+
            +--------->| payments |   | sale_refund|
            |          +-----------+   +-----------+
            |
   +----------------+   +-----------+   +-----------+
   | carts/cart_    |-->| returns   |-->| return_   |
   | items          |   |           |   | items     |
   +----------------+   +-----------+   +-----------+
                              |
                       +-----------------+
                       | return_approvals|
                       | refund_payments |
                       +-----------------+

   INVENTORY OPS
   +----------------+   +-----------+   +-----------+
   | vouchers       |   | promotions|   | expenses  |
   +----------------+   +-----------+   +-----------+
                                       | payrolls  |
                                       +-----------+

   KARYAWAN & INTEGRATION
   +----------------+   +-----------+   +-----------+
   | users          |-->| user_     |   | employees_|
   | (karyawan.md)  |   | outlets   |   | invitations|
   +----------------+   +-----------+   +-----------+
   | user_metrics_  |
   | cache          |
   +----------------+

   INTEGRATION & AUDIT
   +----------------+   +-----------+   +-----------+
   | integrations   |   | devices   |   | alerts    |
   |  (katalog +    |   +-----------+   +-----------+
   |   installs)    |
   +----------------+   +-----------+   +-----------+
   | integration_   |   | audit_logs|   | login_    |
   | logs, webhooks |   | (umum)    |   | audit     |
   +----------------+   +-----------+   +-----------+
```

## Standar Naming
- Tabel: `snake_case` jamak (`products`, `sale_items`, `stock_movements`)
- PK: `id` UUID
- FK: `<entity>_id` (`product_id`, `outlet_id`)
- Tenant: `tenant_id` di semua tabel bisnis
- Timestamp: `created_at`, `updated_at`, `deleted_at` (soft delete)
- Snapshot: suffix `_snapshot` (mis. `sku_snapshot`)
- Enum: `lowercase_snake` value
- Index: `unique (tenant_id, ...)`, `index (tenant_id, ...)` komposit

## Standar API
- Base path: `/api`
- Auth: `Authorization: Bearer <jwt>`
- Tenant scoping: otomatis dari JWT (`tenant_id`)
- Format response error: `{ error: { code, message, details? } }`
- Pagination: `?page=&pageSize=` (default 20, max 100), response `{ data: [], meta: { total, page, pageSize } }`
- Export: `Accept: text/csv` atau `?format=csv|pdf`
- Webhook: signed payload + `X-Signature` header (HMAC-SHA256)

## Catatan Implementasi
- FE saat ini masih pakai mock (`src/lib/mock*.ts`); backend harus isi kontrak yang sama agar drop-in
- Gunakan pola service layer (1 service per modul) + thin controller
- Validasi input di boundary (zod / class-validator) - jangan percaya FE
- Locking: `SELECT ... FOR UPDATE` di `stocks` & `payments` saat update
- Time zone: simpan UTC; render lokal di FE (Asia/Jakarta)
- Big number: `numeric(15,2)` untuk IDR, `numeric(15,3)` untuk qty/stok


# Modul: Transaksi (Riwayat Penjualan)

Route FE: `http://localhost:5173/riwayat-transaksi`
File FE: `src/pages/RiwayatTransaksiPage.tsx`, `src/components/TransactionHeader.tsx`, `src/components/TransactionFiltersBar.tsx`, `src/components/TransactionTable.tsx`, `src/components/TransactionKpi.tsx`, `src/components/TransactionDrawer.tsx`
Type/domain: `src/lib/transaction.types.ts`, hook: `src/lib/useTransactionList.ts`, data: `src/lib/mockTransactions.ts`

## Ringkasan Fitur
- List transaksi (invoice) lintas outlet
- Filter: rentang tanggal, toko, shift, kasir, metode bayar, status, pencarian bebas
- Status: `lunas | pending | refund | void`
- Metode bayar: `cash | qris | debit | credit | ewallet`
- Sync status: `synced | pending | failed`
- Flag: `refund | void | manual-discount | unsynced`
- Drawer detail: line items, diskon, voucher, pajak, refund, audit trail
- KPI: total trx, total revenue, avg order value, net profit (estimasi 32,7%)

## Skema Tabel Backend

### `sales` (header invoice)
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `outlet_id` UUID FK `outlets`
- `shift_id` UUID FK `shifts` (lihat `stok.md`)
- `cashier_user_id` UUID FK `users`
- `customer_id` UUID nullable FK `customers`
- `invoice_no` varchar unique per tenant (`INV-XXXX`)
- `date` timestamp
- `subtotal` numeric(15,2)
- `discount` numeric(15,2) default 0
- `voucher` numeric(15,2) default 0
- `tax` numeric(15,2) default 0
- `total` numeric(15,2)
- `payment_method` enum `('cash','qris','debit','credit','ewallet')`
- `status` enum `('lunas','pending','refund','void')` default `lunas`
- `sync_status` enum `('synced','pending','failed')` default `synced`
- `manual_discount` bool default false
- `notes` text nullable
- `created_by` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable

### `sale_items` (line item)
- `id` UUID PK
- `sale_id` UUID FK `sales`
- `product_id` UUID FK `products`
- `sku_snapshot` varchar
- `name_snapshot` varchar
- `qty` numeric(15,3)
- `price` numeric(15,2)
- `discount_pct` numeric(5,2) default 0
- `subtotal` numeric(15,2)
- `tenant_id` UUID

### `sale_refunds`
- `id` UUID PK
- `tenant_id` UUID
- `sale_id` UUID FK `sales`
- `amount` numeric(15,2)
- `reason` text
- `by_user_id` UUID FK `users`
- `at` timestamp

### `sale_audit`
- `id` UUID PK
- `tenant_id` UUID
- `sale_id` UUID FK `sales`
- `actor_user_id` UUID FK `users`
- `action` enum `('create','update','void','refund','sync','manual_discount')`
- `detail` text
- `kind` enum `('info','warning','danger','success')` default `info`
- `at` timestamp

## Field yang Diturunkan (dari `Transaction` FE)
| Field FE | Sumber di DB |
|---|---|
| `id`, `invoiceNo` | `sales.id`, `sales.invoice_no` |
| `date` | `sales.date` |
| `store`, `storeId` | `outlets.name`, `sales.outlet_id` |
| `shift` | `shifts.code` atau `sale_shifts.shift_code` snapshot |
| `cashier` | `users.name` |
| `customer` | `customers.name` (atau `Walk-in` jika null) |
| `items[]` | `sale_items` + `products` snapshot |
| `subtotal`, `discount`, `voucher`, `tax`, `total` | `sales.*` |
| `payment` | `sales.payment_method` |
| `status` | `sales.status` |
| `sync` | `sales.sync_status` |
| `flags[]` | diturunkan dari status/sync/manual_discount |
| `refund` | `sale_refunds` terakhir |
| `audit[]` | `sale_audit` order by `at desc` |
| `manualDiscount` | `sales.manual_discount` |

## Index Penting
- `unique (tenant_id, invoice_no)`
- `index (tenant_id, outlet_id, date desc)`
- `index (tenant_id, cashier_user_id, date desc)`
- `index (tenant_id, status, date desc)`
- `sale_items(tenant_id, product_id, sale_id)`
- `sale_audit(tenant_id, sale_id, at desc)`

## API Minimal
- `GET /api/sales?dateFrom=&dateTo=&storeId=&shift=&cashier=&payment=&status=&q=&page=&pageSize=`
- `GET /api/sales/:id` (ikut `items[], refunds[], audit[]`)
- `POST /api/sales` (create dari kasir)
- `PATCH /api/sales/:id` (limited: status, notes, payment)
- `POST /api/sales/:id/void` (audit + permission `sales.void`)
- `POST /api/sales/:id/refund` (body: `amount, reason`)
- `POST /api/sales/:id/sync` (status sync untuk POS offline)
- Export: `GET /api/sales/export?...` (CSV/PDF sesuai filter)

## Catatan SaaS
- `tenant_id` wajib + RLS
- Snapshot `sku_snapshot` & `name_snapshot` di `sale_items` agar histori stabil walau produk dihapus/rename
- Audit log wajib untuk void/refund/manual_discount
- Saat `status='void'`, kurangi dari revenue (service-layer atau filter di query)
- Trigger pengurangan `stocks.qty` saat `sales.status='lunas'` (lihat `stok.md`)

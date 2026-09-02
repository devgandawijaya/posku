# Modul: Kasir (POS Front)

Route FE: `http://localhost:5173/kasir-baru` (+ `/kasir` lama)
File FE: `src/pages/KasirBaruPage.tsx`, `src/components/KasirHeader.tsx`, `src/components/SearchScanBar.tsx`, `src/components/ProductGrid.tsx`, `src/components/CartPanel.tsx`, `src/components/HoldOrderDrawer.tsx`, `src/components/ShiftPanel.tsx`, `src/components/KbdHint.tsx`
Type/domain: `src/lib/kasir.types.ts` (Product, CartItem, Voucher, HoldOrder, ShiftInfo), hook: `src/lib/useKasir.ts`, data: `src/lib/mockKasir.ts`

## Ringkasan Fitur
- Quick POS: scan barcode / cari produk, add to cart, qty +/-, per-item discount
- Multi-payment: cash, qris, debit, credit, ewallet
- Voucher: percent / amount dengan `minSpend`
- Hold order (simpan & recall)
- Shift info (id, store, cashier, openedAt, invoiceNo)
- Keyboard shortcuts: F2 scan, F4 hold, F8 ganti payment, F9 bayar, Esc clear, +/- qty
- Promo per produk: `BOGO` / `Hemat X%`
- Status badge: aktif / low / out

## Skema Tabel Backend

### `shifts`
- `id` UUID PK
- `tenant_id` UUID
- `outlet_id` UUID FK `outlets`
- `cashier_user_id` UUID FK `users`
- `code` varchar (`S-1041`) unique per tenant per hari
- `opened_at` timestamp
- `closed_at` timestamp nullable
- `opening_cash` numeric(15,2)
- `cash_in` numeric(15,2) default 0
- `cash_out` numeric(15,2) default 0
- `expected_cash` numeric(15,2) default 0
- `actual_cash` numeric(15,2) nullable
- `status` enum `('open','closing','closed','overdue')` default `open`
- `notes` text nullable
- `created_at`, `updated_at`
- `index (tenant_id, outlet_id, opened_at desc)`

### `shift_cash_movements` (cash in/out di luar transaksi)
- `id` UUID PK
- `shift_id` UUID FK `shifts`
- `tenant_id` UUID
- `direction` enum `('in','out')`
- `amount` numeric(15,2)
- `reason` enum `('topup','withdraw','drop_safe','correction','expense','other')`
- `note` text nullable
- `created_by` UUID FK `users`
- `created_at` timestamp

### `carts` (keranjang aktif, opsional persisted)
- `id` UUID PK
- `tenant_id` UUID
- `shift_id` UUID FK `shifts`
- `cashier_user_id` UUID FK `users`
- `customer_id` UUID nullable FK `customers`
- `status` enum `('active','held','converted','abandoned')` default `active`
- `subtotal` numeric(15,2)
- `discount` numeric(15,2) default 0
- `tax` numeric(15,2) default 0
- `total` numeric(15,2)
- `voucher_code` varchar nullable
- `voucher_discount` numeric(15,2) default 0
- `payment_method` enum `('cash','qris','debit','credit','ewallet')` nullable
- `created_at`, `updated_at`

### `cart_items`
- `id` UUID PK
- `cart_id` UUID FK `carts`
- `product_id` UUID FK `products`
- `sku_snapshot` varchar
- `name_snapshot` varchar
- `price` numeric(15,2)
- `qty` numeric(15,3)
- `discount_pct` numeric(5,2) default 0
- `note` text nullable
- `tenant_id` UUID

### `vouchers` (katalog voucher)
- `id` UUID PK
- `tenant_id` UUID
- `code` varchar unique per tenant
- `label` varchar
- `type` enum `('percent','amount')`
- `value` numeric(15,2)
- `min_spend` numeric(15,2) default 0
- `max_discount` numeric(15,2) nullable (untuk percent)
- `usage_limit` int nullable
- `used_count` int default 0
- `valid_from` date
- `valid_until` date
- `is_active` bool default true
- `created_at`, `updated_at`

### `promotions` (promo per produk)
- `id` UUID PK
- `tenant_id` UUID
- `name` varchar
- `type` enum `('bogo','percent','amount')`
- `value` numeric(15,2) (percent atau amount; BOGO = qty bonus)
- `product_ids` UUID[] (bisa spesifik atau kategori)
- `category_id` UUID nullable (apply per kategori)
- `min_qty` int default 1
- `valid_from`, `valid_until` timestamp
- `is_active` bool default true

### `payments` (pencatatan payment per sale)
- `id` UUID PK
- `tenant_id` UUID
- `sale_id` UUID FK `sales`
- `shift_id` UUID FK `shifts`
- `method` enum `('cash','qris','debit','credit','ewallet')`
- `amount` numeric(15,2)
- `cash_received` numeric(15,2) nullable (khusus cash)
- `change_amount` numeric(15,2) nullable
- `ref_no` varchar nullable (payment gateway ref)
- `status` enum `('success','failed','pending','refunded')`
- `paid_at` timestamp
- `created_at`

## Field yang Diturunkan (FE)
| Field FE | Sumber di DB |
|---|---|
| `Product` (kasir) | `products` + `promotions` aktif -> `promo`; `sum(stocks.qty where outlet_id=...)` -> `stock` |
| `CartItem` | `cart_items` (active cart) |
| `Voucher` | `vouchers` where `code` & `is_active` & `valid_until>=now` |
| `HoldOrder` | `carts` where `status='held'` |
| `ShiftInfo` | `shifts` where `cashier_user_id=?` & `status='open'` |
| `PaymentMethod` (enum) | `payments.method` |

## Index Penting
- `shifts(tenant_id, outlet_id, status, opened_at desc)`
- `carts(tenant_id, shift_id, status)`
- `cart_items(tenant_id, cart_id)`
- `payments(tenant_id, sale_id)`
- `vouchers(tenant_id, code, is_active, valid_until)`

## API Minimal
- `POST /api/shifts/open` (body: `outletId, openingCash`)
- `POST /api/shifts/:id/close` (body: `actualCash, notes?`)
- `POST /api/shifts/:id/cash-movement` (in/out)
- `GET /api/shifts/active` (balik shiftInfo kasir yang sedang open)
- `GET /api/cashier/products?q=&category=&barcode=` (produk + harga + stok outlet + promo)
- `GET /api/cashier/vouchers?code=` (validasi)
- `POST /api/carts` (create / hold)
- `PATCH /api/carts/:id/items` (add/remove/update qty/discount)
- `POST /api/carts/:id/apply-voucher`
- `POST /api/carts/:id/checkout` (convert cart -> `sales` + `payments`, kurangi stok, return `{saleId, payment, change}`)
- `POST /api/sales/:id/print` (generate struk PDF/thermal)
- `GET /api/cashier/hold-orders?cashierId=`

## Catatan SaaS
- `tenant_id` wajib + RLS
- Saat checkout, atomic transaction: insert `sales` + `sale_items` + `payments`, kurangi `stocks.qty`, generate `stock_movements(-, reason='sale')`
- Shift harus `open` sebelum boleh checkout (validasi)
- Voucher `used_count++` di-increment dalam transaksi yang sama
- Idempotency-key untuk `checkout` (cegah duplikat saat retry offline)
- Offline mode: antrian `pending_sales` di FE, sinkron saat online -> `sales.sync_status='pending' -> 'synced'`

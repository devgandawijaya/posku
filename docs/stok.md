# Modul: Stok (Inventory)

Route FE: `http://localhost:5173/manage-stok` (+ transfer/restock/log)
File FE: `src/pages/ManageStokPage.tsx`, `src/components/ManageStokHeader.tsx`, `src/components/ManageStokFilters.tsx`, `src/components/ManageStokTable.tsx`, `src/components/ManageStokKpi.tsx`, `src/components/ManageStokCharts.tsx`
Type/domain: `src/lib/product.types.ts` (StockByStore, StockMovement, PriceHistory, SalesHistory), hook: `src/lib/useManageStok.ts`

## Ringkasan Fitur
- Daftar produk + stok per toko
- Filter: kategori, status (`normal | low | out`), search, toko
- Bulk action: transfer, aktifkan/nonaktifkan, hapus
- Min/Max rule (rule of thumb FE: `min = 20% stock`, `max = 180% stock`)
- Aksi per baris: restok, transfer, log
- KPI: total SKU, normal, low/menipis, habis, total nilai aset
- Chart: stok per cabang (bar), masuk vs keluar 14 hari (line)
- Status per produk: `Normal (>=10) | Hampir Habis (<10) | Habis (=0)`

## Skema Tabel Backend

### `stocks` (stok per outlet)
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `product_id` UUID FK `products`
- `outlet_id` UUID FK `outlets`
- `qty` numeric(15,3) default 0
- `reserved_qty` numeric(15,3) default 0 (untuk hold order)
- `min_stock` numeric(15,3) default 0 (override produk, default = `products.min_stock`)
- `max_stock` numeric(15,3) nullable
- `updated_by` UUID FK `users`
- `updated_at` timestamp
- unique `(tenant_id, product_id, outlet_id)`

### `stock_movements` (audit pergerakan stok)
- `id` UUID PK
- `tenant_id` UUID
- `product_id` UUID FK `products`
- `outlet_id` UUID FK `outlets`
- `delta` numeric(15,3) (signed: + masuk, - keluar)
- `reason` enum `('masuk','keluar','adjust','return','sale','transfer_in','transfer_out','restock','qc','damaged','supplier')`
- `ref_type` (`purchase`, `sale`, `return`, `transfer`, `adjustment`, `qc`, `po`, `manual`)
- `ref_id` UUID nullable
- `note` text nullable
- `created_by` UUID FK `users`
- `created_at` timestamp

### `stock_transfers`
- `id` UUID PK
- `tenant_id` UUID
- `from_outlet_id` UUID FK `outlets`
- `to_outlet_id` UUID FK `outlets`
- `status` enum `('draft','in_transit','received','cancelled')` default `draft`
- `note` text nullable
- `created_by`, `received_by` UUID FK `users`
- `created_at`, `received_at` timestamp

### `stock_transfer_items`
- `id` UUID PK
- `transfer_id` UUID FK `stock_transfers`
- `product_id` UUID FK `products`
- `qty` numeric(15,3)
- `received_qty` numeric(15,3) default 0
- `tenant_id` UUID

### `stock_adjustments` (form stock opname)
- `id` UUID PK
- `tenant_id` UUID
- `outlet_id` UUID FK `outlets`
- `status` enum `('draft','submitted','approved','rejected')`
- `note` text nullable
- `created_by`, `approved_by` UUID FK `users`
- `created_at`, `approved_at` timestamp

### `stock_adjustment_items`
- `id` UUID PK
- `adjustment_id` UUID FK `stock_adjustments`
- `product_id` UUID FK `products`
- `system_qty` numeric(15,3)
- `actual_qty` numeric(15,3)
- `delta` numeric(15,3) generated
- `note` text nullable
- `tenant_id` UUID

## Field yang Diturunkan (FE)
| Field FE | Sumber di DB |
|---|---|
| `Product.stock` (total) | `sum(stocks.qty)` per product |
| `StockByStore[]` | `stocks` join `outlets.name` |
| `StockMovement[]` | `stock_movements` order by created_at desc |
| Status `Habis/Menipis/Normal` | `qty == 0` / `qty < 10` / else |
| `min/max` | `min_stock`/`max_stock` (default hitung 20%/180%) |

## Index Penting
- `unique (tenant_id, product_id, outlet_id)` di `stocks`
- `index (tenant_id, outlet_id)` di `stocks`
- `index (tenant_id, product_id, created_at desc)` di `stock_movements`
- `stock_transfer_items(tenant_id, product_id)`

## API Minimal
- `GET /api/stocks?storeId=&category=&status=&q=&page=&pageSize=` (balik `Product + StockByStore[] + min/max`)
- `GET /api/stocks/:productId/movements?outletId=&from=&to=&limit=`
- `GET /api/stocks/:productId/by-store`
- `POST /api/stocks/adjust` (body: items, create `stock_adjustments` + `stock_movements` saat approved)
- `POST /api/stocks/transfer` (body: from/to + items, generate `stock_transfers` + 2 movements saat received)
- `POST /api/stocks/:productId/restock` (bulk: purchase order simple)
- Bulk: `POST /api/stocks/bulk-activate`, `POST /api/stocks/bulk-deactivate`, `POST /api/stocks/bulk-delete`

## Catatan SaaS
- `tenant_id` wajib + RLS
- Trigger/event: saat `sales.status='lunas'`, buat `stock_movements(-qty, reason='sale')` dan kurangi `stocks.qty`
- Saat `returns.fate='restock'`, buat `stock_movements(+qty, reason='return')`
- Transfer 2 langkah: `in_transit` -> `received` (saat received, catat 2 movement: `-` di source, `+` di target)
- Audit log untuk adjust/transfer
- Locking: gunakan `SELECT ... FOR UPDATE` di `stocks` saat update agar race-free

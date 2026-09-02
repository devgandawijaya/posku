# Modul: Laporan Stok

Route FE: `http://localhost:5173/laporan-stok`
File FE: `src/pages/LaporanStokPage.tsx`, `src/components/LaporanStokHeader.tsx`, `src/components/LaporanStokFilters.tsx`, `src/components/LaporanStokKpi.tsx`, `src/components/LaporanStokCharts.tsx`, `src/components/LaporanStokTable.tsx`
Type/domain: `src/lib/useLaporanStok.ts`

## Ringkasan Fitur
- Rekap stok global & per outlet
- KPI: total SKU, total qty, nilai aset, low/out
- Chart: stok per kategori, top produk menipis, pergerakan (masuk/keluar)
- Tabel: ringkasan per produk + status

## Skema Tabel Backend (read-only, sumber: `products`, `stocks`, `stock_movements`)

### View
#### `v_stock_summary`
- `tenant_id`, `product_id`, `sku`, `name`, `category_id`
- `total_qty` numeric
- `outlet_count` int
- `min_stock`, `max_stock`
- `status` enum derived `('normal','low','out')`
- `stock_value` numeric (= `total_qty * products.cost`)

#### `v_stock_movement_daily`
- `tenant_id`, `date`, `product_id`
- `masuk` numeric, `keluar` numeric
- sumber: `stock_movements` group by date, product

## API Minimal
- `GET /api/reports/stock/summary?storeId=&category=&status=`
- `GET /api/reports/stock/by-category`
- `GET /api/reports/stock/movement?dateFrom=&dateTo=&storeId=&productId=`
- `GET /api/reports/stock/top-low?limit=10`
- `GET /api/reports/stock/export?...`

## Catatan SaaS
- `tenant_id` wajib + RLS
- Materialized view refresh terjadwal
- Tabel besar (`stock_movements`): partition by `created_at` (monthly)

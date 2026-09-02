# Modul: Laporan Penjualan

Route FE: `http://localhost:5173/laporan-penjualan`
File FE: `src/pages/LaporanPenjualanPage.tsx`, `src/components/LaporanHeader.tsx`, `src/components/LaporanFilters.tsx`, `src/components/LaporanKpi.tsx`, `src/components/LaporanCharts.tsx`, `src/components/LaporanTable.tsx`
Type/domain: `src/lib/transaction.types.ts`, hook: `src/lib/useLaporan.ts`

## Ringkasan Fitur
- Filter: rentang tanggal (today/7d/30d/mtd/custom), toko, metode bayar, status, search
- KPI: total revenue, jumlah order, AOV, net profit (estimasi 32,7%), rata-rata per toko
- Chart: line omzet harian, donut by payment, grouped by store
- Tabel agregat per toko (revenue, trx)
- Exclude `status='void'` dari revenue (di query layer)

## Skema Tabel Backend (read-only, sumber: `sales` + `sale_items`)

### View / Tabel Agregat
#### `v_sales_daily` (view)
- `tenant_id`, `outlet_id`, `date`
- `orders` int
- `revenue` numeric(18,2)
- `avg_order` numeric(18,2)
- Disusun dari `sales` where `status<>'void'`

#### `v_sales_by_payment` (view)
- `tenant_id`, `outlet_id`, `date`, `payment_method`
- `amount` numeric, `count` int

#### `v_sales_by_store` (view)
- `tenant_id`, `outlet_id`, `name`
- `revenue`, `orders`

## API Minimal
- `GET /api/reports/sales/summary?dateFrom=&dateTo=&storeId=&payment=&status=`
  - Balik: `total, orders, aov, net, avgPerStore, byStoreMap`
- `GET /api/reports/sales/timeseries?dateFrom=&dateTo=&storeId=`
  - Balik: `[{date, omzet}]`
- `GET /api/reports/sales/by-payment?dateFrom=&dateTo=&storeId=`
- `GET /api/reports/sales/by-store?dateFrom=&dateTo=&storeId=`
- `GET /api/reports/sales/top-products?dateFrom=&dateTo=&storeId=&limit=10`
- `GET /api/reports/sales/export?...` (CSV/PDF)

## Catatan SaaS
- `tenant_id` wajib + RLS
- Materialized view di-refresh tiap 5-15 menit (jadwal), atau on-demand untuk plan tertentu
- Field `net` adalah estimasi margin (32,7% di mock); production: ambil dari `products.cost` & HPP riil

# Modul: Laporan Keuangan

Route FE: `http://localhost:5173/laporan-keuangan`
File FE: `src/pages/LaporanKeuanganPage.tsx`, `src/components/LaporanKeuanganHeader.tsx`, `src/components/LaporanKeuanganFilters.tsx`, `src/components/LaporanKeuanganKpi.tsx`, `src/components/LaporanKeuanganCharts.tsx`, `src/components/LaporanKeuanganTable.tsx`
Type/domain: `src/lib/useLaporanKeuangan.ts`

## Ringkasan Fitur
- P&L (laba rugi) per outlet & agregat tenant
- Pendapatan: omzet penjualan
- HPP (cogs): 60% pendapatan (rule mock)
- Beban operasional: gaji, sewa outlet, pemasaran
- Laba kotor, laba bersih, net margin
- Cash flow in/out
- Expense breakdown by category

## Skema Tabel Backend (read + write operasional expenses)

### `expenses`
- `id` UUID PK
- `tenant_id` UUID
- `outlet_id` UUID nullable FK `outlets` (null = company level)
- `category` enum `('gaji','sewa','pemasaran','listrik','air','internet','perlengkapan','transport','lainnya')`
- `amount` numeric(15,2)
- `date` date
- `note` text nullable
- `ref_type` (`manual`, `payroll`, `invoice`)
- `ref_id` UUID nullable
- `created_by` UUID FK `users`
- `created_at`, `updated_at`

### `payrolls`
- `id` UUID PK
- `tenant_id` UUID
- `outlet_id` UUID FK `outlets`
- `user_id` UUID FK `users`
- `period` varchar (`2026-08`)
- `base_salary` numeric(15,2)
- `allowance` numeric(15,2) default 0
- `deduction` numeric(15,2) default 0
- `net` numeric(15,2)
- `status` enum `('draft','paid','cancelled')`
- `paid_at` timestamp nullable
- `created_by` UUID FK `users`
- `created_at`

### `rent_contracts` (opsional, untuk beban sewa)
- `id` UUID PK
- `tenant_id` UUID
- `outlet_id` UUID FK `outlets`
- `monthly_rent` numeric(15,2)
- `start_date` date, `end_date` date nullable
- `status` enum `('aktif','nonaktif')`

### View
#### `v_pl_summary`
- `tenant_id`, `outlet_id`, `date`
- `pendapatan` numeric (sum sales.total where status<>'void')
- `cogs` numeric (60% default, atau join `sale_items` ke `products.cost`)
- `opex` numeric (sum expenses)
- `laba_kotor` numeric (= pendapatan - cogs)
- `laba_bersih` numeric (= laba_kotor - opex)
- `net_margin` numeric

#### `v_cashflow_daily`
- `tenant_id`, `outlet_id`, `date`
- `cash_in` numeric (sales payment cash + refund in)
- `cash_out` numeric (expenses + refund out + transfer out)

## API Minimal
- `GET /api/reports/finance/pl?dateFrom=&dateTo=&storeId=`
- `GET /api/reports/finance/cashflow?dateFrom=&dateTo=&storeId=`
- `GET /api/reports/finance/expense-breakdown?dateFrom=&dateTo=&storeId=`
- `GET /api/reports/finance/by-store?dateFrom=&dateTo=`
- `GET /api/reports/finance/export?...`
- CRUD `expenses`: `GET/POST/PATCH/DELETE /api/expenses`
- CRUD `payrolls`: `GET/POST/PATCH/DELETE /api/payrolls`

## Catatan SaaS
- `tenant_id` wajib + RLS
- HPP default 60% di mock; production harus ambil `products.cost` riil per sale_items
- Materialized view refresh terjadwal; expense & payroll di-cache nightly

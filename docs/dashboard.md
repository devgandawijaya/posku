# Modul: Dashboard (Monitoring SaaS Multi-Tenant)

Route FE: `http://localhost:5173/dashboard`
File FE: `src/pages/DashboardPage.tsx`, `src/components/KpiRow.tsx`, `src/components/DashboardFilters.tsx`, `src/components/AlertsCenter.tsx`, `src/components/ShiftPanel.tsx`, `src/components/TenantStoreMonitor.tsx`, `src/components/PaymentMixList.tsx`, `src/components/IntGrid.tsx` (re-use), `src/components/SummaryCard.tsx`
Type/domain: `src/lib/dashboard.types.ts` (AlertItem, Shift, TenantStore, KpiProfit, PaymentMix, InventoryHealth, SaasSub), hook: `src/lib/useDashboardData.ts`, data: `src/lib/mockDashboard.ts`

## Ringkasan Fitur
- KPI utama: total revenue, orders, avg ticket, cash float
- Profit breakdown: gross, COGS, opex, net, margin %, delta
- Inventory health: total SKU, low/out, stock value, turnover days
- Payment mix: per metode bayar
- Alerts: shift, stock, device, sync (severity: critical/warning/info)
- Shift panel: list shift aktif, opening cash, cash in/out, expected/actual, status
- Tenant & store monitor: per toko (status, devices, open shifts, revenue today, health %)
- SaaS subscription: MRR, ARR, churn, renewals, trial expiring, plan tier
- Filter: tenant, store, cashier, shift, payment, range (today/7d/30d/mtd/custom)

## Skema Tabel Backend

### `tenants`
- `id` UUID PK
- `name` varchar
- `slug` varchar unique
- `plan_id` UUID FK `plans` (lihat SaaS billing)
- `status` enum `('active','suspended','trial','churned')` default `trial`
- `trial_ends_at` timestamp nullable
- `owner_user_id` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable

### `devices` (POS device per outlet)
- `id` UUID PK
- `tenant_id` UUID
- `outlet_id` UUID FK `outlets`
- `name` varchar (e.g. `Tablet kasir #07`)
- `type` enum `('tablet','printer','scanner','display','cash_drawer','kitchen_display')`
- `serial_no` varchar nullable
- `status` enum `('online','offline','degraded')`
- `last_heartbeat_at` timestamp nullable
- `firmware_version` varchar nullable
- `created_at`, `updated_at`

### `alerts` (alert center, semua jenis)
- `id` UUID PK
- `tenant_id` UUID
- `outlet_id` UUID nullable
- `kind` enum `('shift','stock','device','sync','security','billing')`
- `severity` enum `('critical','warning','info')`
- `title` varchar
- `detail` text
- `entity_type` varchar (`shift`, `product`, `device`, `sale`)
- `entity_id` UUID nullable
- `status` enum `('open','acknowledged','resolved')` default `open`
- `acknowledged_by` UUID FK `users` nullable
- `acknowledged_at` timestamp nullable
- `created_at` timestamp
- `index (tenant_id, status, created_at desc)`

### `plan_health_daily` (cache agregat untuk dashboard, dihitung harian)
- `tenant_id` UUID
- `date` date
- `revenue` numeric(18,2)
- `orders` int
- `cogs` numeric(18,2)
- `opex` numeric(18,2)
- `net` numeric(18,2)
- `inventory_value` numeric(18,2)
- `turnover_days` numeric(6,2)
- `low_stock` int
- `out_stock` int
- PK `(tenant_id, date)`

### `payment_mix_cache` (harian)
- `tenant_id` UUID
- `outlet_id` UUID
- `date` date
- `method` enum `('cash','qris','debit','credit','ewallet')`
- `amount` numeric
- `count` int
- PK `(tenant_id, outlet_id, date, method)`

### `saas_subscription_metrics` (snapshot periodik)
- `id` UUID PK
- `plan` enum `('Starter','Pro','Enterprise')`
- `tenants` int
- `stores` int
- `mrr` numeric(18,2)
- `arr` numeric(18,2)
- `churn` numeric(5,2) (%)
- `renewals` int
- `trial_expiring` int
- `snapshot_at` timestamp

## Field yang Diturunkan (FE)
| Field FE | Sumber di DB |
|---|---|
| `KpiProfit` | `plan_health_daily` latest (gross=cogs+net, cogs, opex, net, margin, delta) |
| `PaymentMix[]` | `payment_mix_cache` (filter by date range) |
| `InventoryHealth` | `plan_health_daily` (totalSku, lowStock, outOfStock, stockValue, turnoverDays) |
| `SaasSub` | `saas_subscription_metrics` latest + `tenants` count |
| `Shift` (panel) | `shifts` where `status in ('open','closing','overdue')` |
| `AlertItem` | `alerts` where `status='open'` order by `severity, created_at desc` |
| `TenantStore` | agregat `devices` (count by status), `shifts` (open count), `revenueToday` |
| `DashboardFilters` | persisted di `dashboard_layouts` (opsional) per user |

## Index Penting
- `devices(tenant_id, outlet_id, status)`
- `alerts(tenant_id, status, severity, created_at desc)`
- `plan_health_daily(tenant_id, date desc)`
- `payment_mix_cache(tenant_id, date desc)`
- partition `plan_health_daily` & `payment_mix_cache` by month

## API Minimal
- `GET /api/dashboard/summary?tenantId=&storeId=&cashier=&shift=&payment=&range=`
  - Balik: `totals, profit, payments, inventory, saas`
- `GET /api/dashboard/alerts?severity=&kind=&status=`
- `POST /api/alerts/:id/acknowledge`
- `POST /api/alerts/:id/resolve`
- `GET /api/dashboard/tenant-stores?tenantId=` (multi-tenant overview)
- `GET /api/dashboard/shifts/active?storeId=`
- `GET /api/dashboard/saas-metrics` (khusus owner/superadmin)
- `GET /api/dashboard/devices?storeId=&status=`

## Catatan SaaS
- `tenant_id` wajib + RLS
- `saas_subscription_metrics` dihitung nightly job (cron)
- `alerts` di-generate via rule engine (event-driven): shift overdue, stock <= min, device offline > 5min, sync pending > N trx
- Untuk superadmin (internal posku): `tenantId` filter diabaikan
- Heavy aggregations: materialized view + cache invalidation saat underlying data berubah

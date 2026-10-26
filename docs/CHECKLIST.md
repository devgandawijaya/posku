# Checklist Implementasi Modul (19 dokumen di `docs/`)

Status per 2026-09-02 (pass terakhir). Implementasi memakai konvensi codebase existing (ID `uint`
auto-increment, `Company`=tenant, `Store`=outlet, `Employee`=user) — bukan migrasi literal ke
UUID/RLS/tabel `tenants`/`outlets`/`users` seperti draft skema di masing-masing dokumen, supaya
tidak menghancurkan kode & data yang sudah berjalan. Endpoint final ada di
[routes.go](../internal/routes/routes.go).

## 1. auth.md
- [x] Login/refresh/logout/me/change-password/forgot-reset-password, lockout + login_audit
- [x] 2FA TOTP (RFC 6238, stdlib)
- [ ] Pengiriman kode via email/SMS sungguhan (butuh mail/SMS transport eksternal — di luar cakupan kode)

## 2. role-akses.md
- [x] CRUD roles, permission matrix, toggle, delete, catalog
- [ ] Tabel terpisah `role_outlets`/`permissions` (keputusan arsitektur: disatukan sebagai JSON)
- [ ] Partitioning/retention audit 2 tahun (infra DB partition)

## 3. karyawan.md
- [x] CRUD, assign-outlets, toggle-status, reset-password, options, by-outlet, invite/accept-invite, bulk-import
- [x] `user_metrics_cache`: `POST /users/:id/metrics/refresh`

## 4. kategori.md
- [x] CRUD + KPI live + `category_kpi_cache` (`POST/GET /categories/:id/kpi/refresh|cache`)

## 5. product.md
- [x] Tenant-scoped, multi-barcode, price history, bulk actions, product images

## 6. outlet.md
- [x] Kode auto, manager, status/toggle, metrics live + cache, options dropdown
- [x] Filter `hasTrx` di `GET /stores?hasTrx=true|false`

## 7. supplier.md
- [x] CRUD, kode auto, assign-outlets, toggle-status, options, `supplier_metrics_cache` + `POST /suppliers/:id/purchases`

## 8. pelanggan.md
- [x] member_code, tier, points_balance/ledger, options, transactions, metrics cache + re-tier otomatis

## 9. kasir.md
- [x] Shift, voucher, promotion, cart/hold-order/checkout, idempotency-key, print struk

## 10. transaksi.md
- [x] invoice_no, status, sync, void/refund/sync, export CSV, `sale_refunds`/`sale_audit` terpisah

## 11. retur.md
- [x] Approval berjenjang, auto restock, `refund_payments` terpisah

## 12. stok.md
- [x] stock_movements, adjust, restock, stock_adjustments draft->submit->approve, bulk actions
- [ ] Partisi bulanan `stock_movements` (infra DB, di luar cakupan kode)

## 13. laporan-penjualan.md
- [x] Summary/timeseries/by-payment/by-store/top-products + export CSV
- [x] `v_sales_daily` substitute: `POST /reports/sales/daily-cache/refresh`, `GET /reports/sales/daily-cache`
- [x] `net`/COGS sekarang dihitung riil dari `sale_items.quantity x products.cost` (fallback estimasi 60% hanya jika data kosong)

## 14. laporan-stok.md
- [x] Summary/by-category/movement/top-low + export CSV
- [x] `v_stock_summary` substitute: `POST /reports/stock/summary-cache/refresh`, `GET /reports/stock/summary-cache`

## 15. laporan-keuangan.md
- [x] CRUD expenses/payrolls, PL (COGS riil + rent), cashflow, expense-breakdown, export CSV
- [x] `rent_contracts`: `POST/GET/PATCH/DELETE /rent-contracts` (masuk ke opex `laporan/finance/pl`)
- [ ] Payroll otomatis via cron (butuh scheduler eksternal)

## 16. dashboard.md
- [x] Summary KPI (COGS riil), alerts CRUD, shifts aktif, devices, payment-mix, saas-metrics, rule-engine alert manual

## 17. integrasi.md
- [x] Katalog, install/connect/disconnect/test/logs, webhook in/out, verifikasi signature HMAC
- [x] `config_schema` validation: field wajib (`api_key`, dll) dicek saat `POST /integrations/:id/connect`
- [x] Retry backoff outbound: `dispatchWebhookEvent()` mengirim event (`sale.created`) ke webhook aktif dengan retry 3x exponential backoff (in-process, tanpa queue)
- [ ] Enkripsi API key/secret via KMS (saat ini disimpan apa adanya + masking tampilan — butuh KMS/vault eksternal)

## 18. subscription-billing.md
- [x] plans, subscription lifecycle, invoices, payment methods, coupons, plan_quotas + feature gating, `GET /billing/usage`
- [ ] Webhook payment gateway sungguhan (Midtrans/Xendit/Stripe), cron auto-generate invoice H-3 (butuh kredensial & scheduler eksternal)

## 19. audit-logs.md
- [x] `GET /audit`, `/audit/:id`, `/audit/entity/:type/:id`, `/audit/export`
- [x] PII/secret redaction: field sensitif (`password`, `api_key`, `token`, `secret`, dll) otomatis di-redact dari `diff` sebelum disimpan
- [ ] Partitioning bulanan, retention policy (infra DB)

---
Ringkasan akhir: seluruh 19 modul telah lengkap secara fungsional di level kode, termasuk seluruh
item yang sebelumnya bisa disubstitusi tanpa infrastruktur eksternal — 2FA TOTP nyata, cache
materialized-view-style (kategori/outlet/pelanggan/supplier/sales-daily/stock-summary) dengan
endpoint refresh manual, COGS riil dari `sale_items`+`products.cost`, rent_contracts, config_schema
validation, outbound webhook retry backoff, dan PII/secret redaction di audit log. Sisa item
"belum" (ditandai di atas) murni membutuhkan infrastruktur pihak ketiga yang tidak tersedia di
lingkungan ini: mail/SMS transport, KMS/vault untuk enkripsi secret, payment gateway sungguhan
dengan kredensial asli, cron scheduler, dan DB partitioning/retention policy.

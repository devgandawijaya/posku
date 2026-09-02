# Modul: Subscription & Billing (SaaS)

Sumber ringkasan: `docs/product.md` (bagian Subscription / Billing) + `docs/dashboard.md` (`SaasSub`). Tidak ada halaman FE dedicated untuk owner tenant (saat ini hanya panel internal posku). Akan muncul di halaman `Billing` (akan dibuat) atau di Superadmin panel.

## Ringkasan Fitur
- Paket bertingkat: `free | pro | enterprise` (atau `Starter | Pro | Enterprise`)
- Tenant subscribe ke 1 plan, dengan periode tagihan (monthly/yearly)
- Invoice otomatis per periode
- Payment via gateway (Midtrans/Xendit/Stripe) - sudah ada provider di `integrations`
- KPI SaaS: MRR, ARR, churn %, renewals, trial expiring
- Feature gating per plan (mis. multi-outlet hanya di Pro+)

## Skema Tabel Backend

### `plans` (katalog global)
- `id` UUID PK
- `code` varchar unique (`free`, `pro`, `enterprise`, atau `starter`, `pro`, `enterprise`)
- `name` varchar
- `description` text
- `price_monthly` numeric(15,2)
- `price_yearly` numeric(15,2)
- `currency` varchar(3) default `IDR`
- `max_outlets` int nullable (null = unlimited)
- `max_users` int nullable
- `max_products` int nullable
- `max_transactions_per_month` int nullable
- `features` JSONB (feature flags: `{multi_outlet: true, custom_domain: false, ...}`)
- `is_active` bool default true
- `sort_order` int default 0
- `created_at`, `updated_at`

### `subscriptions`
- `id` UUID PK
- `tenant_id` UUID FK `tenants` unique (1 tenant = 1 active subscription)
- `plan_id` UUID FK `plans`
- `status` enum `('trial','active','past_due','cancelled','expired')` default `trial`
- `billing_cycle` enum `('monthly','yearly')` default `monthly`
- `current_period_start` timestamp
- `current_period_end` timestamp
- `trial_ends_at` timestamp nullable
- `cancelled_at` timestamp nullable
- `cancel_at_period_end` bool default false
- `payment_method_id` UUID nullable FK `tenant_payment_methods`
- `created_at`, `updated_at`
- `index (status, current_period_end)`

### `tenant_payment_methods` (metode bayar tenant)
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `type` enum `('virtual_account','credit_card','qris','ewallet','manual_transfer')`
- `provider` varchar (`midtrans`, `xendit`, `stripe`, `manual`)
- `external_id` varchar (token/id dari provider)
- `masked_info` varchar (mis. `VISA **** 1234`)
- `is_default` bool default false
- `expires_at` date nullable
- `created_at`, `updated_at`

### `invoices`
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `subscription_id` UUID FK `subscriptions`
- `invoice_no` varchar unique (`INV-2026-08-00001`)
- `period_start` date
- `period_end` date
- `subtotal` numeric(15,2)
- `tax` numeric(15,2) default 0
- `discount` numeric(15,2) default 0
- `total` numeric(15,2)
- `currency` varchar(3) default `IDR`
- `status` enum `('draft','open','paid','past_due','void','uncollectible')` default `draft`
- `due_date` date
- `paid_at` timestamp nullable
- `notes` text nullable
- `created_at`, `updated_at`
- `index (tenant_id, status, due_date)`

### `invoice_items`
- `id` UUID PK
- `invoice_id` UUID FK `invoices`
- `description` varchar (mis. `Pro plan - Aug 2026`)
- `quantity` int default 1
- `unit_price` numeric(15,2)
- `amount` numeric(15,2)
- `period_start` date
- `period_end` date
- `proration` bool default false
- `tenant_id` UUID

### `payments`
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `invoice_id` UUID FK `invoices`
- `payment_method_id` UUID nullable FK `tenant_payment_methods`
- `method` enum `('virtual_account','credit_card','qris','ewallet','manual_transfer')`
- `amount` numeric(15,2)
- `currency` varchar(3) default `IDR`
- `external_ref` varchar (ref dari payment gateway)
- `status` enum `('pending','success','failed','refunded','expired')`
- `paid_at` timestamp nullable
- `raw_response` JSONB nullable (response payment gateway)
- `created_at`, `updated_at`
- `index (tenant_id, status, paid_at desc)`

### `coupons` (diskon untuk subscription)
- `id` UUID PK
- `code` varchar unique
- `type` enum `('percent','amount')`
- `value` numeric(15,2)
- `max_redemptions` int nullable
- `redeemed_count` int default 0
- `valid_from`, `valid_until` timestamp
- `applies_to_plans` UUID[] (restrict ke plan id tertentu; empty = semua)
- `is_active` bool default true
- `created_at`

### `subscription_events` (audit subscription)
- `id` UUID PK
- `tenant_id` UUID
- `subscription_id` UUID FK `subscriptions`
- `event` enum `('created','trial_started','trial_ended','activated','renewed','upgraded','downgraded','cancelled','reactivated','payment_failed','payment_succeeded','expired')`
- `from_plan_id` UUID nullable
- `to_plan_id` UUID nullable
- `actor_user_id` UUID nullable FK `users` (null = system)
- `at` timestamp

### `plan_quotas` (counter penggunaan tenant, reset periodik)
- `tenant_id` UUID PK
- `period` varchar (`2026-08`)
- `transactions_count` int default 0
- `api_calls_count` int default 0
- `storage_bytes` bigint default 0
- `updated_at` timestamp

## View / Materialized

### `v_saas_mrr` (untuk dashboard `SaasSub`)
- `plan`, `active_subscriptions`, `mrr`, `arr`
- dihitung dari `subscriptions` where `status in ('active','trial')` x `plans.price_monthly`

### `v_churn_cohort`
- cohort per bulan signup, hitung retention

## Index Penting
- `subscriptions(tenant_id, status, current_period_end)`
- `invoices(tenant_id, status, due_date desc)`
- `payments(tenant_id, status, paid_at desc)`
- partition `subscription_events` & `payments` by month

## API Minimal
- `GET /api/billing/plans` (katalog publik)
- `GET /api/billing/subscription` (current)
- `POST /api/billing/subscription` (subscribe: body `{planId, billingCycle, paymentMethodId?}`)
- `POST /api/billing/subscription/change-plan` (upgrade/downgrade)
- `POST /api/billing/subscription/cancel` (body `{atPeriodEnd: bool}`)
- `GET /api/billing/invoices?status=&from=&to=`
- `GET /api/billing/invoices/:id`
- `POST /api/billing/invoices/:id/pay` (initiate payment)
- `POST /api/webhooks/in/:provider` (payment callback - Midtrans/Xendit/Stripe)
- `GET /api/billing/payment-methods`
- `POST /api/billing/payment-methods`
- `DELETE /api/billing/payment-methods/:id`
- `GET /api/billing/usage` (balik counter dari `plan_quotas` vs limit plan)
- `GET /api/admin/saas/metrics` (khusus superadmin posku: MRR/ARR/churn)

## Feature Gating (Service Layer)
- Cek `subscriptions.plan.features` + `plans.max_*` sebelum enable fitur di service:
  - `multi_outlet` -> cek `tenants.outlets.length <= plan.max_outlets`
  - `max_products` -> cek `products.count <= plan.max_products`
  - `max_users` -> cek `users.count <= plan.max_users`
  - `max_transactions_per_month` -> reset di `plan_quotas`
- Saat over quota: return 402 Payment Required / 403 Forbidden dengan pesan spesifik

## Catatan SaaS
- `tenant_id` wajib + RLS
- Invoice auto-generate H-3 sebelum `current_period_end` (cron)
- Payment webhook harus **idempotent**: cek `payments.external_ref` sebelum insert
- Subscription status auto-transition: `trial -> past_due` saat `trial_ends_at` lewat tanpa payment; `active -> past_due` jika invoice unpaid > X hari; `past_due -> cancelled` setelah grace period
- Tax: PPN 11% (Indonesia) - simpan di `invoices.tax`
- Currency multi: simpan `currency` per invoice; konversi saat reporting
- Audit log untuk semua perubahan subscription (lihat `subscription_events`)

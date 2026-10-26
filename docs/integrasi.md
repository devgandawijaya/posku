# Modul: Integrasi

Route FE: `http://localhost:5173/integrasi`
File FE: `src/pages/IntegrasiPage.tsx`, `src/components/IntHeader.tsx`, `src/components/IntGrid.tsx`, `src/components/IntKpi.tsx`
Type/domain: `src/lib/integrasi.types.ts`, hook: `src/lib/useIntegrasi.ts`, data: `src/lib/mockIntegrasi.ts`

## Ringkasan Fitur
- 8 kategori integrasi: payment, delivery, notification, accounting, marketplace, ecommerce, pos, api
- Scope: `company` (tenant-wide) atau `store` (per outlet, `storeIds[]`)
- Status: `connected | disconnected | error | pending`
- Metadata key-value per integrasi (mis. `API: v3`, `Pesanan/hr: 24`, `Error: invalid key`)
- API key masked (`****_m9n0`)
- Webhook URL
- Last sync timestamp
- Aksi: connect, disconnect, configure, test connection, delete, export
- Hak akses: `connect/disconnect/configure` = owner/admin, `delete` = owner, `export` = owner/admin/manager

## Skema Tabel Backend

### `integrations` (katalog provider - shared)
- `id` varchar PK (e.g. `midtrans`, `xendit`, `shopee`)
- `provider` varchar (nama perusahaan provider)
- `category` enum `('payment','delivery','notification','accounting','marketplace','ecommerce','pos','api')`
- `display_name` varchar
- `description` text
- `icon` varchar (FA class)
- `brand_color` varchar(7) (hex)
- `docs_url` text nullable
- `config_schema` JSONB (definisi field konfigurasi per provider)
- `created_at` timestamp

### `integration_installations` (install per tenant)
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `integration_id` varchar FK `integrations.id`
- `scope` enum `('company','store')`
- `status` enum `('connected','disconnected','error','pending')` default `disconnected`
- `api_key_encrypted` text nullable (encrypted at rest)
- `api_key_masked` varchar nullable (untuk display)
- `webhook_url` text nullable
- `webhook_secret_encrypted` text nullable
- `last_sync_at` timestamp nullable
- `installed_at` timestamp nullable
- `installed_by` UUID FK `users`
- `error_message` text nullable
- `created_at`, `updated_at`
- `unique (tenant_id, integration_id)`

### `integration_outlets` (assign store-level)
- `installation_id` UUID FK `integration_installations`
- `outlet_id` UUID FK `outlets`
- PK komposit `(installation_id, outlet_id)`
- `tenant_id` UUID

### `integration_meta` (key-value dinamis)
- `id` UUID PK
- `installation_id` UUID FK `integration_installations`
- `key` varchar
- `value` text
- `sensitive` bool default false
- `tenant_id` UUID
- `unique (installation_id, key)`

### `integration_logs` (audit/sync events)
- `id` UUID PK
- `tenant_id` UUID
- `installation_id` UUID FK `integration_installations`
- `event` enum `('connect','disconnect','sync','error','webhook_in','webhook_out','test')`
- `level` enum `('info','warning','error')`
- `message` text
- `payload` JSONB nullable
- `latency_ms` int nullable
- `at` timestamp

### `webhooks` (outbound webhook endpoints)
- `id` UUID PK
- `tenant_id` UUID
- `url` text
- `secret_encrypted` text
- `events` text[] (e.g. `['sale.created','stock.updated']`)
- `is_active` bool default true
- `created_by` UUID FK `users`
- `created_at`, `updated_at`

## Field yang Diturunkan (FE `Integration`)
| Field FE | Sumber di DB |
|---|---|
| `id` | `integration_installations.id` |
| `companyId` | `integration_installations.tenant_id` |
| `provider` | `integrations.provider` |
| `category` | `integrations.category` |
| `displayName` | `integrations.display_name` |
| `description` | `integrations.description` |
| `icon`, `brandColor` | `integrations.icon/brand_color` |
| `status` | `integration_installations.status` |
| `scope` | `integration_installations.scope` |
| `storeIds` | `integration_outlets.outlet_id[]` |
| `apiKeyMasked` | `integration_installations.api_key_masked` |
| `webhookUrl` | `integration_installations.webhook_url` |
| `lastSyncAt` | `integration_installations.last_sync_at` |
| `installedAt` | `integration_installations.installed_at` |
| `meta[]` | `integration_meta` -> `[{key, value, sensitive}]` |
| `docsUrl` | `integrations.docs_url` |

## Index Penting
- `unique (tenant_id, integration_id)` di `integration_installations`
- `integration_installations(tenant_id, status)`
- `integration_logs(tenant_id, installation_id, at desc)`
- partition `integration_logs` by month untuk volume tinggi

## API Minimal
- `GET /api/integrations?q=&status=&category=&scope=`
- `GET /api/integrations/catalog` (semua provider di katalog)
- `GET /api/integrations/:id` (ikut meta + lastSync + errorMessage)
- `POST /api/integrations/:id/connect` (body: config per `config_schema`)
- `POST /api/integrations/:id/disconnect`
- `PATCH /api/integrations/:id` (update meta, webhook, scope, outlets)
- `POST /api/integrations/:id/scope` (body: `scope, storeIds[]`)
- `POST /api/integrations/:id/test` (balik `{ok, latencyMs, message}`)
- `DELETE /api/integrations/:id` (owner only)
- `GET /api/integrations/:id/logs?from=&to=&level=`
- `GET /api/integrations/export` (CSV konfigurasi)

## Webhook In/Out
- Inbound: `POST /api/webhooks/in/:provider` (validasi signature, dispatch ke `integration_logs` + handler per event)
- Outbound: `POST /api/webhooks/out` (signed; retries exponential backoff)
- Retensi: simpan `payload` 90 hari (sesuai compliance)

## Catatan SaaS
- `tenant_id` wajib + RLS
- `api_key_encrypted`/`webhook_secret_encrypted` pakai KMS (AWS KMS / Vault)
- `api_key_masked` = `****_last4` untuk display
- Rate limit + circuit breaker per provider (hindari overload downstream)
- Audit log untuk connect/disconnect/configure/delete
- Secret rotation terjadwal (per 90 hari, configurable)

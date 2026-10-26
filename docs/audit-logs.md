# Modul: Audit Logs (Append-Only)

Digunakan lintas modul: `products`, `stocks`, `sales`, `returns`, `roles`, `users`, `integrations`, `billing`, dll. Tidak punya halaman FE khusus; dipanggil via drawer detail (contoh: `TransactionDrawer` -> `Taudit[]`).

## Skema Tabel Backend

### `audit_logs` (umum, lintas entitas)
- `id` UUID PK
- `tenant_id` UUID FK `tenants` (nullable untuk superadmin/owner posku)
- `actor_user_id` UUID nullable FK `users`
- `actor_name` varchar (snapshot)
- `actor_role` varchar (snapshot)
- `actor_ip` inet nullable
- `actor_user_agent` text nullable
- `entity_type` varchar (`product`, `stock`, `sale`, `return`, `role`, `user`, `integration`, `outlet`, `supplier`, `customer`, `category`, `shift`, `expense`, `payroll`, `promotion`, `voucher`, `billing`)
- `entity_id` UUID
- `entity_label` varchar (snapshot nama/identifier)
- `action` enum generic: `('create','update','delete','view','export','print','login','logout','approve','reject','void','refund','assign','unassign','toggle','sync','connect','disconnect','configure','pay','refund_payment','open_shift','close_shift','stock_adjust','transfer_send','transfer_receive','checkout')`
- `diff` JSONB nullable (`{before: {...}, after: {...}}`)
- `severity` enum `('info','warning','danger','success')` default `info`
- `at` timestamp default now()
- partition by month

### `role_audit` (khusus role/permission - lihat `role-akses.md`)
- Pemisahan agar filter & retensi beda (compliance lebih ketat, 2 tahun)

### `login_audit` (lihat `auth.md`)
- Pemisahan karena volume tinggi + PII

### `integration_logs` (lihat `integrasi.md`)
- Event sync/webhook/error per provider

### `sale_audit` (lihat `transaksi.md`)
- Audit spesifik per invoice

## Index Penting
- `audit_logs(tenant_id, entity_type, entity_id, at desc)` - lookup by entity
- `audit_logs(tenant_id, actor_user_id, at desc)` - lookup by actor
- `audit_logs(tenant_id, action, at desc)` - filter per aksi
- partition by `at` month; retain 2-7 tahun sesuai compliance

## API Minimal
- `GET /api/audit?entity=&entityId=&actor=&action=&from=&to=&severity=&page=&pageSize=`
- `GET /api/audit/:id` (detail satu entry + full diff)
- `GET /api/audit/export?...` (CSV, untuk compliance)
- `GET /api/audit/entity/:entityType/:entityId` (histori per entity, contoh: `GET /api/audit/entity/sale/<saleId>`)
- Tulis hanya via service internal (tidak ada endpoint POST dari FE)

## Catatan SaaS
- `tenant_id` wajib + RLS
- Append-only: tidak ada UPDATE/DELETE pada baris audit (enforce via DB role permission atau trigger)
- `diff` disimpan sebagai JSONB, exclude field sensitif (password_hash, api_key_encrypted, webhook_secret_encrypted)
- PII redaction untuk email/phone bila perlu (`actor_email` hashed)
- Retention policy: minimal 2 tahun untuk SaaS compliance (SOC2, GDPR right-to-access)
- Untuk superadmin posku (internal): `tenant_id` nullable + scope global

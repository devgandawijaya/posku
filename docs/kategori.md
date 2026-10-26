# Modul: Kategori

Route FE: `http://localhost:5173/kategori`
File FE: `src/pages/KategoriPage.tsx`, `src/components/KategoriHeader.tsx`, `src/components/KategoriFilters.tsx`, `src/components/KategoriTable.tsx`, `src/components/KategoriCharts.tsx`, `src/components/KategoriKpi.tsx`
Type/domain: `src/lib/kategori.types.ts`, hook: `src/lib/useKategori.ts`

## Ringkasan Fitur
- CRUD kategori + subkategori (self-reference, 1 level)
- Setiap kategori punya scope toko: `string[]` (daftar `storeId`) atau `'all'`
- Per-toko: agregat stok (`perStoreStock[]`) + omzet
- Status: `active | inactive`
- KPI: total, aktif, nonaktif, total produk, total omzet

## Skema Tabel Backend

### `categories`
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `parent_id` UUID nullable FK `categories` (self-ref, subkategori)
- `code` varchar (mis. `Kopi`, `Snack`)
- `name` varchar
- `slug` varchar (lowercase, dash)
- `icon` varchar (FA class: `fa-mug-hot`)
- `scope` enum `('all', 'specific')`
- `is_active` bool
- `sort_order` int default 0
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable

### `category_stores` (mapping scope 'specific')
- `category_id` UUID FK
- `outlet_id` UUID FK `outlets`
- PK komposit `(category_id, outlet_id)`
- `tenant_id` UUID (denormalized untuk RLS)

### `category_kpi_cache` (view/tabel agregat)
- `category_id` UUID
- `tenant_id` UUID
- `product_count` int
- `total_stock` numeric
- `total_omzet` numeric(15,2)
- `last_synced_at` timestamp
- Dipakai untuk render KPI tabel; update via trigger/event saat `products` / `sales` berubah.

## Field yang Diturunkan (dari `Category` FE)
| Field FE | Sumber di DB |
|---|---|
| `id` | `categories.id` |
| `code`, `name`, `icon` | `categories.code/name/icon` |
| `parentId` | `categories.parent_id` |
| `storeIds` (`'all' \| string[]`) | `scope='all'` -> `'all'`; else `category_stores.outlet_id[]` |
| `perStoreStock[]` | agregat `stocks.qty` join `outlets.name` group by `outlet_id` |
| `totalOmzet` | sum `sale_items.subtotal` join `products` where `products.category_id` |
| `status` | `categories.is_active` -> `active`/`inactive` |

## Index Penting
- `unique (tenant_id, parent_id, slug)` (slug unik per parent)
- `unique (tenant_id, code)` (code unik per tenant)
- `category_stores(tenant_id, outlet_id)`

## API Minimal
- `GET /api/categories?q=&status=&storeId=&parentId=`
- `POST /api/categories` (body: `name, code, parentId?, scope, storeIds[], icon, isActive`)
- `PATCH /api/categories/:id`
- `DELETE /api/categories/:id` (tolak bila masih ada `products.category_id` aktif; atau reassign)
- `GET /api/categories/:id/kpi?from=&to=` (balik `productCount, totalOmzet, perStoreStock`)

## Catatan SaaS
- `tenant_id` wajib di semua tabel + filter service / RLS
- Soft delete `deleted_at` untuk restore
- Audit log untuk create/update/delete (ikut modul `audit_logs` umum)

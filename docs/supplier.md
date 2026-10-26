# Modul: Supplier

Route FE: `http://localhost:5173/supplier`
File FE: `src/pages/SupplierPage.tsx`, `src/components/SupplierHeader.tsx`, `src/components/SupplierTable.tsx`, `src/components/SupplierKpi.tsx`, `src/components/SupplierCharts.tsx`, `src/components/SupplierModals.tsx`
Type/domain: `src/lib/supplier.types.ts`, hook: `src/lib/useSupplier.ts`

## Ringkasan Fitur
- CRUD supplier per tenant
- Field: kode, nama, contact person, phone, email, alamat, kategori, status
- Assign multi-outlet (`assignedStoreIds[]`)
- Denormalized metrics: `totalProducts`, `totalPurchases`, `lastOrderAt`
- KPI: total, aktif, nonaktif, total purchases
- Pagination + filter (status, store, search)
- Hak akses granular (create/edit/delete/assign/toggle)

## Skema Tabel Backend

### `suppliers`
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `code` varchar unique per tenant (auto `SUP1001`)
- `name` varchar
- `contact_person` varchar
- `phone` varchar
- `email` varchar nullable
- `address` text nullable
- `category` varchar (`Bahan Baku`, `Kemasan`, `Minuman`, `Snack`, `ATK`, `Lainnya`)
- `status` enum `('aktif', 'nonaktif')` default `aktif`
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`, `deleted_at` nullable

### `supplier_outlets` (assign toko)
- `supplier_id` UUID FK
- `outlet_id` UUID FK `outlets`
- PK komposit `(supplier_id, outlet_id)`
- `tenant_id` UUID

### `supplier_metrics_cache`
- `supplier_id` UUID PK
- `tenant_id` UUID
- `total_products` int
- `total_purchases` numeric(18,2)
- `last_order_at` timestamp nullable
- `updated_at` timestamp

## Field yang Diturunkan (dari `Supplier` FE)
| Field FE | Sumber di DB |
|---|---|
| `id`, `code`, `name` | `suppliers.*` |
| `companyId` | `suppliers.tenant_id` |
| `contactPerson`, `phone`, `email`, `address` | `suppliers.*` |
| `category` | `suppliers.category` |
| `assignedStoreIds` | `supplier_outlets.outlet_id[]` |
| `totalProducts` | `supplier_metrics_cache.total_products` |
| `totalPurchases` | `supplier_metrics_cache.total_purchases` |
| `status` | `suppliers.status` |
| `createdAt` | `suppliers.created_at` |
| `lastOrderAt` | `supplier_metrics_cache.last_order_at` |

## Index Penting
- `unique (tenant_id, code)`
- `index (tenant_id, status)`
- `index (tenant_id, name)` (search)
- `supplier_outlets(tenant_id, outlet_id)`

## API Minimal
- `GET /api/suppliers?storeId=&status=&q=&page=&pageSize=`
- `GET /api/suppliers/:id`
- `POST /api/suppliers`
- `PATCH /api/suppliers/:id`
- `DELETE /api/suppliers/:id` (tolak bila `total_purchases > 0`; reassign ke supplier lain)
- `POST /api/suppliers/:id/toggle-status`
- `POST /api/suppliers/:id/assign-outlets` (body: `outletIds[]`)
- `GET /api/suppliers/options` (dropdown)

## Catatan SaaS
- `tenant_id` wajib + RLS
- Audit log untuk create/update/delete/assign/toggle
- Validasi `email` regex + `phone` format lokal

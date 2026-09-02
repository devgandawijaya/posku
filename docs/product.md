# Fitur Halaman Tambah Produk

Route: `http://localhost:5173/tambah-produk`
File FE: `src/pages/TambahProdukPage.tsx` + `ProductForm.tsx` + `ProductDetailDrawer.tsx` + `ProductTable.tsx` + `ProductFiltersBar.tsx` + `useProductCatalog.ts`
Type/domain: `src/lib/product.types.ts`, hook: `src/lib/useProductCatalog.ts`

## Ringkasan Halaman
- Judul: "Manajemen Produk"
- Dua tab: **Daftar Produk** dan **Tambah / Edit Produk**
- Tombol aksi kanan-atas toggle antara `+ Tambah Produk` dan `<- Kembali ke Daftar`
- Judul: "Manajemen Produk"
- Dua tab: **Daftar Produk** dan **Tambah / Edit Produk**
- Tombol aksi kanan-atas toggle antara `+ Tambah Produk` dan `<- Kembali ke Daftar`


## Halaman FE Terkait (Lengkap)
- `/tambah-produk` -> `TambahProdukPage.tsx` + `ProductForm.tsx` + `ProductDetailDrawer.tsx` + `ProductTable.tsx` + `ProductFiltersBar.tsx` + `useProductCatalog.ts`
- `/manajemen-produk` (atau `DaftarProdukPage.tsx`) -> `ProductListHeader.tsx` + `ProductListFilters.tsx` + `ProductListTable.tsx` + `ProductListKpi.tsx` + `ProductListCharts.tsx` + `ProductListBulk.tsx` + `useProductList.ts`

Perbedaan:
- `TambahProdukPage`: fokus form create/edit + detail drawer + tab daftar sederhana (10/page, filter kategori + status)
- `DaftarProdukPage`: list enterprise dengan bulk action (transfer/activate/deactivate/delete), import CSV, export, filter toko + filter stok (`available|low|out`), KPI total/active/out/value, chart by store & top sellers
## Fitur Form Tambah / Edit Produk
- Upload gambar (placeholder icon FA): pasang / lepas gambar, catatan "JPG/PNG maks 2MB"
- Field teks wajib: **Nama Produk**, **SKU**, **Barcode**
- Dropdown: **Kategori** (Kopi, Teh, Minuman, Makanan, Snack, ATK), **Satuan** (pcs, kg, pack, box, liter)
- Numerik: **Harga Jual**, **Harga Modal**, **Stok Awal** (semua `min=0`)
- Dropdown **Toko Awal** (dari `mockStores`)
- Dropdown **Status** (Aktif / Nonaktif)
- Aksi: `Simpan` (submit form) dan `Batal` (kembali ke list)

## Fitur Daftar Produk
- **Pencarian** bebas teks (cari di Nama / SKU / Barcode)
- **Filter Kategori** + **Filter Status** (Semua / Aktif / Nonaktif)
- Tombol **Reset** filter + tombol **Tambah**
- **Tabel** kolom: Produk (thumb + nama + tgl dibuat), SKU/Barcode, Kategori, Harga, Stok, Satuan, Status, Aksi
- Stok `< 10` tampil badge warning warna
- Aksi baris: **Detail** (buka drawer), **Edit** (masuk form), **Hapus** (konfirmasi)
- **Pagination** 10 baris/halaman, kontrol Prev/Next + info "Halaman X dari Y"

## Fitur Drawer Detail Produk
- Header: nama, SKU, kategori-satuan, badge status, barcode monospace
- **Informasi Inti**: Harga Jual, Harga Modal, Margin %, Stok Total, Total Terjual, Pendapatan
- Tab mini:
  - **Stok per Toko**: distribusi qty per outlet + pergerakan stok terakhir (alasan: masuk/keluar/adjust/return/sale, ref)
  - **Riwayat Harga**: daftar perubahan harga, persen delta, oleh siapa
  - **Histori Penjualan**: spark bar qty/hari + list qty & revenue per tanggal
- Footer drawer: **Cetak Label** barcode (mock), **Edit Produk**

## Catatan Implementasi Saat Ini (Mock)
- Submit form hanya `alert("disimpan (mock)")`, belum hit API
- Hapus hanya `alert`, belum panggil endpoint
- Sumber data `mockProducts` di `src/lib/mockProducts.ts`
- Pagination/filter client-side

---

# Kebutuhan Tabel Backend (Sistem SaaS Multi-Tenant)

Skema dirancang untuk SaaS dengan isolasi per **tenant** (perusahaan/merchant), multi-outlet, multi-user dengan role.

## Wajib Multi-Tenant
- Setiap tabel bisnis membawa `tenant_id` (UUID) + index komposit
- Strategi isolasi: **shared DB, shared schema** + `tenant_id` (RLS di DB direkomendasikan)
- Tabel `tenants`, `users`, `tenant_members(tenant_id, user_id, role_id, outlet_ids[])`

## Tabel Inti Produk

### `tenants`
- `id` UUID PK
- `name`, `slug` unique
- `plan` enum (free/pro/enterprise), `status` (active/suspended)
- `created_at`, `updated_at`

### `categories`
- `id` UUID PK, `tenant_id` UUID FK
- `name`, `slug`, `parent_id` UUID nullable (self-ref)
- `is_active` bool
- unique `(tenant_id, slug)`

### `units`
- `id` UUID PK, `tenant_id` UUID FK
- `code` (pcs/kg/pack/box/liter), `name`
- unique `(tenant_id, code)`

### `products`
- `id` UUID PK, `tenant_id` UUID FK
- `sku` varchar, `barcode` varchar nullable
- `name`, `description` text nullable
- `category_id` UUID FK -> categories
- `unit_id` UUID FK -> units
- `price` numeric(15,2) (harga jual)
- `cost` numeric(15,2) (modal, nullable)
- `min_stock` numeric(15,3) default 0 (ambang warning)
- `is_active` bool (active/inactive)
- `image_url` text nullable, `icon` varchar nullable (fallback FA)
- `created_by`, `updated_by` UUID FK -> users
- `created_at`, `updated_at`, `deleted_at` nullable (soft delete)
- unique `(tenant_id, sku)`
- index `(tenant_id, category_id)`, `(tenant_id, is_active)`, `lower(name)` untuk search

### `product_images`
- `id` UUID PK, `product_id` FK
- `url`, `is_primary` bool, `sort` int
- `created_at`

### `product_barcodes`
- 1 produk bisa multi barcode
- `id`, `product_id`, `barcode` unique per tenant
- index `unique (tenant_id, barcode)`

## Outlet & Stok per Outlet

### `outlets`
- `id` UUID PK, `tenant_id` UUID FK
- `name`, `code`, `address`, `phone`, `is_active`
- `created_at`, `updated_at`

### `stocks`
- `id` UUID PK, `tenant_id`, `product_id`, `outlet_id`
- `qty` numeric(15,3) default 0
- `reserved_qty` numeric(15,3) default 0 (untuk hold order)
- unique `(tenant_id, product_id, outlet_id)`
- index `(tenant_id, outlet_id)`, `(tenant_id, product_id)`

### `stock_movements`
- `id` UUID PK, `tenant_id`
- `product_id`, `outlet_id`
- `delta` numeric(15,3) (signed)
- `reason` enum: `masuk|keluar|adjust|return|sale|transfer`
- `ref_type` (purchase/sale/return/transfer/adjustment), `ref_id` UUID
- `note`, `created_by`, `created_at`
- index `(tenant_id, product_id, outlet_id, created_at desc)`

### `stock_transfers` (opsional)
- `id`, `tenant_id`, `from_outlet_id`, `to_outlet_id`, `status`
- `items[]` JSONB / tabel `stock_transfer_items`

## Harga & Riwayat

### `price_levels` (opsional, harga grosir/member)
- `id`, `tenant_id`, `name`

### `product_prices`
- `id`, `tenant_id`, `product_id`, `price_level_id` nullable
- `price` numeric(15,2), `min_qty` int default 1

### `price_history`
- `id`, `tenant_id`, `product_id`
- `old_price`, `new_price` numeric(15,2)
- `changed_by`, `changed_at`

## Supplier & Pembelian (Purchase)

### `suppliers`
- `id`, `tenant_id`, `name`, `phone`, `email`, `address`, `is_active`

### `purchases` & `purchase_items`
- Header: `id`, `tenant_id`, `supplier_id`, `outlet_id`, `invoice_no`, `status` (draft/approved/void), `total`, `created_at`
- Items: `purchase_id`, `product_id`, `qty`, `cost`, `subtotal`

## Penjualan & Retur (untuk histori penjualan + retur refund)

### `sales` & `sale_items`
- Header: `id`, `tenant_id`, `outlet_id`, `customer_id` nullable, `invoice_no`, `status`, `subtotal`, `discount`, `tax`, `total`, `payment_method`, `created_by`, `created_at`
- Items: `sale_id`, `product_id`, `qty`, `price`, `discount`, `subtotal`

### `returns` & `return_items`
- Header: `id`, `tenant_id`, `sale_id`, `outlet_id`, `status`, `reason`, `refund_method`, `total_refund`, `created_at`
- Items: `return_id`, `sale_item_id`, `product_id`, `qty`, `amount`

## Hak Akses (RBAC)

### `roles`
- `id`, `tenant_id`, `name` (Owner/Admin/Manager/Kasir/Gudang)
- unique `(tenant_id, name)`

### `permissions`
- `id`, `code` (`product.read`, `product.write`, `product.delete`, `stock.adjust`, dll)

### `role_permissions`
- `role_id`, `permission_id` PK komposit

### `user_outlets` (scoping kasir ke outlet)
- `user_id`, `outlet_id`, PK komposit

## Audit Log (wajib untuk SaaS)
- `audit_logs`
- `id`, `tenant_id`, `actor_id`, `entity` (products/stocks/...), `entity_id`, `action` (create/update/delete), `diff` JSONB, `ip`, `created_at`
- Index `(tenant_id, entity, entity_id, created_at desc)`

## Subscription / Billing (SaaS)

### `plans`, `subscriptions`, `invoices`, `payments`
- `subscriptions(tenant_id, plan_id, status, current_period_start/end)`
- `invoices(tenant_id, amount, status, due_date)`
- `payments(tenant_id, invoice_id, method, amount, paid_at)`

## API Endpoint Minimal
- `POST /api/products` (create + validasi SKU unik per tenant)
- `GET /api/products?q=&category=&status=&page=&page_size=`
- `GET /api/products/:id` (ikutkan `stocks_by_outlet`, `price_history`, `sales_history` ringkas)
- `PATCH /api/products/:id`
- `DELETE /api/products/:id` (soft delete)
- `POST /api/products/:id/image` (upload, simpan object storage)
- `GET /api/products/:id/stock-movements?limit=`
- `GET /api/products/:id/price-history?limit=`
- `GET /api/products/:id/sales-history?from=&to=`
- `GET /api/products/:id/barcode/label` (cetak PDF/thermal)
- Bulk: `POST /api/products/bulk` (import CSV/Excel), `POST /api/products/bulk-delete`

## Constraint & Index Penting
- `unique (tenant_id, sku)` di `products`
- `unique (tenant_id, barcode)` di `product_barcodes`
- `unique (tenant_id, product_id, outlet_id)` di `stocks`
- Index untuk query SaaS umum:
  - `products(tenant_id, category_id, is_active)`
  - `stocks(tenant_id, outlet_id)`
  - `sales(tenant_id, outlet_id, created_at desc)`
  - `stock_movements(tenant_id, product_id, created_at desc)`
- Soft delete `deleted_at` di tabel yang boleh restore

## Pertahanan & Kepatuhan
- Row-Level Security policy `tenant_id = current_setting('app.tenant_id')`
- Audit log untuk create/update/delete/harga/stok
- Rate-limit endpoint tulis
- Validasi size & tipe gambar (maks 2MB sesuai FE), scan antivirus
- Backup harian per tenant, export data (GDPR-like)

## Catatan SaaS
- Gunakan **shared DB + shared schema** untuk efisiensi; tambah `tenant_id` di semua tabel + filter wajib di service layer / RLS
- Skema siap **horizontal scaling**: UUID PK, hindari sequence per-tenant
- Pisah service: catalog, stock, sales, purchase, return, billing, audit
- Siapkan **feature flag** per plan (mis. multi-outlet hanya di plan Pro+)





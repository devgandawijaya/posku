# Modul: Retur & Refund

Route FE: `http://localhost:5173/retur-refund`
File FE: `src/pages/ReturRefundPage.tsx`, `src/components/ReturnFiltersBar.tsx`, `src/components/ReturnTable.tsx`, `src/components/ReturnDrawer.tsx`, `src/components/ReturnCharts.tsx`, `src/components/ReturnKpi.tsx`
Type/domain: `src/lib/retur.types.ts`, hook: `src/lib/useReturnList.ts`, data: `src/lib/mockReturns.ts`

## Ringkasan Fitur
- List pengajuan retur lintas outlet
- Filter: rentang tanggal, toko, shift, kasir, status, alasan, search
- Status alur: `pending | approved | rejected | processing | completed`
- Alasan: `damaged | wrong-item | not-as-described | expired | changed-mind | other`
- Fate item (integrasi stok): `restock | qc | damaged | supplier`
- Approval berjenjang: `Kasir -> Supervisor -> Manager`
- KPI: total pengajuan, nominal, pending, approved/processing

## Skema Tabel Backend

### `returns` (header pengajuan)
- `id` UUID PK
- `tenant_id` UUID FK `tenants`
- `outlet_id` UUID FK `outlets`
- `shift_id` UUID FK `shifts` nullable
- `cashier_user_id` UUID FK `users`
- `customer_id` UUID nullable FK `customers`
- `origin_sale_id` UUID FK `sales`
- `origin_invoice_no` varchar (snapshot)
- `origin_payment` enum `('cash','qris','debit','credit','ewallet')`
- `date` timestamp
- `total_refund` numeric(15,2)
- `reason` enum `('damaged','wrong-item','not-as-described','expired','changed-mind','other')`
- `reason_note` text nullable
- `status` enum `('pending','approved','rejected','processing','completed')` default `pending`
- `qc_note` text nullable
- `restock_at` timestamp nullable
- `created_by`, `updated_by` UUID FK `users`
- `created_at`, `updated_at`

### `return_items`
- `id` UUID PK
- `return_id` UUID FK `returns`
- `sale_item_id` UUID FK `sale_items`
- `product_id` UUID FK `products`
- `sku_snapshot`, `name_snapshot` varchar
- `qty` numeric(15,3)
- `price` numeric(15,2)
- `amount` numeric(15,2)
- `fate` enum `('restock','qc','damaged','supplier')`
- `tenant_id` UUID

### `return_approvals` (workflow berjenjang)
- `id` UUID PK
- `return_id` UUID FK `returns`
- `step` int (1,2,3)
- `role` enum `('Kasir','Supervisor','Manager')`
- `approver_user_id` UUID nullable FK `users`
- `approver_name` varchar (snapshot)
- `status` enum `('done','current','pending','rejected')`
- `ts` timestamp nullable
- `note` text nullable
- `tenant_id` UUID

### `refund_payments` (pencatatan refund uang)
- `id` UUID PK
- `return_id` UUID FK `returns`
- `tenant_id` UUID
- `amount` numeric(15,2)
- `method` enum `('cash','qris','debit','credit','ewallet','voucher')`
- `ref_no` varchar nullable
- `paid_at` timestamp
- `by_user_id` UUID FK `users`

## Field yang Diturunkan (dari `Return` FE)
| Field FE | Sumber di DB |
|---|---|
| `id`, `date` | `returns.id`, `returns.date` |
| `store`, `storeId` | `outlets.name`, `returns.outlet_id` |
| `shift` | `shifts.code` snapshot |
| `cashier` | `users.name` |
| `customer` | `customers.name` (atau `Walk-in`) |
| `originInvoice` | `returns.origin_invoice_no` |
| `originPayment` | `returns.origin_payment` |
| `items[]` | `return_items` + `fate` |
| `total` | `returns.total_refund` |
| `reason`, `reasonNote` | `returns.reason`, `returns.reason_note` |
| `status` | `returns.status` |
| `approval[]` | `return_approvals` order by step |
| `qcNote`, `restockAt` | `returns.qc_note`, `returns.restock_at` |

## Index Penting
- `index (tenant_id, outlet_id, date desc)`
- `index (tenant_id, status, date desc)`
- `index (tenant_id, origin_sale_id)`
- `return_items(tenant_id, product_id)`
- `return_approvals(tenant_id, return_id, step)`

## API Minimal
- `GET /api/returns?dateFrom=&dateTo=&storeId=&shift=&cashier=&status=&reason=&q=&page=&pageSize=`
- `GET /api/returns/:id` (ikut `items[], approval[], refund_payments[]`)
- `POST /api/returns` (dari kasir saat transaksi)
- `POST /api/returns/:id/approve` (next step, sesuai permission role)
- `POST /api/returns/:id/reject` (body: `note`)
- `POST /api/returns/:id/process` (set status `processing`, eksekusi `fate` -> stok)
- `POST /api/returns/:id/complete` (set `completed`, trigger refund payment)
- `POST /api/returns/:id/refund` (catat `refund_payments`)

## Catatan SaaS
- `tenant_id` wajib + RLS
- Snapshot field di `return_items` agar histori stabil
- Saat `fate='restock'`, generate `stock_movements` (+delta) -> lihat `stok.md`
- Workflow approval dikonfigurasi per tenant (default: Kasir -> Supervisor -> Manager)
- Audit log wajib untuk approve/reject/process

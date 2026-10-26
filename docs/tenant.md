# Tenant Module

## 1. Scope

Dokumen ini hanya mengatur pekerjaan **Tenant Module** pada existing POS SaaS project.

AI Assistant hanya boleh melakukan perubahan yang berhubungan langsung dengan:

* Tenant / Company
* Tenant identification
* Tenant context
* Tenant data isolation
* Tenant relationship
* Tenant lifecycle
* Tenant validation
* Tenant authorization boundary
* Tenant-related database changes
* Tenant-related API jika memang diperlukan
* Tenant-related testing

Dokumen ini **tidak membahas atau mengubah**:

* Arsitektur aplikasi
* Framework
* Bahasa pemrograman
* Struktur folder
* Struktur file utama
* Arsitektur JSON API response
* Existing authentication architecture
* Existing authorization architecture
* Existing UI architecture
* Existing business module architecture

---

# 2. Existing Project Rules

Project ini adalah **existing project**.

AI Assistant wajib:

1. Membaca implementasi Tenant yang sudah ada.
2. Mengidentifikasi bagaimana Tenant saat ini disimpan.
3. Mengidentifikasi bagaimana Tenant saat ini diambil.
4. Mengidentifikasi bagaimana Tenant saat ini dikaitkan dengan user.
5. Mengidentifikasi bagaimana Tenant saat ini digunakan dalam business query.
6. Mengidentifikasi apakah tenant isolation sudah tersedia.
7. Mengidentifikasi gap yang masih ada.
8. Memperbaiki hanya bagian Tenant yang diperlukan.

---

# 3. DO NOT CHANGE

## 3.1 Application Architecture

Jangan mengubah:

* Application architecture
* Framework
* Programming language
* Existing architectural pattern
* Existing module architecture

Jika Tenant perlu diintegrasikan ke architecture existing, lakukan secara **incremental**.

---

## 3.2 File & Folder Architecture

Jangan:

* Memindahkan folder.
* Rename folder.
* Rename package/module utama.
* Membuat struktur aplikasi baru.
* Membuat parallel architecture.
* Membuat duplicate Tenant module.

AI harus mengikuti struktur file dan folder yang sudah ada.

---

## 3.3 JSON API Response Architecture

**JSON API response architecture SUDAH ADA.**

AI Assistant:

> **DILARANG mengubah, mengganti, atau membuat standard response JSON baru.**

Gunakan response architecture yang sudah digunakan existing project.

AI wajib mencari dan mengikuti:

* Existing response wrapper
* Existing response DTO
* Existing success response
* Existing error response
* Existing validation response
* Existing pagination response jika digunakan

Jika Tenant membutuhkan API baru:

> Gunakan response architecture existing.

Jangan membuat:

```text
TenantResponseV2
TenantApiResponse
TenantStandardResponse
TenantCustomResponse
```

jika tidak diperlukan oleh architecture existing.

---

# 4. Tenant Definition

Tenant adalah **Company / Business Entity** yang menjadi batas utama isolasi data dalam POS SaaS.

Contoh:

```text
Tenant A
└── Company A

Tenant B
└── Company B

Tenant C
└── Company C
```

Data Tenant A tidak boleh dapat diakses oleh Tenant B.

---

# 5. Tenant Boundary

Tenant menjadi business data boundary.

Target relationship:

```text
Tenant
│
├── Store
├── Employee
├── User
├── Role
├── Permission
├── Product
├── Category
├── Warehouse
├── Inventory
├── Customer
├── Supplier
├── Purchase
├── Sales
├── Payment
├── Shift
├── Closing
├── Finance
├── Integration
└── Audit Log
```

Tidak semua entity harus mempunyai `tenant_id` secara langsung jika architecture existing menggunakan relationship yang valid untuk menentukan Tenant.

AI harus **memeriksa existing relationship terlebih dahulu**.

---

# 6. Primary Objective

Tujuan utama pekerjaan Tenant:

> Memastikan seluruh business operation yang seharusnya tenant-scoped berjalan dalam Tenant Context yang benar tanpa merusak architecture existing.

Prioritas:

```text
Tenant Identification
        ↓
Tenant Context
        ↓
Tenant Validation
        ↓
Tenant Authorization Boundary
        ↓
Tenant Data Isolation
```

---

# 7. Tenant Context

AI harus mencari bagaimana existing project menentukan tenant.

Kemungkinan sumber:

```text
Authenticated User
        ↓
User Tenant
```

atau:

```text
JWT / Session
        ↓
Tenant Context
```

atau:

```text
Employee
        ↓
Company / Tenant
```

atau mekanisme existing lainnya.

**Jangan membuat mekanisme baru jika existing mechanism sudah tersedia.**

---

# 8. Tenant Context Rules

Tenant context harus berasal dari sumber yang terpercaya.

Prioritas:

```text
Authenticated Context
        ↓
Tenant Context
```

Bukan:

```text
Client Request
        ↓
tenant_id bebas
        ↓
Database
```

Jika existing endpoint menerima:

```json
{
  "tenant_id": 123
}
```

AI harus menganalisis apakah field tersebut:

* memang diperlukan,
* hanya digunakan sebagai filter,
* digunakan sebagai security boundary,
* atau berpotensi menyebabkan tenant data leakage.

Jangan langsung menghapus parameter existing karena dapat menyebabkan breaking API.

Jika perlu diperbaiki, lakukan dengan perubahan minimal dan backward compatible.

---

# 9. Tenant Isolation

Tenant isolation adalah prioritas utama.

Contoh:

```text
Tenant A
    |
    +── Product A
    +── Customer A
    +── Sales A
    +── Stock A


Tenant B
    |
    +── Product B
    +── Customer B
    +── Sales B
    +── Stock B
```

Tenant A tidak boleh:

```text
READ    → Tenant B
CREATE  → Tenant B
UPDATE  → Tenant B
DELETE  → Tenant B
```

---

# 10. Tenant Isolation Validation

AI harus memeriksa query/data access yang berkaitan dengan Tenant.

Contoh pola yang harus diperiksa:

```text
SELECT *
FROM products
WHERE id = ?
```

Jika Product adalah tenant-scoped, AI harus memastikan ownership Tenant tetap tervalidasi.

Target logical behavior:

```text
SELECT *
FROM products
WHERE id = ?
AND tenant_id = currentTenant
```

Namun:

> Jangan memaksakan query tersebut jika existing architecture sudah mempunyai tenant filtering melalui mechanism lain.

AI harus memahami architecture existing terlebih dahulu.

---

# 11. Tenant Data Access

Untuk setiap module yang menggunakan Tenant, AI harus memeriksa:

```text
CREATE
READ
UPDATE
DELETE
LIST
SEARCH
DETAIL
REPORT
EXPORT
IMPORT
```

Pastikan operation tersebut tidak melewati Tenant boundary.

---

# 12. Tenant Create

Jika existing project sudah memiliki Tenant creation flow:

> Reuse existing flow.

AI hanya memperbaiki jika ditemukan gap.

Data Tenant minimal secara business concept dapat mencakup:

```text
id
code
name
status
created_at
updated_at
```

Tetapi AI **tidak boleh langsung menambahkan field tersebut** sebelum memeriksa existing schema.

Jika field existing memiliki nama berbeda:

> Gunakan mapping existing.

---

# 13. Tenant Identity

Tenant harus mempunyai identity yang stabil.

AI harus mencari existing identifier seperti:

```text
tenant_id
company_id
organization_id
business_id
```

Jika existing project menggunakan:

```text
company_id
```

dan secara business meaning sudah berfungsi sebagai Tenant:

> Jangan membuat `tenant_id` kedua hanya karena target model menggunakan istilah Tenant.

Buat mapping:

```text
Existing:
company_id

Business Concept:
tenant_id
```

Jika secara fungsi memang sama.

---

# 14. Tenant Code

Jika existing Tenant memiliki unique business identifier seperti:

```text
tenant_code
company_code
organization_code
```

gunakan mekanisme existing.

Jangan membuat identifier kedua tanpa kebutuhan.

Tenant code harus unique sesuai business rule existing.

---

# 15. Tenant Status

AI harus mencari status Tenant existing.

Contoh business state:

```text
ACTIVE
INACTIVE
SUSPENDED
```

Namun jangan menambahkan status baru jika existing project sudah mempunyai lifecycle/status yang dapat digunakan.

Tenant yang tidak aktif harus mengikuti business rule existing.

AI harus memeriksa apakah Tenant inactive:

* Tidak dapat login.
* Tidak dapat melakukan transaksi.
* Tidak dapat membuat data baru.
* Masih dapat dibaca oleh administrator.
* Atau memiliki behavior lain.

**Jangan menentukan behavior sendiri jika belum ditemukan di existing project.**

---

# 16. Tenant Relationship

AI harus melakukan mapping relationship Tenant.

Minimal analisis:

```text
Tenant
├── Users
├── Employees
├── Stores
├── Roles
├── Products
├── Warehouses
├── Customers
├── Suppliers
├── Transactions
├── Payments
├── Inventory
├── Finance
└── Audit Logs
```

Untuk setiap relationship tentukan:

```text
Entity
Relationship
Current Foreign Key
Tenant Scope
Current Query
Risk
```

---

# 17. Tenant & User

AI harus mencari relationship:

```text
Tenant
   ↓
User
```

atau:

```text
User
   ↓
Employee
   ↓
Tenant
```

atau mekanisme existing lainnya.

Jangan membuat relationship kedua jika relationship existing sudah benar.

---

# 18. Tenant & Employee

Employee harus dapat diketahui Tenant-nya.

Target:

```text
Tenant
  |
  └── Employee
```

Jika employee sudah memiliki relationship ke Company/Organization/Business:

> Gunakan relationship existing.

---

# 19. Tenant & Store

Store harus mempunyai Tenant ownership.

Target:

```text
Tenant A
├── Store Jakarta
├── Store Bandung
└── Store Depok
```

Tenant B:

```text
Tenant B
├── Store Jakarta
└── Store Bogor
```

Store dengan nama sama tetap dapat berada pada Tenant berbeda karena Tenant merupakan boundary.

---

# 20. Cross Tenant Store Protection

AI harus memastikan:

```text
Tenant A User
    ↓
Store B milik Tenant B
```

tidak dapat digunakan.

Hal ini harus divalidasi pada:

* Store selection
* Transaction
* Product
* Stock
* Purchase
* Sales
* Shift
* Closing
* Report

---

# 21. Tenant & Product

Product harus mengikuti tenant ownership sesuai architecture existing.

Target:

```text
Tenant A
├── Product A
└── Product B

Tenant B
├── Product C
└── Product D
```

AI harus memeriksa apakah product:

* Direct tenant scoped
* Store scoped
* Global master
* Shared master
* Tenant-owned

Jangan mengubah model tersebut sebelum analysis.

---

# 22. Tenant & Inventory

Inventory harus tidak dapat cross-tenant.

Target:

```text
Tenant A
├── Warehouse A
│   └── Stock A
│
└── Warehouse B
    └── Stock B
```

Tenant B tidak boleh membaca stock Tenant A.

AI harus memeriksa:

* Warehouse ownership
* Stock ownership
* Stock movement
* Stock transfer
* Stock adjustment

---

# 23. Tenant & Transaction

Transaction harus tenant scoped.

AI harus memeriksa:

```text
Sales
Purchase
Payment
Return
Refund
Shift
Closing
```

Tidak boleh:

```text
Tenant A
   ↓
Transaction belonging to Tenant B
```

---

# 24. Tenant & Finance

Financial data harus tenant isolated.

AI harus memeriksa:

```text
Financial Account
Financial Transaction
Financial Ledger
Receivable
Payable
Cash Movement
```

Tenant A tidak boleh melihat financial data Tenant B.

Karena financial data bersifat critical:

> Perubahan Tenant isolation pada Finance termasuk **HIGH/CRITICAL risk**.

---

# 25. Tenant & Reporting

Semua report harus mengikuti Tenant Context.

Contoh:

```text
GET /sales/report
```

Harus menghasilkan data berdasarkan authenticated Tenant.

Bukan:

```text
GET /sales/report?tenant_id=otherTenant
```

Jika existing API menggunakan parameter tenant:

> Analisis security behavior sebelum melakukan perubahan.

---

# 26. Tenant & Audit Log

Audit log harus dapat mengidentifikasi Tenant.

Target:

```text
Audit Log
├── tenant
├── user
├── employee
├── store
├── action
├── entity
├── entity_id
└── timestamp
```

Jika audit log existing sudah mempunyai tenant/company relationship:

> Reuse.

---

# 27. Tenant API

AI hanya boleh membuat atau mengubah Tenant API jika memang diperlukan oleh existing business flow.

Sebelum membuat endpoint:

1. Cari existing Tenant controller.
2. Cari existing Tenant service.
3. Cari existing Tenant repository.
4. Cari existing Tenant entity/model.
5. Cari existing Tenant route.
6. Cari existing Tenant API client.
7. Cari existing response DTO.

Jika sudah ada:

> REUSE.

---

# 28. JSON Response Rule

**JANGAN membuat response architecture baru.**

Jika existing API memiliki:

```json
{
  "existing": "response structure"
}
```

Tenant API wajib menggunakan structure tersebut.

AI tidak boleh mengubah:

```text
Response Wrapper
Error Wrapper
Pagination Wrapper
Validation Response
HTTP Contract
```

hanya untuk Tenant Module.

---

# 29. Tenant API Validation

Tenant endpoint harus divalidasi terhadap:

```text
Authentication
Authorization
Tenant Ownership
Input Validation
Data Integrity
Existing API Contract
```

---

# 30. Database Analysis

AI harus melakukan:

```text
1. Find existing tenant table
2. Find existing tenant identifier
3. Find related entities
4. Find foreign keys
5. Find indexes
6. Find unique constraints
7. Find tenant-related migration
8. Find tenant-related query
```

Kemudian dokumentasikan:

```text
Current Table
Current Columns
Current Relations
Current Constraints
Current Usage
Gap
Risk
Recommendation
```

---

# 31. Migration Rules

Migration Tenant hanya boleh dibuat jika diperlukan.

Dilarang:

```text
DROP tenant table
TRUNCATE tenant table
RESET tenant data
DELETE existing tenant records
```

Jika membutuhkan migration:

```text
Existing Schema
      ↓
Impact Analysis
      ↓
Migration Proposal
      ↓
Approval if required
      ↓
Incremental Migration
```

---

# 32. Tenant Security Checklist

AI wajib memeriksa:

* [ ] Tenant context berasal dari authenticated context.
* [ ] Tenant tidak dapat dimanipulasi client.
* [ ] Tenant ownership divalidasi.
* [ ] Tenant-scoped query aman.
* [ ] Detail endpoint aman.
* [ ] List endpoint aman.
* [ ] Search endpoint aman.
* [ ] Update endpoint aman.
* [ ] Delete endpoint aman.
* [ ] Report endpoint aman.
* [ ] Export endpoint aman.
* [ ] Import endpoint aman.
* [ ] Transaction aman.
* [ ] Inventory aman.
* [ ] Finance aman.
* [ ] Audit log aman.

---

# 33. Tenant Gap Analysis

Gunakan format:

| Area           | Existing     | Target                 | Gap | Risk | Recommendation |
| -------------- | ------------ | ---------------------- | --- | ---- | -------------- |
| Tenant Entity  | Not verified | Tenant entity exists   | TBD | TBD  | Analyze        |
| Tenant Context | Not verified | Authenticated context  | TBD | TBD  | Analyze        |
| User Relation  | Not verified | User belongs to tenant | TBD | TBD  | Analyze        |
| Store Relation | Not verified | Store tenant scoped    | TBD | TBD  | Analyze        |
| Product        | Not verified | Tenant isolated        | TBD | TBD  | Analyze        |
| Inventory      | Not verified | Tenant isolated        | TBD | TBD  | Analyze        |
| Transaction    | Not verified | Tenant isolated        | TBD | TBD  | Analyze        |
| Finance        | Not verified | Tenant isolated        | TBD | TBD  | Analyze        |
| Report         | Not verified | Tenant scoped          | TBD | TBD  | Analyze        |

**AI harus mengganti `Not verified` setelah membaca project.**

Jangan mengisi berdasarkan asumsi.

---

# 34. Tenant Conflict Analysis

Jika ditemukan conflict gunakan:

```markdown
## Conflict

### ID
TENANT-CONFLICT-001

### Area
<area>

### Current Implementation
<existing behavior>

### Target
<target behavior>

### Conflict
<explanation>

### Impact
<impact>

### Risk
LOW / MEDIUM / HIGH / CRITICAL

### Recommendation
<minimal change>

### Breaking Change
YES / NO

### Approval Required
YES / NO
```

---

# 35. Tenant Files Analysis

AI harus mencari file existing yang berkaitan dengan Tenant.

Jangan mengarang nama file.

Output harus berupa:

| File            | Purpose     | Tenant Relation | Change             |
| --------------- | ----------- | --------------- | ------------------ |
| `<actual-file>` | `<purpose>` | `<relation>`    | `<YES/NO/LIMITED>` |

Jika file tidak ditemukan:

```text
NOT FOUND IN EXISTING PROJECT
```

---

# 36. Tenant Testing

Minimal test scenario:

## Tenant Isolation

```text
Tenant A cannot read Tenant B.
Tenant A cannot update Tenant B.
Tenant A cannot delete Tenant B.
Tenant A cannot create data for Tenant B.
```

## Authentication Context

```text
Authenticated User → Tenant A
Business Operation → Tenant A
```

## Store Context

```text
Tenant A → Store A
Tenant A cannot use Store B belonging to Tenant B.
```

## Transaction

```text
Tenant A cannot access Tenant B transaction.
```

## Inventory

```text
Tenant A cannot access Tenant B stock.
```

## Finance

```text
Tenant A cannot access Tenant B financial data.
```

## Reporting

```text
Tenant A report contains only Tenant A data.
```

---

# 37. Acceptance Criteria

Tenant module dianggap selesai jika:

* Tenant identity existing telah dipahami.
* Tenant context existing telah dipahami.
* Tenant relationship telah dipetakan.
* Tenant isolation telah diverifikasi.
* Cross-tenant access telah diuji.
* Existing API response tidak berubah.
* Existing application architecture tidak berubah.
* Existing folder architecture tidak berubah.
* Existing file architecture tidak berubah.
* Tidak ada duplicate Tenant system.
* Tidak ada data existing yang hilang.
* Migration jika diperlukan bersifat incremental.
* Test Tenant isolation tersedia.
* Business flow Tenant tervalidasi.

---

# 38. AI Working Procedure

Untuk Tenant Module:

```text
STEP 1
READ EXISTING PROJECT

STEP 2
FIND TENANT IMPLEMENTATION

STEP 3
MAP TENANT RELATIONSHIPS

STEP 4
ANALYZE TENANT CONTEXT

STEP 5
ANALYZE TENANT ISOLATION

STEP 6
ANALYZE DATABASE

STEP 7
ANALYZE API

STEP 8
ANALYZE SECURITY

STEP 9
GAP ANALYSIS

STEP 10
CONFLICT ANALYSIS

STEP 11
RISK ANALYSIS

STEP 12
RECOMMEND MINIMAL CHANGES

STEP 13
STOP

STEP 14
WAIT FOR APPROVAL

STEP 15
IMPLEMENT ONLY AFTER APPROVAL
```

---

# 39. First Response for Tenant Module

Pada pertama kali AI diminta mengerjakan Tenant Module:

**JANGAN CODING.**

AI harus memberikan:

```text
# TENANT MODULE ANALYSIS

## 1. Existing Tenant Implementation

## 2. Existing Tenant Database

## 3. Existing Tenant Context

## 4. Existing Tenant Relationship

## 5. Existing Tenant API

## 6. Existing Tenant Data Access

## 7. Existing Tenant Isolation

## 8. Existing Tenant Security

## 9. Gap Analysis

## 10. Conflict Analysis

## 11. Risk Analysis

## 12. Files Affected

## 13. Database Changes

## 14. API Changes

## 15. Test Scenarios

## 16. Recommended Implementation Plan

## 17. Approval Required
```

---

# 40. Mandatory Final Status

Setelah analysis selesai:

```text
STATUS: ANALYSIS_COMPLETED
MODULE: TENANT
CODING: NOT_STARTED
API_RESPONSE_ARCHITECTURE: UNCHANGED
APPLICATION_ARCHITECTURE: UNCHANGED
FILE_FOLDER_ARCHITECTURE: UNCHANGED
APPROVAL_REQUIRED: YES
```

> **STOP.**
>
> Jangan membuat perubahan code sebelum user memberikan approval.

---

# 41. Core Principle

> **Tenant Module hanya boleh memperbaiki dan melengkapi fungsi Tenant pada existing POS SaaS project.**

> **Jangan mengganggu architecture JSON API yang sudah ada.**

> **Jangan mengganggu architecture aplikasi yang sudah ada.**

> **Jangan mengganggu struktur file dan folder yang sudah ada.**

> **Jangan membuat ulang module yang sudah tersedia.**

> **Baca existing implementation terlebih dahulu.**

> **Jika tidak ditemukan, katakan `NOT FOUND IN EXISTING PROJECT`.**

> **Jangan berasumsi.**

> **Jangan coding sebelum analysis dan approval selesai.**

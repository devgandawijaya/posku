# API JSON Response Architecture

Dokumen ini mendefinisikan **standar arsitektur JSON response** yang digunakan oleh API.

Fokus dokumen ini hanya pada:

* Struktur JSON response
* Standard field
* Success response
* Error response
* Pagination
* Metadata
* Version
* Timestamp
* Request ID
* Error code
* Data structure

Dokumen ini **tidak membahas arsitektur aplikasi, backend, database, controller, service, repository, atau framework**.

---

# 1. Standard Response Structure

Semua response API harus memiliki struktur dasar berikut:

```json
{
  "success": true,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Request successfully processed",
  "data": {},
  "meta": null,
  "error": null,
  "requestId": "req_01K3XYZABC123"
}
```

Struktur JSON:

```text
Response
├── success
├── version
├── timestamp
├── message
├── data
├── meta
├── error
└── requestId
```

---

# 2. Response Field

| Field       | Type              | Required | Description                            |
| ----------- | ----------------- | -------- | -------------------------------------- |
| `success`   | boolean           | Yes      | Menentukan status keberhasilan request |
| `version`   | string            | Yes      | Versi API                              |
| `timestamp` | string            | Yes      | Waktu response dibuat                  |
| `message`   | string            | Yes      | Informasi hasil request                |
| `data`      | object/array/null | Yes      | Payload utama                          |
| `meta`      | object/null       | Yes      | Metadata response                      |
| `error`     | object/null       | Yes      | Informasi error                        |
| `requestId` | string            | Yes      | Identitas unik request                 |

---

# 3. Success Response

Pada response sukses:

```text
success = true
error   = null
```

Contoh:

```json
{
  "success": true,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Product retrieved successfully",
  "data": {
    "id": 1001,
    "sku": "PRD-001",
    "name": "Indomie Goreng",
    "price": 3500,
    "status": "ACTIVE"
  },
  "meta": null,
  "error": null,
  "requestId": "req_01K3XYZABC123"
}
```

---

# 4. Success List Response

Untuk response yang menghasilkan banyak data:

```json
{
  "success": true,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Products retrieved successfully",
  "data": [
    {
      "id": 1001,
      "sku": "PRD-001",
      "name": "Indomie Goreng",
      "price": 3500
    },
    {
      "id": 1002,
      "sku": "PRD-002",
      "name": "Teh Botol",
      "price": 5000
    }
  ],
  "meta": null,
  "error": null,
  "requestId": "req_01K3XYZABC123"
}
```

`data` dapat berupa:

```text
object
array
null
```

---

# 5. Empty Data Response

Tidak adanya data bukan berarti error.

Gunakan:

```json
{
  "success": true,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Products retrieved successfully",
  "data": [],
  "meta": null,
  "error": null,
  "requestId": "req_01K3XYZABC123"
}
```

Jangan gunakan:

```json
{
  "success": false,
  "message": "Products not found"
}
```

untuk kondisi list kosong.

---

# 6. Pagination Response

Jika response menggunakan pagination, informasi pagination ditempatkan di dalam `meta`.

```json
{
  "success": true,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Products retrieved successfully",
  "data": [
    {
      "id": 1001,
      "name": "Indomie Goreng"
    },
    {
      "id": 1002,
      "name": "Teh Botol"
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "perPage": 20,
      "total": 150,
      "totalPages": 8
    }
  },
  "error": null,
  "requestId": "req_01K3XYZABC123"
}
```

Struktur:

```text
meta
└── pagination
    ├── page
    ├── perPage
    ├── total
    └── totalPages
```

---

# 7. Pagination dengan Navigation

Untuk API yang membutuhkan informasi navigasi:

```json
{
  "success": true,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Products retrieved successfully",
  "data": [],
  "meta": {
    "pagination": {
      "page": 2,
      "perPage": 20,
      "total": 150,
      "totalPages": 8,
      "hasNext": true,
      "hasPrevious": true
    }
  },
  "error": null,
  "requestId": "req_01K3XYZABC123"
}
```

---

# 8. Metadata

`meta` digunakan untuk informasi tambahan yang tidak termasuk payload utama.

Contoh:

```json
{
  "meta": {
    "pagination": {
      "page": 1,
      "perPage": 20,
      "total": 150,
      "totalPages": 8
    }
  }
}
```

Contoh metadata lainnya:

```json
{
  "meta": {
    "pagination": {
      "page": 1,
      "perPage": 20,
      "total": 150,
      "totalPages": 8
    },
    "sort": {
      "field": "name",
      "direction": "asc"
    },
    "filter": {
      "status": "ACTIVE"
    }
  }
}
```

---

# 9. Error Response

Pada response error:

```text
success = false
data    = null
error   = object
```

Contoh:

```json
{
  "success": false,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Product not found",
  "data": null,
  "meta": null,
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "details": null
  },
  "requestId": "req_01K3XYZABC123"
}
```

---

# 10. Error Object

Struktur error:

```text
error
├── code
└── details
```

Contoh:

```json
{
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "details": {
      "productId": 1001
    }
  }
}
```

---

# 11. Error Code

`error.code` harus menggunakan identifier yang konsisten.

Contoh:

```text
VALIDATION_ERROR
UNAUTHORIZED
FORBIDDEN
RESOURCE_NOT_FOUND
PRODUCT_NOT_FOUND
CUSTOMER_NOT_FOUND
DUPLICATE_RESOURCE
INVALID_REQUEST
INVALID_PARAMETER
INSUFFICIENT_STOCK
INVALID_STATUS
CONFLICT
INTERNAL_SERVER_ERROR
SERVICE_UNAVAILABLE
```

Gunakan:

```text
UPPER_SNAKE_CASE
```

Jangan gunakan:

```text
"ProductNotFound"
"product-not-found"
"productNotFound"
```

---

# 12. Validation Error

Untuk validation error, `details` dapat berisi error berdasarkan field.

```json
{
  "success": false,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Validation failed",
  "data": null,
  "meta": null,
  "error": {
    "code": "VALIDATION_ERROR",
    "details": {
      "name": [
        "Name is required"
      ],
      "price": [
        "Price must be greater than 0"
      ],
      "sku": [
        "SKU already exists"
      ]
    }
  },
  "requestId": "req_01K3XYZABC123"
}
```

---

# 13. Multiple Validation Errors

Satu field dapat memiliki lebih dari satu error:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "details": {
      "password": [
        "Password is required",
        "Password must contain at least 8 characters",
        "Password must contain at least one number"
      ]
    }
  }
}
```

---

# 14. Timestamp

`timestamp` harus menggunakan format ISO 8601.

Recommended:

```text
2026-08-22T15:04:28Z
```

atau dengan timezone:

```text
2026-08-22T22:04:28+07:00
```

Format yang tidak direkomendasikan:

```text
22/08/2026 15:04:28
22-08-2026 15:04:28
08/22/2026
```

Untuk sistem API, penggunaan UTC lebih direkomendasikan:

```text
2026-08-22T15:04:28Z
```

---

# 15. API Version

Field:

```json
{
  "version": "v1"
}
```

menunjukkan versi contract API yang digunakan oleh response.

Contoh:

```json
{
  "version": "v1"
}
```

Jika API berubah secara breaking:

```json
{
  "version": "v2"
}
```

Version tidak digunakan untuk versi aplikasi.

Contoh yang **tidak direkomendasikan**:

```json
{
  "version": "1.8.2"
}
```

Jika ingin menyimpan versi aplikasi, gunakan field berbeda:

```json
{
  "version": "v1",
  "appVersion": "1.8.2"
}
```

---

# 16. Request ID

`requestId` adalah identifier unik untuk satu request.

Contoh:

```json
{
  "requestId": "req_01K3XYZABC123"
}
```

Karakteristik:

* Unique
* Tidak berubah selama satu request
* Dapat digunakan untuk tracing
* Tidak mengandung informasi sensitif

---

# 17. Data Object

Jika response menghasilkan satu resource:

```json
{
  "data": {
    "id": 1001,
    "name": "Indomie Goreng"
  }
}
```

Struktur:

```text
data
├── id
├── name
└── ...
```

---

# 18. Data Array

Jika response menghasilkan collection:

```json
{
  "data": [
    {
      "id": 1001,
      "name": "Indomie Goreng"
    },
    {
      "id": 1002,
      "name": "Teh Botol"
    }
  ]
}
```

---

# 19. Data Null

Jika tidak ada payload:

```json
{
  "data": null
}
```

Contoh error:

```json
{
  "success": false,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Unauthorized",
  "data": null,
  "meta": null,
  "error": {
    "code": "UNAUTHORIZED",
    "details": null
  },
  "requestId": "req_01K3XYZABC123"
}
```

---

# 20. Consistency Rule

Response harus selalu mengikuti contract yang sama.

### Success

```text
success = true
data    = object | array | null
error   = null
```

### Error

```text
success = false
data    = null
error   = object
```

Jangan membuat struktur seperti:

```json
{
  "status": "success",
  "result": {}
}
```

kemudian endpoint lain:

```json
{
  "code": 200,
  "payload": {}
}
```

Gunakan satu contract.

---

# 21. Recommended Final Structure

Struktur yang direkomendasikan:

```json
{
  "success": true,
  "version": "v1",
  "timestamp": "2026-08-22T15:04:28Z",
  "message": "Request successfully processed",
  "data": {},
  "meta": null,
  "error": null,
  "requestId": "req_01K3XYZABC123"
}
```

Secara konseptual:

```text
API Response
│
├── success
│
├── version
│
├── timestamp
│
├── message
│
├── data
│   ├── object
│   ├── array
│   └── null
│
├── meta
│   ├── pagination
│   ├── sorting
│   ├── filtering
│   └── other metadata
│
├── error
│   ├── code
│   └── details
│
└── requestId
```

---

# 22. Contract Summary

| Condition        | success | data       | meta        | error  |
| ---------------- | ------: | ---------- | ----------- | ------ |
| Single success   |  `true` | object     | null/object | null   |
| List success     |  `true` | array      | null/object | null   |
| Empty list       |  `true` | array `[]` | null/object | null   |
| Create success   |  `true` | object     | null/object | null   |
| Update success   |  `true` | object     | null/object | null   |
| Delete success   |  `true` | null       | null        | null   |
| Validation error | `false` | null       | null        | object |
| Unauthorized     | `false` | null       | null        | object |
| Forbidden        | `false` | null       | null        | object |
| Not found        | `false` | null       | null        | object |
| Conflict         | `false` | null       | null        | object |
| Server error     | `false` | null       | null        | object |

---

# 23. Golden Rule

> **API response harus predictable, consistent, machine-readable, dan mudah diproses oleh client.**

Contract utama:

```json
{
  "success": true,
  "version": "v1",
  "timestamp": "ISO-8601",
  "message": "Human readable message",
  "data": {},
  "meta": null,
  "error": null,
  "requestId": "unique-request-id"
}
```

Contract ini menjadi **single JSON response standard** untuk seluruh API.

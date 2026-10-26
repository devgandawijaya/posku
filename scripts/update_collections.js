const fs = require('fs');
const path = require('path');

const postmanPath = path.join(__dirname, '..', 'posku_postman_collection.json');
const swaggerDocsPath = path.join(__dirname, '..', 'docs', 'swagger.json');
const swaggerCtrlPath = path.join(__dirname, '..', 'internal', 'controllers', 'swagger.json');

// --- Helper for Postman Items ---
function makeHeader(key, value) {
  return { key, value };
}

function makeJsonPostmanItem(name, method, url, rawBody) {
  const item = {
    name,
    request: {
      method,
      header: rawBody ? [makeHeader('Content-Type', 'application/json')] : [],
      url
    }
  };
  if (rawBody) {
    item.request.body = {
      mode: 'raw',
      raw: rawBody
    };
  }
  return item;
}

// 1. Update Postman Collection
console.log('Loading Postman collection...');
let postmanRaw = fs.readFileSync(postmanPath, 'utf8').replace(/^\uFEFF/, '');
const coll = JSON.parse(postmanRaw);

function getFolder(name) {
  const f = coll.item.find(x => x.name.toLowerCase().includes(name.toLowerCase()));
  if (!f) throw new Error(`Folder "${name}" not found in collection`);
  return f;
}

// 1.1 Warehouses
const whFolder = getFolder('Warehouses');
// Update standard POST /warehouses
const postWh = whFolder.item.find(x => x.name === 'POST /warehouses');
if (postWh) {
  postWh.name = 'POST /warehouses (Outlet Warehouse)';
  postWh.request.body.raw = JSON.stringify({
    store_id: '{{store_id}}',
    code: 'WH-0001',
    name: 'Gudang Toko Utama',
    location: 'Jl. Sudirman No. 10',
    is_central: false
  }, null, 2).replace('"{{store_id}}"', '{{store_id}}');
}

// Add Central Warehouse POST if not exists
if (!whFolder.item.some(x => x.name.includes('Central Warehouse'))) {
  const centralWh = makeJsonPostmanItem(
    'POST /warehouses (Central Distribution Warehouse)',
    'POST',
    '{{base_url}}/warehouses',
    JSON.stringify({
      code: 'WH-CENTRAL',
      name: 'Gudang Logistik Pusat',
      location: 'Kawasan Industri MM2100',
      is_central: true
    }, null, 2)
  );
  whFolder.item.splice(1, 0, centralWh);
}

// Update GET /warehouses pagination
const getWh = whFolder.item.find(x => x.name === 'GET /warehouses');
if (getWh) {
  getWh.request.url = '{{base_url}}/warehouses?page=1&per_page=20';
}

// 1.2 Products
const prodFolder = getFolder('Products');
const getProd = prodFolder.item.find(x => x.name === 'GET /products');
if (getProd) {
  getProd.request.url = '{{base_url}}/products?page=1&per_page=20';
}

// Add Store Prices endpoints to Products
if (!prodFolder.item.some(x => x.name.includes('/store-prices'))) {
  prodFolder.item.push(
    makeJsonPostmanItem(
      'POST /products/:id/store-prices',
      'POST',
      '{{base_url}}/products/{{product_id}}/store-prices',
      JSON.stringify({
        store_id: '{{store_id}}',
        price: 125000,
        cost: 85000
      }, null, 2).replace('"{{store_id}}"', '{{store_id}}')
    ),
    makeJsonPostmanItem(
      'GET /products/:id/store-prices',
      'GET',
      '{{base_url}}/products/{{product_id}}/store-prices'
    ),
    makeJsonPostmanItem(
      'DELETE /products/:id/store-prices/:storeId',
      'DELETE',
      '{{base_url}}/products/{{product_id}}/store-prices/{{store_id}}'
    )
  );
}

// 1.3 Stok Extras: Add GET /stocks/:productId/by-warehouse
const stockFolder = getFolder('Stok Extras');
if (!stockFolder.item.some(x => x.name.includes('by-warehouse'))) {
  stockFolder.item.push(
    makeJsonPostmanItem(
      'GET /stocks/:productId/by-warehouse',
      'GET',
      '{{base_url}}/stocks/{{product_id}}/by-warehouse'
    )
  );
}

// 1.4 Kasir: Shifts
const shiftFolder = getFolder('Kasir: Shifts');
const closeShift = shiftFolder.item.find(x => x.name.includes('close'));
if (closeShift) {
  closeShift.request.body.raw = JSON.stringify({
    actual_cash: 650000,
    notes: 'Tutup shift kasir pagi. Kas fisik sesuai hasil perhitungan.'
  }, null, 2);
}

if (!shiftFolder.item.some(x => x.name.includes('summary'))) {
  shiftFolder.item.push(
    makeJsonPostmanItem(
      'GET /shifts/:id/summary',
      'GET',
      '{{base_url}}/shifts/{{shift_id}}/summary'
    )
  );
}

const getShifts = shiftFolder.item.find(x => x.name === 'GET /shifts');
if (getShifts) {
  getShifts.request.url = '{{base_url}}/shifts?page=1&per_page=20';
}

// 1.5 Kasir: Carts (Hold Order & Checkout)
const cartFolder = getFolder('Kasir: Carts');
if (!cartFolder.item.some(x => x.name.includes('Split Payment'))) {
  cartFolder.item.push(
    makeJsonPostmanItem(
      'POST /carts/:id/checkout (Split Payment / Multi-Tender)',
      'POST',
      '{{base_url}}/carts/1/checkout',
      JSON.stringify({
        customer_id: '{{customer_id}}',
        payments: [
          {
            method: 'cash',
            amount: 50000,
            cash_received: 50000
          },
          {
            method: 'qris',
            amount: 25000,
            reference_number: 'QRIS-20261026-001'
          }
        ]
      }, null, 2).replace('"{{customer_id}}"', '{{customer_id}}')
    )
  );
}

const holdOrders = cartFolder.item.find(x => x.name.includes('hold-orders'));
if (holdOrders) {
  holdOrders.request.url = '{{base_url}}/cashier/hold-orders?cashier_id={{employee_id}}&page=1&per_page=20';
}

// 1.6 Sales Transactions (Transaksi)
const salesFolder = getFolder('Sales Transactions');
if (!salesFolder.item.some(x => x.name.includes('Split Payment'))) {
  salesFolder.item.splice(1, 0,
    makeJsonPostmanItem(
      'POST /sales (Split Payment / Multi-Tender)',
      'POST',
      '{{base_url}}/sales',
      JSON.stringify({
        store_id: '{{store_id}}',
        employee_id: '{{employee_id}}',
        customer_id: '{{customer_id}}',
        total_amount: 75000,
        items: [
          {
            product_id: '{{product_id}}',
            quantity: 1,
            price: 75000
          }
        ],
        payments: [
          {
            method: 'cash',
            amount: 50000,
            cash_received: 50000,
            reference_number: ''
          },
          {
            method: 'qris',
            amount: 25000,
            reference_number: 'QRIS-20261026-001'
          }
        ]
      }, null, 2)
        .replace('"{{store_id}}"', '{{store_id}}')
        .replace('"{{employee_id}}"', '{{employee_id}}')
        .replace('"{{customer_id}}"', '{{customer_id}}')
        .replace('"{{product_id}}"', '{{product_id}}')
    )
  );
}

const getSales = salesFolder.item.find(x => x.name === 'GET /sales');
if (getSales) {
  getSales.request.url = '{{base_url}}/sales?page=1&per_page=20&store_id={{store_id}}&employee_id={{employee_id}}&shift_id={{shift_id}}&status=completed';
}

// 1.7 Reports (Laporan)
const reportFolder = getFolder('Reports (Laporan)');
if (!reportFolder.item.some(x => x.name.includes('by-employee'))) {
  reportFolder.item.push(
    makeJsonPostmanItem(
      'GET /reports/sales/by-employee',
      'GET',
      '{{base_url}}/reports/sales/by-employee?date_from=2026-10-01&date_to=2026-10-31'
    )
  );
}
if (!reportFolder.item.some(x => x.name.includes('by-shift'))) {
  reportFolder.item.push(
    makeJsonPostmanItem(
      'GET /reports/sales/by-shift',
      'GET',
      '{{base_url}}/reports/sales/by-shift?date_from=2026-10-01&date_to=2026-10-31'
    )
  );
}
if (!reportFolder.item.some(x => x.name.includes('by-warehouse'))) {
  reportFolder.item.push(
    makeJsonPostmanItem(
      'GET /reports/stock/by-warehouse',
      'GET',
      '{{base_url}}/reports/stock/by-warehouse?warehouse_id={{warehouse_id}}'
    )
  );
}

// 1.8 Pagination Query Parameters in other master data folders
try {
  const storeFolder = getFolder('Stores');
  const getStores = storeFolder.item.find(x => x.name === 'GET /stores');
  if (getStores) getStores.request.url = '{{base_url}}/stores?page=1&per_page=20';
} catch (e) {}

try {
  const catFolder = getFolder('Categories');
  const getCats = catFolder.item.find(x => x.name === 'GET /categories');
  if (getCats) getCats.request.url = '{{base_url}}/categories?page=1&per_page=20';
} catch (e) {}

try {
  const empFolder = getFolder('Employees');
  const getEmps = empFolder.item.find(x => x.name === 'GET /employees');
  if (getEmps) getEmps.request.url = '{{base_url}}/employees?page=1&per_page=20';
} catch (e) {}

try {
  const custFolder = getFolder('Customers');
  const getCusts = custFolder.item.find(x => x.name === 'GET /customers');
  if (getCusts) getCusts.request.url = '{{base_url}}/customers?page=1&per_page=20';
} catch (e) {}

try {
  const suppFolder = getFolder('Suppliers');
  const getSupps = suppFolder.item.find(x => x.name === 'GET /suppliers');
  if (getSupps) getSupps.request.url = '{{base_url}}/suppliers?page=1&per_page=20';
} catch (e) {}

try {
  const vpFolder = getFolder('Kasir: Vouchers & Promotions');
  const getVouchers = vpFolder.item.find(x => x.name === 'GET /vouchers');
  if (getVouchers) getVouchers.request.url = '{{base_url}}/vouchers?page=1&per_page=20';
  const getPromos = vpFolder.item.find(x => x.name === 'GET /promotions');
  if (getPromos) getPromos.request.url = '{{base_url}}/promotions?page=1&per_page=20';
} catch (e) {}

// Write updated Postman collection
fs.writeFileSync(postmanPath, JSON.stringify(coll, null, 2), 'utf8');
console.log('Successfully updated posku_postman_collection.json');


// 2. Update Swagger Documentation
console.log('Loading Swagger document...');
let swaggerRaw = fs.readFileSync(swaggerDocsPath, 'utf8').replace(/^\uFEFF/, '');
const swagger = JSON.parse(swaggerRaw);

const stdResponses = {
  "200": { "description": "Successful operation" },
  "400": { "description": "Bad Request" },
  "401": { "description": "Unauthorized" },
  "403": { "description": "Forbidden" },
  "404": { "description": "Not Found" },
  "500": { "description": "Internal Server Error" }
};

// Add /products/{{product_id}}/store-prices
swagger.paths['/products/{{product_id}}/store-prices'] = {
  post: {
    summary: "POST /products/:id/store-prices",
    tags: ["Products"],
    parameters: [],
    requestBody: {
      content: {
        "application/json": {
          example: JSON.stringify({
            store_id: "{{store_id}}",
            price: 125000,
            cost: 85000
          }, null, 2)
        }
      }
    },
    responses: stdResponses
  },
  get: {
    summary: "GET /products/:id/store-prices",
    tags: ["Products"],
    parameters: [],
    responses: stdResponses
  }
};

// Add /products/{{product_id}}/store-prices/{{store_id}}
swagger.paths['/products/{{product_id}}/store-prices/{{store_id}}'] = {
  delete: {
    summary: "DELETE /products/:id/store-prices/:storeId",
    tags: ["Products"],
    parameters: [],
    responses: stdResponses
  }
};

// Add /stocks/{{product_id}}/by-warehouse
swagger.paths['/stocks/{{product_id}}/by-warehouse'] = {
  get: {
    summary: "GET /stocks/:productId/by-warehouse",
    tags: ["Stok Extras"],
    parameters: [],
    responses: stdResponses
  }
};

// Add /shifts/{{shift_id}}/summary
swagger.paths['/shifts/{{shift_id}}/summary'] = {
  get: {
    summary: "GET /shifts/:id/summary",
    tags: ["Kasir: Shifts"],
    parameters: [],
    responses: stdResponses
  }
};

// Add /reports/sales/by-employee
swagger.paths['/reports/sales/by-employee'] = {
  get: {
    summary: "GET /reports/sales/by-employee",
    tags: ["Reports (Laporan)"],
    parameters: [],
    responses: stdResponses
  }
};

// Add /reports/sales/by-shift
swagger.paths['/reports/sales/by-shift'] = {
  get: {
    summary: "GET /reports/sales/by-shift",
    tags: ["Reports (Laporan)"],
    parameters: [],
    responses: stdResponses
  }
};

// Add /reports/stock/by-warehouse
swagger.paths['/reports/stock/by-warehouse'] = {
  get: {
    summary: "GET /reports/stock/by-warehouse",
    tags: ["Reports (Laporan)"],
    parameters: [],
    responses: stdResponses
  }
};

// Write updated Swagger to docs/swagger.json and internal/controllers/swagger.json
const swaggerStr = JSON.stringify(swagger, null, 2);
fs.writeFileSync(swaggerDocsPath, swaggerStr, 'utf8');
fs.writeFileSync(swaggerCtrlPath, swaggerStr, 'utf8');
console.log('Successfully updated docs/swagger.json and internal/controllers/swagger.json');


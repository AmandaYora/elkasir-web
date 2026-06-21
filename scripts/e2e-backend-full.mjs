// COMPREHENSIVE backend E2E — exercises every previously-untested module/endpoint
// against the live API exactly as real clients do, plus edge cases & cross-tenant checks.
// Run: node scripts/e2e-backend-full.mjs   (API up + base fixtures seeded)
const BASE = process.env.API_BASE_URL || 'http://localhost:8081/api/v1';
const ADMIN = { email: 'admin', password: 'admin123' };           // seeded owner
const CASHIER = { username: 'kasiruji', password: 'kasir123' };    // seeded staff cashier
const SUP = { username: 'supuji', password: 'super123' };          // seeded staff supervisor
const WEBHOOK_TOKEN = process.env.XENDIT_WEBHOOK_TOKEN || 'elk-test-webhook-token';
const STORE_B_PRODUCT_ID = process.env.STORE_B_PRODUCT_ID || '';   // inserted via SQL for tenancy test
const TAG = String(Date.now()).slice(-6);

let pass = 0, fail = 0;
const fails = [];
function ok(cond, label) {
  if (cond) { pass++; console.log(`  ✓ ${label}`); }
  else { fail++; fails.push(label); console.log(`  ✗ FAIL: ${label}`); }
}
function section(t) { console.log(`\n══ ${t} ══`); }

async function api(path, { method = 'GET', token, body, idem, raw, headers } = {}) {
  const res = await fetch(BASE + path, {
    method,
    headers: {
      'content-type': 'application/json',
      ...(token ? { authorization: `Bearer ${token}` } : {}),
      ...(idem ? { 'Idempotency-Key': idem } : {}),
      ...(headers || {}),
    },
    body: raw !== undefined ? raw : body ? JSON.stringify(body) : undefined,
  });
  let json = null;
  try { json = await res.json(); } catch { /* 204 */ }
  return { status: res.status, json };
}
const adminLogin = async (c = ADMIN) => (await api('/auth/admin/login', { method: 'POST', body: c })).json?.data?.accessToken;
const staffLogin = async (c = CASHIER) => (await api('/auth/staff/login', { method: 'POST', body: c }));

let AT, CT, ST; // admin token, cashier token, supervisor token
let prod; // base product Kopi Uji

(async () => {
  console.log(`COMPREHENSIVE backend E2E @ ${BASE}  (tag ${TAG})`);
  AT = await adminLogin();
  CT = (await staffLogin(CASHIER)).json?.data?.accessToken;
  ST = (await staffLogin(SUP)).json?.data?.accessToken;
  if (!AT || !CT || !ST) { console.error('login bootstrap failed'); process.exit(2); }
  prod = (await api('/products', { token: AT })).json?.data?.find((p) => p.name === 'Kopi Uji');
  if (!prod) { console.error('base product Kopi Uji missing — run e2e-fixtures.mjs'); process.exit(2); }

  // ===================== AUTH FLOWS =====================
  section('AUTH: refresh rotation, me, cross-actor guards, logout');
  {
    const login = await api('/auth/admin/login', { method: 'POST', body: ADMIN });
    ok(login.status === 200, 'admin login 200');
    const rt = login.json.data.refreshToken;
    const refreshed = await api('/auth/refresh', { method: 'POST', body: { refreshToken: rt } });
    ok(refreshed.status === 200 && refreshed.json.data.accessToken, 'refresh returns a new token pair');
    const reuseOld = await api('/auth/refresh', { method: 'POST', body: { refreshToken: rt } });
    ok(reuseOld.status === 401, 'old refresh token is revoked after rotation (401)');

    const meAdmin = await api('/auth/me', { token: AT });
    ok(meAdmin.status === 200 && meAdmin.json.data.actor === 'admin', 'me() resolves admin actor');
    const meStaff = await api('/auth/me', { token: CT });
    ok(meStaff.status === 200 && meStaff.json.data.actor === 'staff', 'me() resolves staff actor');

    // cross-actor: staff cannot hit admin-only master data; admin cannot create a sale.
    const staffWritesProduct = await api('/products', { method: 'POST', token: CT, body: { sku: `X-${TAG}`, name: 'X', price: 1, cost: 0, stock: 1, status: 'active' } });
    ok(staffWritesProduct.status === 403, 'staff actor forbidden on admin product create (403)');
    const adminCreatesSale = await api('/transactions', { method: 'POST', token: AT, idem: `x-${TAG}`, body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'cash', amountReceived: 99999, orderType: 'takeaway' } });
    ok(adminCreatesSale.status === 403, 'admin actor forbidden on staff-only sale create (403)');

    const wrong = await api('/auth/admin/login', { method: 'POST', body: { email: 'admin', password: 'salah' } });
    ok(wrong.status === 401, 'wrong admin password 401');

    // logout on a throwaway session, then the refresh is dead.
    const t2 = await api('/auth/admin/login', { method: 'POST', body: ADMIN });
    const lo = await api('/auth/logout', { method: 'POST', body: { refreshToken: t2.json.data.refreshToken } });
    ok(lo.status === 204 || lo.status === 200, 'logout accepted (204)');
    const afterLogout = await api('/auth/refresh', { method: 'POST', body: { refreshToken: t2.json.data.refreshToken } });
    ok(afterLogout.status === 401, 'refresh after logout is rejected (401)');
  }

  // ===================== CATEGORIES CRUD =====================
  section('CATEGORIES: CRUD + duplicate');
  let catId;
  {
    const c = await api('/categories', { method: 'POST', token: AT, body: { name: `Minuman-${TAG}`, sortOrder: 1 } });
    ok(c.status === 201, 'create category 201');
    catId = c.json.data.id;
    const list = await api('/categories', { token: AT });
    ok(list.json.data.some((x) => x.id === catId), 'category appears in list');
    const get = await api(`/categories/${catId}`, { token: AT });
    ok(get.status === 200, 'get category 200');
    const upd = await api(`/categories/${catId}`, { method: 'PUT', token: AT, body: { name: `Minuman2-${TAG}`, sortOrder: 2 } });
    ok(upd.status === 200, 'update category 200');
    const dup = await api('/categories', { method: 'POST', token: AT, body: { name: `Minuman2-${TAG}` } });
    ok(dup.status === 409, 'duplicate category name 409');
  }

  // ===================== PRODUCTS mutations =====================
  section('PRODUCTS: create(+category), update, adjust-stock, dup SKU, delete');
  let pid;
  {
    const c = await api('/products', { method: 'POST', token: AT, body: { categoryId: catId, sku: `TEH-${TAG}`, name: `Teh-${TAG}`, price: 8000, cost: 3000, stock: 100, status: 'active' } });
    ok(c.status === 201, 'create product 201');
    pid = c.json.data.id;
    const upd = await api(`/products/${pid}`, { method: 'PUT', token: AT, body: { categoryId: catId, sku: `TEH-${TAG}`, name: `Teh Manis-${TAG}`, price: 9000, cost: 3000, stock: 100, status: 'active' } });
    ok(upd.status === 200 && upd.json.data.price === 9000, 'update product price 200');
    const add = await api(`/products/${pid}/adjust-stock`, { method: 'POST', token: AT, body: { delta: 50 } });
    ok(add.status === 200, 'adjust-stock +50 200');
    const after1 = (await api(`/products/${pid}`, { token: AT })).json.data.stock;
    ok(after1 === 150, `stock 100 -> 150 after +50 (got ${after1})`);
    const sub = await api(`/products/${pid}/adjust-stock`, { method: 'POST', token: AT, body: { delta: -30 } });
    ok(sub.status === 200, 'adjust-stock -30 200');
    const after2 = (await api(`/products/${pid}`, { token: AT })).json.data.stock;
    ok(after2 === 120, `stock 150 -> 120 after -30 (got ${after2})`);
    const dup = await api('/products', { method: 'POST', token: AT, body: { sku: `TEH-${TAG}`, name: 'dup', price: 1, cost: 0, stock: 1, status: 'active' } });
    ok(dup.status === 409, 'duplicate SKU 409');
  }

  // ===================== TABLES mutations =====================
  section('TABLES: create, update, dup code, delete');
  let tid;
  {
    const c = await api('/tables', { method: 'POST', token: AT, body: { code: `T-${TAG}`, name: 'Meja Tes', area: 'Indoor', seats: 4, status: 'active' } });
    ok(c.status === 201, 'create table 201');
    tid = c.json.data.id;
    const upd = await api(`/tables/${tid}`, { method: 'PUT', token: AT, body: { code: `T-${TAG}`, name: 'Meja Tes 2', area: 'Outdoor', seats: 6, status: 'active' } });
    ok(upd.status === 200 && upd.json.data.seats === 6, 'update table 200');
    const dup = await api('/tables', { method: 'POST', token: AT, body: { code: `T-${TAG}`, name: 'dup', area: '', seats: 0, status: 'active' } });
    ok(dup.status === 409, 'duplicate table code 409');
  }

  // ===================== STAFF mutations =====================
  section('STAFF: create, update, reset-password(+login), dup username, delete');
  let sid;
  {
    const c = await api('/staff', { method: 'POST', token: AT, body: { name: 'Staf Tes', username: `staf${TAG}`, password: 'awal123', role: 'cashier', status: 'active' } });
    ok(c.status === 201, 'create staff 201');
    sid = c.json.data.id;
    const upd = await api(`/staff/${sid}`, { method: 'PUT', token: AT, body: { name: 'Staf Tes 2', username: `staf${TAG}`, role: 'supervisor', status: 'active' } });
    ok(upd.status === 200, 'update staff 200');
    const rp = await api(`/staff/${sid}/reset-password`, { method: 'POST', token: AT, body: { password: 'baru123' } });
    ok(rp.status === 204 || rp.status === 200, 'staff reset-password 204');
    const relog = await staffLogin({ username: `staf${TAG}`, password: 'baru123' });
    ok(relog.status === 200, 'staff can log in with the reset password');
    const dup = await api('/staff', { method: 'POST', token: AT, body: { name: 'd', username: `staf${TAG}`, password: 'awal123', role: 'cashier', status: 'active' } });
    ok(dup.status === 409, 'duplicate staff username 409');
  }

  // ===================== ADMIN USERS CRUD =====================
  section('ADMIN-USERS: create(manager), login-by-username, reset-password(+login), dup email, delete');
  let auId; const mgrEmail = `mgr${TAG}@elk.id`; const mgrUsername = `mgr${TAG}`;
  {
    const c = await api('/admin-users', { method: 'POST', token: AT, body: { name: 'Manajer Tes', email: mgrEmail, username: mgrUsername, password: 'awal123', role: 'manager', status: 'active' } });
    ok(c.status === 201, 'create admin-user (manager) 201');
    auId = c.json.data.id;
    const byUsername = await api('/auth/admin/login', { method: 'POST', body: { email: mgrUsername, password: 'awal123' } });
    ok(byUsername.status === 200, 'admin-user can log in with username');
    const rp = await api(`/admin-users/${auId}/reset-password`, { method: 'POST', token: AT, body: { password: 'baru123' } });
    ok(rp.status === 204 || rp.status === 200, 'admin-user reset-password 204');
    const relog = await api('/auth/admin/login', { method: 'POST', body: { email: mgrEmail, password: 'baru123' } });
    ok(relog.status === 200, 'admin-user can log in with the reset password');
    const dup = await api('/admin-users', { method: 'POST', token: AT, body: { name: 'd', email: mgrEmail, username: `${mgrUsername}x`, password: 'awal123', role: 'admin', status: 'active' } });
    ok(dup.status === 409, 'duplicate admin email 409');
  }
  const mgrToken = (await api('/auth/admin/login', { method: 'POST', body: { email: mgrEmail, password: 'baru123' } })).json?.data?.accessToken;

  // ===================== WITHDRAWALS =====================
  section('WITHDRAWALS: owner create, list, validation, role/actor guards');
  {
    const c = await api('/withdrawals', { method: 'POST', token: AT, body: { amount: 500000, bank: 'BCA', account: '123456', holder: 'Adi' } });
    ok(c.status === 201, 'owner creates withdrawal 201');
    const list = await api('/withdrawals', { token: AT });
    ok(list.status === 200 && list.json.data.length >= 1, 'withdrawals list 200');
    const bad = await api('/withdrawals', { method: 'POST', token: AT, body: { amount: 0, bank: 'BCA', account: '1', holder: 'A' } });
    ok(bad.status === 400, 'withdrawal amount<=0 rejected (400)');
    const byMgr = await api('/withdrawals', { method: 'POST', token: mgrToken, body: { amount: 1000, bank: 'BCA', account: '1', holder: 'A' } });
    ok(byMgr.status === 403, 'non-owner admin forbidden from withdrawal (403)');
    const byStaff = await api('/withdrawals', { method: 'POST', token: CT, body: { amount: 1000, bank: 'BCA', account: '1', holder: 'A' } });
    ok(byStaff.status === 403, 'staff actor forbidden from withdrawal (403)');
  }

  // ===================== SHIFT #1 + TRANSACTIONS + CASH MOVEMENTS =====================
  section('SHIFT #1 (cashier): open + sales + cash-movements');
  let shift1;
  {
    // ensure no stale open shift
    const cur0 = await api('/shifts/current', { token: CT });
    if (cur0.status === 200 && cur0.json?.data?.id) {
      await api(`/shifts/${cur0.json.data.id}/close`, { method: 'POST', token: CT, body: { actualCash: 0, drawerOpenCount: 0, closeApprovedBy: 'reset' } });
    }
    const open = await api('/shifts', { method: 'POST', token: CT, body: { initialCash: 100000 } });
    ok(open.status === 201, 'cashier opens shift 201');
    shift1 = open.json.data.id;
    const dupOpen = await api('/shifts', { method: 'POST', token: CT, body: { initialCash: 50000 } });
    ok(dupOpen.status === 409, 'opening a second shift while one is open 409');
    const cur = await api('/shifts/current', { token: CT });
    ok(cur.status === 200 && cur.json.data.id === shift1, 'current shift resolves the open one');
  }

  section('TRANSACTIONS: cash, qris(change=0), discount gate, idempotency, stock, validation');
  {
    // cash sale qty1 (12000) — feeds reports + shift cash sales
    const cash = await api('/transactions', { method: 'POST', token: CT, idem: `cash-${TAG}`, body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'cash', amountReceived: 20000, orderType: 'dineIn' } });
    ok(cash.status === 201 && cash.json.data.changeAmount === 8000, 'cash sale 201, change=8000');
    // idempotent replay
    const replay = await api('/transactions', { method: 'POST', token: CT, idem: `cash-${TAG}`, body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'cash', amountReceived: 20000, orderType: 'dineIn' } });
    ok(replay.status === 200 && replay.json.data.id === cash.json.data.id, 'idempotent replay returns same tx (200)');
    // same key, different body -> 409
    const conflict = await api('/transactions', { method: 'POST', token: CT, idem: `cash-${TAG}`, body: { items: [{ productId: prod.id, quantity: 2 }], paymentMethod: 'cash', amountReceived: 30000, orderType: 'dineIn' } });
    ok(conflict.status === 409, 'same idem key + different body 409');
    // missing idempotency key
    const noIdem = await api('/transactions', { method: 'POST', token: CT, body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'cash', amountReceived: 20000, orderType: 'dineIn' } });
    ok(noIdem.status === 400, 'missing Idempotency-Key 400');
    // qris sale -> change 0
    const qris = await api('/transactions', { method: 'POST', token: CT, idem: `qris-${TAG}`, body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'qris', amountReceived: 0, orderType: 'dineIn' } });
    ok(qris.status === 201 && qris.json.data.changeAmount === 0 && qris.json.data.total === 12000, 'qris sale 201, change=0');
    // cash short
    const short = await api('/transactions', { method: 'POST', token: CT, idem: `short-${TAG}`, body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'cash', amountReceived: 1000, orderType: 'dineIn' } });
    ok(short.status === 400, 'cash amountReceived < total 400');
    // discount over policy (12000 -> cap 1200) without approver
    const overDisc = await api('/transactions', { method: 'POST', token: CT, idem: `disc1-${TAG}`, body: { items: [{ productId: prod.id, quantity: 1 }], discount: 5000, paymentMethod: 'cash', amountReceived: 20000, orderType: 'dineIn' } });
    ok(overDisc.status === 403, 'over-policy discount without approver 403');
    // with approver
    const okDisc = await api('/transactions', { method: 'POST', token: CT, idem: `disc2-${TAG}`, body: { items: [{ productId: prod.id, quantity: 1 }], discount: 5000, discountApprovedBy: 'Supervisor Uji', paymentMethod: 'cash', amountReceived: 20000, orderType: 'dineIn' } });
    ok(okDisc.status === 201 && okDisc.json.data.total === 7000, 'over-policy discount WITH approver 201, total=7000');
    // insufficient stock (huge qty)
    const huge = await api('/transactions', { method: 'POST', token: CT, idem: `huge-${TAG}`, body: { items: [{ productId: prod.id, quantity: 999999 }], paymentMethod: 'cash', amountReceived: 999999999999, orderType: 'dineIn' } });
    ok(huge.status === 422, 'insufficient stock 422');
    // list + get
    const list = await api('/transactions?source=cashier&paymentMethod=cash', { token: AT });
    ok(list.status === 200 && list.json.data.length >= 1, 'transactions list filterable 200');
    const get = await api(`/transactions/${cash.json.data.id}`, { token: AT });
    ok(get.status === 200 && get.json.data.items.length >= 1, 'get transaction with items 200');
  }

  section('CASH MOVEMENTS: capital, expense plafond gate, adjustment, list');
  {
    const cap = await api('/cash-movements', { method: 'POST', token: CT, body: { type: 'capital', amount: 50000, notes: 'tambahan' } });
    ok(cap.status === 201, 'capital movement 201');
    const overExp = await api('/cash-movements', { method: 'POST', token: CT, body: { type: 'expense', amount: 250000, notes: 'beli besar' } });
    ok(overExp.status === 403, 'expense over plafond without approver 403');
    const okExp = await api('/cash-movements', { method: 'POST', token: CT, body: { type: 'expense', amount: 20000, notes: 'beli kecil', approvedBy: 'Supervisor Uji' } });
    ok(okExp.status === 201, 'expense within/with-approval 201');
    const adj = await api('/cash-movements', { method: 'POST', token: CT, body: { type: 'adjustment', amount: -3000, notes: 'koreksi' } });
    ok(adj.status === 201, 'negative adjustment 201');
    const list = await api('/cash-movements', { token: CT });
    ok(list.status === 200 && list.json.data.length >= 3, 'cash-movements list 200');
  }

  // ===================== REPORTS (data now exists) =====================
  section('REPORTS: dashboard/sales/top-products/by-category/payment-dist/staff-perf');
  {
    const dash = await api('/reports/dashboard', { token: AT });
    ok(dash.status === 200 && dash.json.data.summary && dash.json.data.summary.txCount >= 1, 'dashboard summary has txns');
    const sales = await api('/reports/sales', { token: AT });
    ok(sales.status === 200 && Array.isArray(sales.json.data), 'sales report array');
    const top = await api('/reports/top-products', { token: AT });
    ok(top.status === 200 && top.json.data.some((r) => r.productName === 'Kopi Uji'), 'top-products includes Kopi Uji');
    const cat = await api('/reports/sales-by-category', { token: AT });
    ok(cat.status === 200 && Array.isArray(cat.json.data), 'sales-by-category array');
    const pay = await api('/reports/payment-distribution', { token: AT });
    ok(pay.status === 200 && pay.json.data.cashTotal >= 1 && pay.json.data.qrisTotal >= 1, 'payment-distribution has cash+qris');
    const staff = await api('/reports/staff-performance', { token: AT });
    ok(staff.status === 200 && Array.isArray(staff.json.data), 'staff-performance array');
  }

  // ===================== SHIFT CLOSE (variance + approval) =====================
  section('SHIFTS: close #1 (bypass), open #2, variance gate, approval, re-close, current=204');
  {
    const close1 = await api(`/shifts/${shift1}/close`, { method: 'POST', token: CT, body: { actualCash: 100000, drawerOpenCount: 1, closeApprovedBy: 'Supervisor Uji' } });
    ok(close1.status === 200 && close1.json.data.cashSales >= 12000, 'close shift #1 200 (cashSales aggregated)');
    const shift2 = (await api('/shifts', { method: 'POST', token: CT, body: { initialCash: 100000 } })).json.data.id;
    ok(!!shift2, 'open shift #2 after #1 closed');
    const overVar = await api(`/shifts/${shift2}/close`, { method: 'POST', token: CT, body: { actualCash: 80000, drawerOpenCount: 0 } });
    ok(overVar.status === 403, 'variance > tolerance without approval 403');
    const okVar = await api(`/shifts/${shift2}/close`, { method: 'POST', token: CT, body: { actualCash: 80000, drawerOpenCount: 0, closeApprovedBy: 'Supervisor Uji' } });
    ok(okVar.status === 200 && okVar.json.data.variance === -20000, 'close with approval 200, variance=-20000');
    const reclose = await api(`/shifts/${shift2}/close`, { method: 'POST', token: CT, body: { actualCash: 80000, drawerOpenCount: 0, closeApprovedBy: 'x' } });
    ok(reclose.status === 409, 'closing an already-closed shift 409');
    const cur = await api('/shifts/current', { token: CT });
    ok(cur.status === 204, 'no open shift -> current 204');
  }

  // ===================== SELF-ORDER admin ops (PATCH status) =====================
  section('SELF-ORDER admin: place -> PATCH status transitions + edges');
  {
    const placed = await api(`/public/order/MEJA-UJI`, { method: 'POST', body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'cash' } });
    const soId = placed.json.data.order.id;
    const toPrep = await api(`/self-orders/${soId}/status`, { method: 'PATCH', token: CT, body: { status: 'preparing' } });
    ok(toPrep.status === 200 && toPrep.json.data.status === 'preparing', 'PATCH status -> preparing 200');
    const toDone = await api(`/self-orders/${soId}/status`, { method: 'PATCH', token: CT, body: { status: 'completed' } });
    ok(toDone.status === 200 && toDone.json.data.status === 'completed', 'PATCH status -> completed 200');
    const bad = await api(`/self-orders/${soId}/status`, { method: 'PATCH', token: CT, body: { status: 'flying' } });
    ok(bad.status === 400, 'invalid status 400');
    const missing = await api(`/self-orders/ZZZNOPE/status`, { method: 'PATCH', token: CT, body: { status: 'preparing' } });
    ok(missing.status === 404, 'PATCH status on unknown order 404');
  }

  // ===================== WEBHOOK (Xendit production path) =====================
  section('WEBHOOK: token guard + paid fulfilment + idempotency');
  {
    const placed = await api(`/public/order/MEJA-UJI`, { method: 'POST', body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'qris' } });
    const soId = placed.json.data.order.id;
    const stock0 = (await api(`/products/${prod.id}`, { token: AT })).json.data.stock;

    const wrongTok = await api('/webhooks/xendit', { method: 'POST', headers: { 'x-callback-token': 'salah' }, body: { id: `evt-${TAG}-1`, event: 'qr.payment', data: { reference_id: soId, status: 'PAID', amount: 12000 } } });
    ok(wrongTok.status === 401, 'webhook with wrong token 401');

    const evtId = `evt-${TAG}-ok`;
    const good = await api('/webhooks/xendit', { method: 'POST', headers: { 'x-callback-token': WEBHOOK_TOKEN }, body: { id: evtId, event: 'qr.payment', data: { reference_id: soId, status: 'PAID', amount: 12000 } } });
    ok(good.status === 200, 'webhook with valid token + paid event 200');
    const st = await api(`/public/order/status/${soId}`);
    ok(st.json.data.paymentStatus === 'paid', 'webhook marked order paid');
    const stock1 = (await api(`/products/${prod.id}`, { token: AT })).json.data.stock;
    ok(stock1 === stock0 - 1, `webhook fulfilment decremented stock (${stock0} -> ${stock1})`);

    const dupEvt = await api('/webhooks/xendit', { method: 'POST', headers: { 'x-callback-token': WEBHOOK_TOKEN }, body: { id: evtId, event: 'qr.payment', data: { reference_id: soId, status: 'PAID', amount: 12000 } } });
    ok(dupEvt.status === 200, 'duplicate webhook event accepted (200)');
    const stock2 = (await api(`/products/${prod.id}`, { token: AT })).json.data.stock;
    ok(stock2 === stock1, 'duplicate webhook event does NOT double-decrement (idempotent)');
  }

  // ===================== MULTI-TENANCY isolation =====================
  if (STORE_B_PRODUCT_ID) {
    section('MULTI-TENANCY: store A admin cannot see store B data');
    const cross = await api(`/products/${STORE_B_PRODUCT_ID}`, { token: AT });
    ok(cross.status === 404, 'GET store B product by id is 404 for store A admin');
    const list = await api('/products', { token: AT });
    ok(!list.json.data.some((p) => p.id === STORE_B_PRODUCT_ID), 'store B product absent from store A product list');
  } else {
    console.log('\n(skip multi-tenancy: STORE_B_PRODUCT_ID not provided)');
  }

  // ===================== cleanup the CRUD entities created via API =====================
  section('CLEANUP (API-level entities created by this run)');
  for (const [label, path] of [
    ['staff', `/staff/${sid}`], ['admin-user', `/admin-users/${auId}`],
    ['product', `/products/${pid}`], ['table', `/tables/${tid}`], ['category', `/categories/${catId}`],
  ]) {
    const d = await api(path, { method: 'DELETE', token: AT });
    ok(d.status === 204 || d.status === 200, `delete ${label} 204`);
  }

  console.log(`\n=== ${pass} passed, ${fail} failed ===`);
  if (fail) console.log('FAILURES:\n - ' + fails.join('\n - '));
  process.exit(fail === 0 ? 0 : 1);
})().catch((e) => { console.error('FATAL', e); process.exit(2); });

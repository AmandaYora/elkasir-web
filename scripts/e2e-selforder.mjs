// Real-usage E2E for the SELF-ORDER scenarios (the customer-facing flows that had
// no automated coverage). Drives the live API exactly as the customer's browser and
// the cashier client do:
//   A) Dine-in, scan table QR → order → pay QRIS at the table (dev simulate)
//   B) Dine-in, scan table QR → order → pay at the cashier via claim-code (barcode)
//   + input-validation / not-found edge cases a real user can hit.
//
// Run: node scripts/e2e-selforder.mjs   (API must be up; fixtures seeded)
const BASE = process.env.API_BASE_URL || 'http://localhost:8081/api/v1';
const ADMIN = { email: process.env.ADMIN_EMAIL || 'admin', password: process.env.ADMIN_PASS || 'admin123' };
const CASHIER = { username: 'kasiruji', password: 'kasir123' };
const TABLE_CODE = 'MEJA-UJI';
const PRODUCT_NAME = 'Kopi Uji';

let pass = 0, fail = 0;
function ok(cond, label) {
  if (cond) { pass++; console.log(`  ✓ ${label}`); }
  else { fail++; console.log(`  ✗ FAIL: ${label}`); }
}
function section(t) { console.log(`\n── ${t} ──`); }

async function api(path, { method = 'GET', token, body, idem } = {}) {
  const res = await fetch(BASE + path, {
    method,
    headers: {
      'content-type': 'application/json',
      ...(token ? { authorization: `Bearer ${token}` } : {}),
      ...(idem ? { 'Idempotency-Key': idem } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  let json = null;
  try { json = await res.json(); } catch { /* 204 / empty */ }
  return { status: res.status, json };
}

async function adminLogin() {
  const r = await api('/auth/admin/login', { method: 'POST', body: ADMIN });
  if (r.status !== 200) throw new Error(`admin login failed [${r.status}] ${JSON.stringify(r.json)}`);
  return r.json.data.accessToken;
}
async function staffLogin() {
  const r = await api('/auth/staff/login', { method: 'POST', body: CASHIER });
  if (r.status !== 200) throw new Error(`staff login failed [${r.status}] ${JSON.stringify(r.json)}`);
  return r.json.data.accessToken;
}
async function productStock(adminToken, productId) {
  const r = await api(`/products/${productId}`, { token: adminToken });
  if (r.status !== 200) throw new Error(`get product failed [${r.status}] ${JSON.stringify(r.json)}`);
  return r.json.data.stock;
}

(async () => {
  console.log(`Self-order E2E @ ${BASE}`);
  const adminToken = await adminLogin();
  const cashierToken = await staffLogin();

  // Discover the menu the way a customer does (scan QR → GET /public/order/{code}).
  const menu = await api(`/public/order/${TABLE_CODE}`);
  ok(menu.status === 200, 'customer fetches table menu by QR code (200)');
  const prod = menu.json?.data?.products?.find((p) => p.name === PRODUCT_NAME);
  ok(!!prod, `menu lists product "${PRODUCT_NAME}"`);
  ok(menu.json?.data?.table?.code === TABLE_CODE, 'menu carries the scanned table');
  if (!prod) { console.log('\nCannot continue without the product fixture.'); process.exit(1); }
  const price = prod.price; // 12000

  // ============ Scenario A: order from table → pay QRIS at the table ============
  section('A) Dine-in → QRIS pay-at-table (dev simulate)');
  const stockA0 = await productStock(adminToken, prod.id);

  const placeA = await api(`/public/order/${TABLE_CODE}`, {
    method: 'POST',
    body: { items: [{ productId: prod.id, quantity: 2 }], paymentMethod: 'qris', customerNote: 'tanpa gula' },
  });
  ok(placeA.status === 201, 'place QRIS order (201)');
  const orderA = placeA.json?.data?.order;
  ok(orderA?.paymentMethod === 'qris', 'order recorded as qris');
  ok(orderA?.paymentStatus === 'pending', 'qris order starts pending');
  // Subtotal = harga barang; total = subtotal + layanan(+gateway QRIS) + PPN (≥ subtotal).
  ok(orderA?.subtotal === price * 2, `subtotal = price*2 (${price * 2})`);
  ok(orderA?.total >= orderA?.subtotal, `total (${orderA?.total}) ≥ subtotal (${orderA?.subtotal})`);
  ok(orderA?.total === orderA?.subtotal + orderA?.serviceLine + orderA?.tax,
    'total = subtotal + layanan + PPN (breakdown konsisten)');
  // Payment-init contract: a live gateway (Tripay/Midtrans) returns a qrImageUrl (QRIS image);
  // in dev (gateway not configured) it returns simulated=true and the frontend renders a
  // fallback QR (elkasir:order:<id>) + a "mark paid" button. Either way the table can pay.
  const payInit = placeA.json?.data;
  ok(payInit?.simulated === true || !!payInit?.qrImageUrl || !!payInit?.qrString,
    `payment init is payable at the table (simulated=${payInit?.simulated}, qr=${payInit?.qrImageUrl || payInit?.qrString ? 'present' : 'empty'})`);

  const statusA1 = await api(`/public/order/status/${orderA.id}`);
  ok(statusA1.json?.data?.paymentStatus === 'pending', 'status poll shows pending before payment');

  const sim = await api(`/public/order/${orderA.id}/simulate-paid`, { method: 'POST' });
  ok(sim.status === 200, 'simulate-paid succeeds (dev gateway)');

  const statusA2 = await api(`/public/order/status/${orderA.id}`);
  ok(statusA2.json?.data?.paymentStatus === 'paid', 'status flips to paid after payment');
  ok(statusA2.json?.data?.status === 'preparing', 'paid QRIS order moves to preparing (kitchen)');

  const stockA1 = await productStock(adminToken, prod.id);
  ok(stockA1 === stockA0 - 2, `stock decremented by 2 on payment (${stockA0} → ${stockA1})`);

  // Idempotency: a second simulate/webhook must not double-fulfil.
  const simAgain = await api(`/public/order/${orderA.id}/simulate-paid`, { method: 'POST' });
  ok(simAgain.status === 200, 'repeat simulate-paid is accepted');
  const stockA2 = await productStock(adminToken, prod.id);
  ok(stockA2 === stockA1, 'repeat payment does NOT decrement stock again (idempotent)');

  // The paid QRIS sale is linked to a real transaction (verified via staff list).
  const incomingA = await api(`/self-orders?status=preparing`, { token: cashierToken });
  const seenA = incomingA.json?.data?.find((o) => o.id === orderA.id);
  ok(!!seenA?.transactionId, 'paid QRIS order is linked to a transaction id');

  // ============ Scenario B: order from table → pay at cashier (claim code) ======
  section('B) Dine-in → pay-at-cashier via claim code (barcode)');
  const stockB0 = await productStock(adminToken, prod.id);

  const placeB = await api(`/public/order/${TABLE_CODE}`, {
    method: 'POST',
    body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'cash' },
  });
  ok(placeB.status === 201, 'place cash order (201)');
  const orderB = placeB.json?.data?.order;
  const claim = placeB.json?.data?.claimCode;
  ok(!!claim, 'a claim code (barcode) is issued for the cashier');
  ok(orderB?.paymentStatus === 'unpaid', 'cash order starts unpaid');

  // Cashier scans the barcode → looks the order up, then settles it.
  const lookup = await api(`/self-orders/redeem/${claim}`, { token: cashierToken });
  ok(lookup.status === 200, 'cashier looks up the order by claim code (200)');
  ok(lookup.json?.data?.id === orderB.id, 'lookup resolves the right order');

  const idemB = `e2e-redeem-${orderB.id}`;
  const checkout = await api(`/self-orders/redeem/${claim}/checkout`, { method: 'POST', token: cashierToken, idem: idemB });
  ok(checkout.status === 200, 'cashier checks out the cash order (200)');
  const txId = checkout.json?.data?.transactionId;
  ok(!!txId, 'checkout returns a transaction id');

  const stockB1 = await productStock(adminToken, prod.id);
  ok(stockB1 === stockB0 - 1, `stock decremented by 1 on checkout (${stockB0} → ${stockB1})`);

  const statusB = await api(`/public/order/status/${orderB.id}`);
  ok(statusB.json?.data?.paymentStatus === 'paid', 'cash order becomes paid after checkout');

  // Idempotent replay: scanning/settling the same barcode again returns the same sale.
  const checkout2 = await api(`/self-orders/redeem/${claim}/checkout`, { method: 'POST', token: cashierToken, idem: idemB });
  ok(checkout2.status === 200, 'repeat checkout is accepted (replay)');
  ok(checkout2.json?.data?.transactionId === txId, 'replay returns the SAME transaction id');
  const stockB2 = await productStock(adminToken, prod.id);
  ok(stockB2 === stockB1, 'repeat checkout does NOT decrement stock again (idempotent)');

  // ============ Edge cases a real user can trigger ============================
  section('C) Validation & not-found edges');
  const emptyOrder = await api(`/public/order/${TABLE_CODE}`, { method: 'POST', body: { items: [], paymentMethod: 'cash' } });
  ok(emptyOrder.status === 400, 'empty order rejected (400)');

  const badMethod = await api(`/public/order/${TABLE_CODE}`, { method: 'POST', body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'gopay' } });
  ok(badMethod.status === 400, 'invalid payment method rejected (400)');

  const zeroQty = await api(`/public/order/${TABLE_CODE}`, { method: 'POST', body: { items: [{ productId: prod.id, quantity: 0 }], paymentMethod: 'cash' } });
  ok(zeroQty.status === 400, 'zero quantity rejected (400)');

  const noTableMenu = await api(`/public/order/MEJA-TIDAK-ADA`);
  ok(noTableMenu.status === 404, 'menu for unknown table is 404');

  const noTablePlace = await api(`/public/order/MEJA-TIDAK-ADA`, { method: 'POST', body: { items: [{ productId: prod.id, quantity: 1 }], paymentMethod: 'cash' } });
  ok(noTablePlace.status === 404, 'order to unknown table is 404');

  const badClaim = await api(`/self-orders/redeem/ELK-NOPE-00000`, { token: cashierToken });
  ok(badClaim.status === 404, 'unknown claim code is 404');

  const simCash = await api(`/public/order/${orderB.id}/simulate-paid`, { method: 'POST' });
  ok(simCash.status === 400, 'simulate-paid on a cash order is rejected (400)');

  const redeemUnauth = await api(`/self-orders/redeem/${claim}`);
  ok(redeemUnauth.status === 401, 'redeem without auth is 401');

  // ============ Summary =======================================================
  console.log(`\n=== ${pass} passed, ${fail} failed ===`);
  // Emit the created ids so the caller can verify / clean up precisely.
  console.log(`CREATED_ORDERS=${[orderA?.id, orderB?.id].filter(Boolean).join(',')}`);
  process.exit(fail === 0 ? 0 : 1);
})().catch((e) => { console.error('FATAL', e); process.exit(2); });

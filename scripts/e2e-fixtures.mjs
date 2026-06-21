// One-off E2E fixture seeder for the Flutter POS UI end-to-end tests.
// Creates (idempotently) the cashier/supervisor/product/table that
// elkasir_mobile/test/ui_flow_test.dart & supervisor_flow_test.dart expect.
const BASE = process.env.API_BASE_URL || 'http://localhost:8081/api/v1';
const ADMIN_EMAIL = process.env.ADMIN_EMAIL || 'admin';
const ADMIN_PASS = process.env.ADMIN_PASS || 'admin123';

async function api(path, { method = 'GET', token, body } = {}) {
  const res = await fetch(BASE + path, {
    method,
    headers: {
      'content-type': 'application/json',
      ...(token ? { authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  let json = null;
  try { json = await res.json(); } catch { /* 204 */ }
  return { status: res.status, json };
}

async function ensure(label, path, token, body) {
  const r = await api(path, { method: 'POST', token, body });
  if (r.status === 201 || r.status === 200) {
    console.log(`  ✓ created ${label}`);
  } else if (r.status === 409) {
    console.log(`  • ${label} already exists (409) — ok`);
  } else {
    console.log(`  ✗ ${label} FAILED [${r.status}] ${JSON.stringify(r.json)}`);
    process.exitCode = 1;
  }
}

(async () => {
  console.log(`Admin login as "${ADMIN_EMAIL}" @ ${BASE}`);
  const login = await api('/auth/admin/login', {
    method: 'POST',
    body: { email: ADMIN_EMAIL, password: ADMIN_PASS },
  });
  if (login.status !== 200) {
    console.error(`Admin login failed [${login.status}]: ${JSON.stringify(login.json)}`);
    process.exit(1);
  }
  const token = login.json.data.accessToken;
  console.log('  ✓ admin authenticated\n');

  console.log('Seeding fixtures:');
  await ensure('product "Kopi Uji" @12000', '/products', token, {
    sku: 'KOPI-UJI', name: 'Kopi Uji', price: 12000, cost: 0, stock: 1000, status: 'active',
  });
  await ensure('table MEJA-UJI', '/tables', token, {
    code: 'MEJA-UJI', name: 'Meja Uji', area: 'Indoor', seats: 4, status: 'active',
  });
  await ensure('cashier kasiruji', '/staff', token, {
    name: 'Kasir Uji', username: 'kasiruji', password: 'kasir123', role: 'cashier', status: 'active',
  });
  await ensure('supervisor supuji (Supervisor Uji)', '/staff', token, {
    name: 'Supervisor Uji', username: 'supuji', password: 'super123', role: 'supervisor', status: 'active',
  });
  console.log('\nDone.');
})();

import re
from playwright.sync_api import sync_playwright, expect

BASE = "http://localhost:8081"
SHOT = "/tmp"
npass = 0; nfail = 0; fails = []
def ok(cond, label):
    global npass, nfail
    if cond: npass += 1; print(f"  PASS  {label}")
    else: nfail += 1; fails.append(label); print(f"  FAIL  {label}")

def add_and_review(page):
    page.goto(f"{BASE}/order/MEJA-UJI"); page.wait_for_load_state("networkidle"); page.wait_for_timeout(400)
    page.get_by_role("button", name="Tambah").first.click(); page.wait_for_timeout(300)
    page.get_by_role("button", name="Lanjut").first.click(); page.wait_for_timeout(500)

with sync_playwright() as p:
    b = p.chromium.launch(headless=True)
    ctx = b.new_context()

    # ============ SCENARIO 1: customer self-order — QRIS pay-at-table ============
    print("\n== Customer self-order: QRIS pay-at-table ==")
    pg = ctx.new_page()
    pg.goto(f"{BASE}/order/MEJA-UJI"); pg.wait_for_load_state("networkidle"); pg.wait_for_timeout(400)
    ok("Meja Uji" in pg.inner_text("body"), "menu shows scanned table 'Meja Uji'")
    ok("Kopi Uji" in pg.inner_text("body"), "menu lists product 'Kopi Uji'")
    add_and_review(pg)
    ok("Total" in pg.inner_text("body") and "12.000" in pg.inner_text("body"), "review shows total Rp 12.000")
    pg.get_by_role("button").filter(has_text="Bayar QRIS").first.click(); pg.wait_for_timeout(800)
    body = pg.inner_text("body")
    ok("Pembayaran QRIS" in body or "Menunggu pembayaran" in body, "QRIS panel shown (QR + waiting)")
    ok(pg.locator("svg").count() > 0, "a QR code is rendered")
    pg.screenshot(path=f"{SHOT}/pw_qris.png", full_page=True)
    # dev gateway → 'Tandai sudah dibayar' simulate button
    sim = pg.get_by_role("button", name=re.compile("Tandai sudah dibayar"))
    ok(sim.count() > 0, "dev simulate button 'Tandai sudah dibayar' present")
    if sim.count() > 0:
        sim.first.click()
        pg.wait_for_timeout(1500)
        ok("Pembayaran berhasil" in pg.inner_text("body"), "after paying, panel shows 'Pembayaran berhasil'")
        pg.screenshot(path=f"{SHOT}/pw_qris_paid.png", full_page=True)
    pg.close()

    # ============ SCENARIO 2: customer self-order — pay at cashier (claim code) ====
    print("\n== Customer self-order: pay-at-cashier (claim code) ==")
    pg = ctx.new_page()
    add_and_review(pg)
    pg.get_by_role("button").filter(has_text="Bayar di kasir").first.click(); pg.wait_for_timeout(900)
    body = pg.inner_text("body")
    ok("Bayar di kasir" in body, "cashier panel shown")
    ok(re.search(r"ELK-", body) is not None, "a claim code (ELK-...) is displayed")
    ok(pg.locator("svg").count() > 0, "a barcode is rendered for the cashier scanner")
    pg.screenshot(path=f"{SHOT}/pw_cashier.png", full_page=True)
    pg.close()

    # ============ SCENARIO 3: admin login + authenticated data views ============
    print("\n== Admin login + authenticated views ==")
    pg = ctx.new_page()
    pg.goto(f"{BASE}/login"); pg.wait_for_load_state("networkidle"); pg.wait_for_timeout(400)
    pg.locator("input[type=text]").first.fill("admin")
    pg.locator("input[type=password]").first.fill("admin123")
    pg.get_by_role("button", name=re.compile("Masuk ke Dasbor")).click()
    pg.wait_for_timeout(1800)
    ok(not pg.url.rstrip("/").endswith("/login"), f"login redirects away from /login (now {pg.url})")
    pg.screenshot(path=f"{SHOT}/pw_dashboard.png", full_page=True)

    # products page should list the seeded product via an authenticated API call
    pg.goto(f"{BASE}/products"); pg.wait_for_load_state("networkidle"); pg.wait_for_timeout(1200)
    ok("Kopi Uji" in pg.inner_text("body"), "admin Products page lists 'Kopi Uji' (authenticated API works)")
    pg.screenshot(path=f"{SHOT}/pw_products.png", full_page=True)

    # tables page should list the seeded table
    pg.goto(f"{BASE}/cashiers"); pg.wait_for_load_state("networkidle"); pg.wait_for_timeout(1000)
    ok("kasiruji" in pg.inner_text("body"), "admin Staff page lists 'kasiruji'")
    pg.close()

    # ============ SCENARIO 4: admin login failure ============
    # Fresh context: scenario 3 stored a valid session in `ctx`, which would auto-redirect
    # /login to the dashboard. A clean (logged-out) context is the correct precondition.
    print("\n== Admin login failure (wrong password) ==")
    ctx2 = b.new_context()
    pg = ctx2.new_page()
    pg.goto(f"{BASE}/login"); pg.wait_for_load_state("networkidle"); pg.wait_for_timeout(400)
    pg.locator("input[type=text]").first.fill("admin")
    pg.locator("input[type=password]").first.fill("salah-password")
    pg.get_by_role("button", name=re.compile("Masuk ke Dasbor")).click()
    pg.wait_for_timeout(1500)
    ok(pg.url.rstrip("/").endswith("/login"), "wrong password keeps user on /login")
    pg.close()

    b.close()

print(f"\n=== {npass} passed, {nfail} failed ===")
if fails:
    print("FAILURES:\n - " + "\n - ".join(fails))
import sys; sys.exit(1 if nfail else 0)

// Package application holds the selforder module's use cases — the cross-module
// self-order orchestrator (customer self-order + QRIS payment + pay-at-cashier via claim
// code). All access to other modules goes through their contracts (productclient,
// salesclient, shiftclient, tableclient, paymentclient); atomic fulfilment is wrapped in
// uow.Run. The repo touches ONLY selforder-owned tables.
package application

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	productclient "github.com/elkasir/api/internal/modules/product/contracts"
	"github.com/elkasir/api/internal/modules/selforder/domain"
	"github.com/elkasir/api/internal/modules/selforder/infrastructure"
	shiftclient "github.com/elkasir/api/internal/modules/shift/contracts"
	tableclient "github.com/elkasir/api/internal/modules/table/contracts"
	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/uow"
)

const orderTTL = 30 * time.Minute

// Service adalah orchestrator "Order" (self-order). Seluruh akses lintas-modul melewati
// contract: productclient, salesclient, shiftclient, tableclient, paymentclient — tidak
// menyentuh tabel modul lain. Fulfilment atomik dibungkus uow.Run.
type Service struct {
	repo     *infrastructure.Repo
	products productclient.Client
	sales    salesclient.Client
	shifts   shiftclient.Client
	tables   tableclient.Client
	payments paymentclient.Client
	uow      *uow.Manager
}

func NewService(
	repo *infrastructure.Repo,
	productClient productclient.Client,
	salesClient salesclient.Client,
	shiftClient shiftclient.Client,
	tableClient tableclient.Client,
	paymentClient paymentclient.Client,
	uowMgr *uow.Manager,
) *Service {
	return &Service{repo: repo, products: productClient, sales: salesClient, shifts: shiftClient, tables: tableClient, payments: paymentClient, uow: uowMgr}
}

func (s *Service) PaymentEnabled() bool { return s.payments.Enabled() }

// ── DTO ──────────────────────────────────────────────────────
type ItemDTO struct {
	ProductName string `json:"productName"`
	Category    string `json:"category"`
	Price       int64  `json:"price"`
	Quantity    int32  `json:"quantity"`
	LineTotal   int64  `json:"lineTotal"`
	Note        string `json:"note,omitempty"`
}

type OrderDTO struct {
	ID            string    `json:"id"`
	TableCode     string    `json:"tableCode"`
	TableName     string    `json:"tableName"`
	Status        string    `json:"status"`
	PaymentMethod string    `json:"paymentMethod"`
	PaymentStatus string    `json:"paymentStatus"`
	ClaimCode     string    `json:"claimCode,omitempty"`
	Subtotal      int64     `json:"subtotal"`
	Total         int64     `json:"total"`
	CustomerNote  string    `json:"customerNote,omitempty"`
	TransactionID string    `json:"transactionId,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	Items         []ItemDTO `json:"items"`
}

type TableDTO struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Area   string `json:"area"`
	Status string `json:"status"`
}

type MenuProductDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Price    int64  `json:"price"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type MenuDTO struct {
	Table      TableDTO         `json:"table"`
	Categories []string         `json:"categories"`
	Products   []MenuProductDTO `json:"products"`
}

type PlaceResult struct {
	Order     OrderDTO `json:"order"`
	QRString  string   `json:"qrString,omitempty"`
	ClaimCode string   `json:"claimCode,omitempty"`
	Simulated bool     `json:"simulated,omitempty"`
}

type StatusDTO struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	PaymentStatus string `json:"paymentStatus"`
	Total         int64  `json:"total"`
}

type CheckoutResult struct {
	TransactionID string   `json:"transactionId"`
	Order         OrderDTO `json:"order"`
}

// ── Input ────────────────────────────────────────────────────
type PlaceItem struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
	Note      string `json:"note"`
}

type PlaceInput struct {
	Items         []PlaceItem `json:"items"`
	PaymentMethod string      `json:"paymentMethod"`
	CustomerNote  string      `json:"customerNote"`
}

// ── Menu publik ──────────────────────────────────────────────
func (s *Service) Menu(ctx context.Context, tableCode string) (MenuDTO, error) {
	t, err := s.tables.FindByCode(ctx, tableCode)
	if errors.Is(err, tableclient.ErrNotFound) {
		return MenuDTO{}, httpx.NotFound("Meja tidak dikenali.")
	}
	if err != nil {
		return MenuDTO{}, err
	}
	prods, err := s.products.ListActive(ctx, t.StoreID)
	if err != nil {
		return MenuDTO{}, err
	}

	seen := map[string]bool{}
	cats := []string{}
	items := make([]MenuProductDTO, 0, len(prods))
	for _, p := range prods {
		if p.Category != "" && !seen[p.Category] {
			seen[p.Category] = true
			cats = append(cats, p.Category)
		}
		items = append(items, MenuProductDTO{ID: p.ID, Name: p.Name, Category: p.Category, Price: p.Price, ImageURL: p.ImageURL})
	}
	return MenuDTO{
		Table:      TableDTO{Code: t.Code, Name: t.Name, Area: t.Area, Status: t.Status},
		Categories: cats,
		Products:   items,
	}, nil
}

// ── Buat self-order (Kondisi 2 & 3) ──────────────────────────
func (s *Service) PlaceOrder(ctx context.Context, tableCode string, in PlaceInput) (PlaceResult, error) {
	t, err := s.tables.FindByCode(ctx, tableCode)
	if errors.Is(err, tableclient.ErrNotFound) {
		return PlaceResult{}, httpx.NotFound("Meja tidak dikenali.")
	}
	if err != nil {
		return PlaceResult{}, err
	}
	if t.Status != "active" {
		return PlaceResult{}, httpx.Unprocessable("Meja sedang tidak menerima pesanan.")
	}
	if in.PaymentMethod != "qris" && in.PaymentMethod != "cash" {
		return PlaceResult{}, httpx.Validation("Metode pembayaran harus 'qris' atau 'cash'.")
	}
	if len(in.Items) == 0 {
		return PlaceResult{}, httpx.Validation("Pesanan tidak boleh kosong.")
	}

	// Snapshot harga dari produk (via productclient; total dihitung server).
	var items []domain.OrderItem
	var subtotal int64
	for _, ci := range in.Items {
		if ci.Quantity <= 0 {
			return PlaceResult{}, httpx.Validation("Kuantitas item harus lebih dari 0.")
		}
		pr, err := s.products.GetForSale(ctx, t.StoreID, ci.ProductID)
		if errors.Is(err, productclient.ErrNotFound) {
			return PlaceResult{}, httpx.Validation("Produk tidak ditemukan: " + ci.ProductID)
		}
		if err != nil {
			return PlaceResult{}, err
		}
		if !pr.Active {
			return PlaceResult{}, httpx.Validation("Produk nonaktif: " + pr.Name)
		}
		lt := pr.Price * int64(ci.Quantity)
		subtotal += lt
		items = append(items, domain.OrderItem{ProductID: pr.ID, ProductName: pr.Name, Category: pr.Category, Price: pr.Price, Quantity: ci.Quantity, LineTotal: lt, Note: strings.TrimSpace(ci.Note)})
	}

	soID := id.New()
	tableID := sql.NullString{String: t.ID, Valid: true}
	note := sql.NullString{String: strings.TrimSpace(in.CustomerNote), Valid: strings.TrimSpace(in.CustomerNote) != ""}

	if in.PaymentMethod == "qris" {
		order := sqlcgen.CreateSelfOrderParams{
			ID: soID, StoreID: t.StoreID, TableID: tableID,
			PaymentMethod: sqlcgen.SelfOrdersPaymentMethodQris, PaymentStatus: sqlcgen.SelfOrdersPaymentStatusPending,
			ClaimCode: sql.NullString{}, Subtotal: subtotal, Total: subtotal, CustomerNote: note,
			ExpiresAt: sql.NullTime{Time: time.Now().Add(orderTTL).UTC(), Valid: true},
		}
		if err := s.repo.CreateOrder(ctx, infrastructure.CreateOrderData{Order: order, Items: items}); err != nil {
			return PlaceResult{}, err
		}

		// Buat tagihan QRIS via paymentclient (Xendit/simulasi) + catat payments.
		charge, err := s.payments.CreateCharge(ctx, t.StoreID, soID, subtotal)
		if err != nil {
			return PlaceResult{}, httpx.Internal("Gagal membuat QR pembayaran: " + err.Error())
		}

		dto, err := s.orderDTO(ctx, t.StoreID, soID)
		return PlaceResult{Order: dto, QRString: charge.QRString, Simulated: charge.Simulated}, err
	}

	// Cash → bayar di kasir: claim code untuk barcode.
	claim := genClaimCode(t.Code)
	order := sqlcgen.CreateSelfOrderParams{
		ID: soID, StoreID: t.StoreID, TableID: tableID,
		PaymentMethod: sqlcgen.SelfOrdersPaymentMethodCash, PaymentStatus: sqlcgen.SelfOrdersPaymentStatusUnpaid,
		ClaimCode: sql.NullString{String: claim, Valid: true}, Subtotal: subtotal, Total: subtotal,
		CustomerNote: note, ExpiresAt: sql.NullTime{},
	}
	if err := s.repo.CreateOrder(ctx, infrastructure.CreateOrderData{Order: order, Items: items}); err != nil {
		return PlaceResult{}, err
	}
	dto, err := s.orderDTO(ctx, t.StoreID, soID)
	return PlaceResult{Order: dto, ClaimCode: claim}, err
}

// ── Status polling (publik) ──────────────────────────────────
func (s *Service) Status(ctx context.Context, soID string) (StatusDTO, error) {
	o, err := s.repo.GetByID(ctx, soID)
	if errors.Is(err, sql.ErrNoRows) {
		return StatusDTO{}, httpx.NotFound("Pesanan tidak ditemukan.")
	}
	if err != nil {
		return StatusDTO{}, err
	}
	return StatusDTO{ID: o.ID, Status: string(o.Status), PaymentStatus: string(o.PaymentStatus), Total: o.Total}, nil
}

// SimulatePaid (DEV) menandai self-order QRIS pending menjadi lunas tanpa gateway.
func (s *Service) SimulatePaid(ctx context.Context, soID string) error {
	o, err := s.repo.GetByID(ctx, soID)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.NotFound("Pesanan tidak ditemukan.")
	}
	if err != nil {
		return err
	}
	if o.PaymentMethod != sqlcgen.SelfOrdersPaymentMethodQris {
		return httpx.Validation("Hanya pesanan QRIS yang bisa disimulasikan.")
	}
	if o.PaymentStatus == sqlcgen.SelfOrdersPaymentStatusPaid {
		return nil
	}
	return s.fulfill(ctx, o, "qris", "", sqlcgen.SelfOrdersStatusPreparing, "", "")
}

// ── Webhook Xendit ───────────────────────────────────────────
// HandleWebhook: verifikasi token (paymentclient), idempoten (webhook_events), dan pada
// status lunas → fulfilment (kurangi stok + transaksi). Selalu 200 agar Xendit tak retry,
// kecuali token salah (401) atau kegagalan sementara (500).
func (s *Service) HandleWebhook(ctx context.Context, header http.Header, body []byte) error {
	if !s.payments.VerifyWebhook(header, body) {
		return httpx.Unauthorized("Callback pembayaran tidak terverifikasi.")
	}
	ev, err := s.payments.ParseWebhook(body)
	if errors.Is(err, paymentclient.ErrInvalidPayload) {
		return httpx.BadRequest("Payload webhook tidak valid.")
	}
	if err != nil {
		return err
	}
	if ev.EventID == "" {
		return nil // tak ada identitas event → abaikan
	}
	seen, err := s.payments.WebhookSeen(ctx, ev.EventID)
	if err != nil {
		return err
	}
	if seen {
		return nil // sudah diproses (idempoten)
	}

	if ev.Paid && ev.OrderRef != "" {
		o, err := s.repo.GetByID(ctx, ev.OrderRef)
		if err == nil && o.PaymentStatus == sqlcgen.SelfOrdersPaymentStatusPending &&
			o.PaymentMethod == sqlcgen.SelfOrdersPaymentMethodQris {
			if err := s.fulfill(ctx, o, "qris", "", sqlcgen.SelfOrdersStatusPreparing, "", ""); err != nil {
				return err // gagal sementara → biarkan Xendit retry (belum ditandai seen)
			}
		}
	}
	return s.payments.MarkWebhookSeen(ctx, ev.EventID)
}

// ── Tebus barcode (Kondisi 3, staff) ─────────────────────────
func (s *Service) Redeem(ctx context.Context, storeID, claimCode string) (OrderDTO, error) {
	o, err := s.repo.GetByClaimCode(ctx, storeID, strings.TrimSpace(claimCode))
	if errors.Is(err, sql.ErrNoRows) {
		return OrderDTO{}, httpx.NotFound("Kode klaim tidak ditemukan.")
	}
	if err != nil {
		return OrderDTO{}, err
	}
	return s.orderDTO(ctx, storeID, o.ID)
}

// RedeemCheckout menebus & menyelesaikan pembayaran tunai (idempoten via order state).
func (s *Service) RedeemCheckout(ctx context.Context, p authcontract.Principal, claimCode, idemKey, reqHash string) (CheckoutResult, error) {
	o, err := s.repo.GetByClaimCode(ctx, p.StoreID, strings.TrimSpace(claimCode))
	if errors.Is(err, sql.ErrNoRows) {
		return CheckoutResult{}, httpx.NotFound("Kode klaim tidak ditemukan.")
	}
	if err != nil {
		return CheckoutResult{}, err
	}
	if o.PaymentMethod != sqlcgen.SelfOrdersPaymentMethodCash {
		return CheckoutResult{}, httpx.Validation("Pesanan ini bukan jalur bayar-di-kasir.")
	}
	if o.PaymentStatus == sqlcgen.SelfOrdersPaymentStatusPaid {
		dto, _ := s.orderDTO(ctx, p.StoreID, o.ID)
		return CheckoutResult{TransactionID: o.TransactionID.String, Order: dto}, nil // replay
	}

	cashierID := ""
	if p.Actor == authcontract.ActorStaff {
		cashierID = p.SubjectID
	}
	if err := s.fulfill(ctx, o, "cash", cashierID, sqlcgen.SelfOrdersStatusCompleted, idemKey, reqHash); err != nil {
		return CheckoutResult{}, err
	}
	updated, _ := s.repo.GetByID(ctx, o.ID)
	dto, _ := s.orderDTO(ctx, p.StoreID, o.ID)
	return CheckoutResult{TransactionID: updated.TransactionID.String, Order: dto}, nil
}

// ── Daftar pesanan masuk & status (staff) ────────────────────
func (s *Service) ListIncoming(ctx context.Context, storeID, status string, limit, offset int) ([]OrderDTO, int64, error) {
	rows, total, err := s.repo.ListIncoming(ctx, storeID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	tableMap := s.tableMap(ctx, storeID)
	out := make([]OrderDTO, 0, len(rows))
	for _, o := range rows {
		items, _ := s.repo.Items(ctx, o.ID)
		code, name := tableInfo(tableMap, o.TableID)
		out = append(out, buildOrderDTO(o, items, code, name))
	}
	return out, total, nil
}

func (s *Service) UpdateStatus(ctx context.Context, storeID, soID, status string) (OrderDTO, error) {
	st, ok := parseOrderStatus(status)
	if !ok {
		return OrderDTO{}, httpx.Validation("Status harus 'placed', 'preparing', atau 'completed'.")
	}
	n, err := s.repo.UpdateStatus(ctx, storeID, soID, st)
	if err != nil {
		return OrderDTO{}, err
	}
	if n == 0 {
		return OrderDTO{}, httpx.NotFound("Pesanan tidak ditemukan.")
	}
	return s.orderDTO(ctx, storeID, soID)
}

// ── Internal ─────────────────────────────────────────────────

// fulfill: kurangi stok (productclient) + catat transaksi self_order (salesclient) +
// tautkan/tandai self-order (repo.MarkPaid) — semua dalam SATU transaksi DB via uow.
func (s *Service) fulfill(ctx context.Context, o sqlcgen.SelfOrder, paymentMethod, cashierID string, newStatus sqlcgen.SelfOrdersStatus, idemKey, reqHash string) error {
	items, err := s.repo.Items(ctx, o.ID)
	if err != nil {
		return err
	}
	saleItems := make([]salesclient.SaleItem, 0, len(items))
	for _, it := range items {
		saleItems = append(saleItems, salesclient.SaleItem{
			ProductID: it.ProductID.String, ProductName: it.ProductName, Category: it.Category,
			Price: it.Price, Quantity: it.Quantity, LineTotal: it.LineTotal, Note: it.Note.String,
		})
	}

	shiftID, err := s.shifts.CurrentOpenID(ctx, o.StoreID)
	if err != nil {
		return err
	}
	tableID := ""
	if o.TableID.Valid {
		tableID = o.TableID.String
	}

	err = s.uow.Run(ctx, func(ctx context.Context) error {
		for _, it := range saleItems {
			if derr := s.products.Decrease(ctx, o.StoreID, it.ProductID, it.Quantity); derr != nil {
				if errors.Is(derr, productclient.ErrInsufficientStock) {
					return fmt.Errorf("%w: %s", productclient.ErrInsufficientStock, it.ProductName)
				}
				return derr
			}
		}
		txID, rerr := s.sales.RecordSale(ctx, salesclient.RecordSaleInput{
			StoreID: o.StoreID, Source: "self_order", PaymentMethod: paymentMethod, OrderType: "dineIn",
			TableID: tableID, SelfOrderID: o.ID, CashierID: cashierID, ShiftID: shiftID,
			Items:          saleItems,
			Subtotal:       o.Subtotal,
			Discount:       0,
			Total:          o.Total,
			AmountReceived: o.Total,
			Change:         0,
			IdempotencyKey: idemKey,
			RequestHash:    reqHash,
		})
		if rerr != nil {
			return rerr
		}
		return s.repo.MarkPaid(ctx, o.ID, txID, newStatus)
	})
	if errors.Is(err, productclient.ErrInsufficientStock) {
		return httpx.Unprocessable("Stok tidak cukup (" + strings.TrimPrefix(err.Error(), "stok tidak cukup: ") + ").")
	}
	return err
}

func (s *Service) orderDTO(ctx context.Context, storeID, soID string) (OrderDTO, error) {
	o, err := s.repo.Get(ctx, storeID, soID)
	if errors.Is(err, sql.ErrNoRows) {
		return OrderDTO{}, httpx.NotFound("Pesanan tidak ditemukan.")
	}
	if err != nil {
		return OrderDTO{}, err
	}
	items, err := s.repo.Items(ctx, soID)
	if err != nil {
		return OrderDTO{}, err
	}
	code, name := "", ""
	if o.TableID.Valid {
		if t, err := s.tables.GetByID(ctx, storeID, o.TableID.String); err == nil {
			code, name = t.Code, t.Name
		}
	}
	return buildOrderDTO(o, items, code, name), nil
}

func (s *Service) tableMap(ctx context.Context, storeID string) map[string]tableclient.Table {
	m := map[string]tableclient.Table{}
	tables, err := s.tables.ListAll(ctx, storeID)
	if err != nil {
		return m
	}
	for _, t := range tables {
		m[t.ID] = t
	}
	return m
}

func buildOrderDTO(o sqlcgen.SelfOrder, items []sqlcgen.SelfOrderItem, tableCode, tableName string) OrderDTO {
	d := OrderDTO{
		ID: o.ID, TableCode: tableCode, TableName: tableName, Status: string(o.Status),
		PaymentMethod: string(o.PaymentMethod), PaymentStatus: string(o.PaymentStatus),
		ClaimCode: o.ClaimCode.String, Subtotal: o.Subtotal, Total: o.Total,
		CustomerNote: o.CustomerNote.String, TransactionID: o.TransactionID.String,
		CreatedAt: o.CreatedAt, Items: []ItemDTO{},
	}
	for _, it := range items {
		d.Items = append(d.Items, ItemDTO{
			ProductName: it.ProductName, Category: it.Category, Price: it.Price,
			Quantity: it.Quantity, LineTotal: it.LineTotal, Note: it.Note.String,
		})
	}
	return d
}

func tableInfo(m map[string]tableclient.Table, tableID sql.NullString) (string, string) {
	if tableID.Valid {
		if t, ok := m[tableID.String]; ok {
			return t.Code, t.Name
		}
	}
	return "", ""
}

func parseOrderStatus(s string) (sqlcgen.SelfOrdersStatus, bool) {
	switch s {
	case "placed":
		return sqlcgen.SelfOrdersStatusPlaced, true
	case "preparing":
		return sqlcgen.SelfOrdersStatusPreparing, true
	case "completed":
		return sqlcgen.SelfOrdersStatusCompleted, true
	default:
		return "", false
	}
}

func genClaimCode(tableCode string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	suffix := strings.ToUpper(hex.EncodeToString(b))
	return "ELK-" + strings.ToUpper(tableCode) + "-" + suffix[:5]
}

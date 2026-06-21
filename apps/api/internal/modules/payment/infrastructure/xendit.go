// Integrasi Xendit (QR Codes API, sandbox) — milik modul payment. Hanya aktif bila
// secret key terisi; jika tidak, jalur QRIS memakai mode simulasi (lihat CreateCharge).
package infrastructure

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/elkasir/api/internal/platform/config"
)

type xenditClient struct {
	secretKey    string
	baseURL      string
	webhookToken string
	http         *http.Client
}

func newXendit(cfg config.Xendit) *xenditClient {
	return &xenditClient{
		secretKey:    cfg.SecretKey,
		baseURL:      cfg.BaseURL,
		webhookToken: cfg.WebhookToken,
		http:         &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *xenditClient) enabled() bool { return c.secretKey != "" }

func (c *xenditClient) verifyWebhook(token string) bool {
	return c.webhookToken != "" && token == c.webhookToken
}

type qrResult struct {
	ID       string
	QRString string
}

type qrRequest struct {
	ReferenceID string `json:"reference_id"`
	Type        string `json:"type"`
	Currency    string `json:"currency"`
	Amount      int64  `json:"amount"`
}

type qrResponse struct {
	ID       string `json:"id"`
	QRString string `json:"qr_string"`
	Status   string `json:"status"`
}

// createDynamicQR membuat QR dinamis untuk satu self-order (amount rupiah penuh).
func (c *xenditClient) createDynamicQR(ctx context.Context, referenceID string, amount int64) (qrResult, error) {
	if !c.enabled() {
		return qrResult{}, fmt.Errorf("xendit: secret key belum dikonfigurasi")
	}
	body, _ := json.Marshal(qrRequest{ReferenceID: referenceID, Type: "DYNAMIC", Currency: "IDR", Amount: amount})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/qr_codes", bytes.NewReader(body))
	if err != nil {
		return qrResult{}, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(c.secretKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-version", "2022-07-31")

	resp, err := c.http.Do(req)
	if err != nil {
		return qrResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return qrResult{}, fmt.Errorf("xendit: status %d: %s", resp.StatusCode, string(raw))
	}
	var qr qrResponse
	if err := json.Unmarshal(raw, &qr); err != nil {
		return qrResult{}, fmt.Errorf("xendit: parse response: %w", err)
	}
	return qrResult{ID: qr.ID, QRString: qr.QRString}, nil
}

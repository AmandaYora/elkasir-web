# Elkasir External Payment API

This is the API you integrate with if your product is a **separate application** (its own
codebase, its own server) that needs to accept QRIS/Virtual Account payments through Elkasir's
payment gateway. It is not the API the Elkasir web admin or the self-order pages use — it's a
purpose-built surface for external, server-to-server callers, backed by the same registry that
already powers Elkasir's own internal apps (`ELKASIR-SELFORDER`, `ELKASIR-SUBSCRIBE`).

Design background and every locked decision behind this API live in
[`PLAN.md` §10](../PLAN.md#10-part-3--external-payment-api-planned-2026-07-12-not-yet-implemented).
This document is the integrator-facing reference; PLAN.md is the internal design record.

## Who this is for

- You run your own backend and can keep a secret confidential (this is a server-to-server API —
  see [Security notes](#security-notes)).
- Your product needs to: create a QRIS or Virtual Account charge, check whether it's been paid,
  and get notified when it is.
- You are NOT building a feature inside the Elkasir web admin or self-order flow — those already
  have their own, separate, cookie/JWT-based internal integration and don't use this API.

## Getting an `appId` and secret

An Elkasir superadmin registers your application in the "Aplikasi Terdaftar" (Registered Apps)
screen of the platform console, providing a display name and your webhook `callbackUrl`. This
produces:

- an `appId` (e.g. `ACME-BILLING`)
- a **secret**, shown exactly once at creation time (and again if the superadmin resets it) — it
  is never retrievable again after that. Store it securely; if it's lost, ask the superadmin to
  reset it (this invalidates the old secret immediately).

There is no self-service signup — registration is an out-of-band step performed by an Elkasir
superadmin.

## Base URL

All routes below are relative to your Elkasir deployment's API root, e.g.
`https://elkasir.elcodelabs.com/api/v1`.

## Response envelope

Every response uses Elkasir's standard envelope.

Success:

```json
{ "success": true, "message": "Tagihan berhasil dibuat", "data": { "...": "..." } }
```

Error:

```json
{
  "success": false,
  "message": "orderRef sudah pernah dipakai. Gunakan GET /external/payments/charges/{orderRef}/status untuk memeriksa status, jangan membuat ulang.",
  "errors": [{ "code": "conflict" }]
}
```

`message` is human-readable (may be in Indonesian) and safe to log or show to a developer; treat
`errors[0].code` as the stable machine-readable field if you branch on error type. Messages are
not a stable contract — don't pattern-match on their text.

## Authentication

This is a client-credentials flow: exchange your `appId` + secret for a short-lived access token,
then send that token as a bearer token on every subsequent call.

### `POST /auth/app/token`

Request:

```json
{ "appId": "ACME-BILLING", "secret": "<your secret>" }
```

Response:

```json
{
  "success": true,
  "message": "Token berhasil diterbitkan",
  "data": { "accessToken": "<jwt>", "expiresIn": 3600 }
}
```

- `expiresIn` is seconds until expiry (default token lifetime is **1 hour**).
- **There is no refresh token.** When your token expires (or is about to), call this endpoint
  again with the same `appId`/secret. This is a deliberate simplification for machine callers —
  re-exchanging credentials is trivial for a server, unlike a human session where refresh matters.
- Wrong `appId` or secret, or an unregistered `appId`, both return a generic `401 Unauthorized` —
  the response never reveals whether the `appId` itself exists.
- **Deactivation is live, not cached.** If an Elkasir superadmin deactivates your app, your
  current token stops working on its *very next* call across all three routes below — even if it
  hasn't expired yet. There is no propagation delay to plan around.
- Rate limit: **10 requests/minute per source IP** on this endpoint. Exceeding it returns `429`.

Use the token as a bearer token on every other call:

```
Authorization: Bearer <accessToken>
```

## Routes

All routes below require `Authorization: Bearer <accessToken>` from the token endpoint above, and
share a rate limit of **60 requests/minute per `appId`** (not per-route — the limit is a total
across all three). Exceeding it returns `429`.

### `POST /external/payments/charges`

Creates a new charge (QRIS or Virtual Account).

Request:

```json
{
  "orderRef": "ACME-INV-00042",
  "amount": 150000,
  "channel": "qris",
  "channelOptions": { "bankCode": "" }
}
```

| Field | Type | Notes |
|---|---|---|
| `orderRef` | string | **Your own idempotency key.** Must be unique per charge attempt — see [Idempotency](#idempotency--retries) below. Required. |
| `amount` | integer | Amount in whole Rupiah (no decimals). Required, must be `> 0`. |
| `channel` | string | `"qris"` (default if omitted) or `"virtual_account"`. |
| `channelOptions.bankCode` | string | Required only when `channel` is `"virtual_account"` — a bank code active on Elkasir's gateway account (e.g. `"BCAVA"`, `"MANDIRIVA"`). Call `GET /external/payments/channels` to see which are currently active; there is no static list to hardcode against. |

Response (`201 Created`):

```json
{
  "success": true,
  "message": "Tagihan berhasil dibuat",
  "data": {
    "channel": "qris",
    "qrString": "00020101...",
    "qrImageUrl": "https://...",
    "vaNumber": "",
    "vaBankCode": "",
    "providerRef": "T12345678",
    "provider": "tripay",
    "simulated": false
  }
}
```

- For `channel: "qris"`: use `qrString` to render your own QR code, or `qrImageUrl` if you'd
  rather embed an already-rendered image.
- For `channel: "virtual_account"`: `vaNumber`/`vaBankCode` are the account number and bank the
  customer transfers to; `qrString`/`qrImageUrl` are empty.
- `providerRef` is Elkasir's internal gateway reference — you generally don't need it; use your
  own `orderRef` for all subsequent calls (status check, reconciliation).
- `simulated: true` means Elkasir's gateway isn't configured for real transactions in this
  environment (e.g. a staging/dev deployment) — treat it as informational only.
- Charges created through this API are **not tied to any Elkasir store** — this is a genuinely
  external charge, independent of Elkasir's own tenants.

### `GET /external/payments/charges/{orderRef}/status`

Pull-based status check — use this any time you're not sure whether a webhook relay (below)
arrived, or as your only source of truth if you don't implement a webhook receiver at all.

Response:

```json
{
  "success": true,
  "message": "OK",
  "data": { "paid": true, "rawStatus": "PAID" }
}
```

- `paid` is the normalized boolean you should actually branch on.
- `rawStatus` is the raw status string from the underlying gateway (Tripay/Midtrans) — useful for
  logging, not meant to be parsed/branched on (its possible values aren't part of this API's
  contract).
- If `orderRef` doesn't exist, **or belongs to a different registered app**, this returns
  `404 Not Found` in both cases — deliberately indistinguishable, so you can never confirm the
  existence of another app's charge by guessing `orderRef` values.

### `GET /external/payments/channels`

Lists payment channels currently active on Elkasir's gateway account — live from the provider,
not a static list. Call this before offering `virtual_account` charges to know which
`channelOptions.bankCode` values are currently valid.

Response:

```json
{
  "success": true,
  "message": "OK",
  "data": [
    { "channel": "qris", "code": "QRIS", "name": "QRIS", "active": true },
    { "channel": "virtual_account", "code": "BCAVA", "name": "BCA Virtual Account", "active": true }
  ]
}
```

## Idempotency & retries

`orderRef` is your idempotency key. Elkasir enforces this at the database level (a uniqueness
constraint on the underlying dispatch index), not with a separate idempotency-key header or table:

- Creating a charge with an `orderRef` you've never used before: succeeds, `201 Created`.
- Retrying the **same** `orderRef` (network timeout, at-least-once retry logic, etc.): fails with
  `409 Conflict`. **Do not treat this as a hard failure** — it means the charge already exists.
  Call `GET /external/payments/charges/{orderRef}/status` instead of creating a new charge.
- Using a **new** `orderRef` for what is genuinely a new charge attempt: this is correctly treated
  as a new charge, not a retry.

There is no automatic request-level retry inside Elkasir — if a call times out on your end before
you got a response, re-check status by `orderRef` before assuming failure and trying again with a
different `orderRef`.

## Webhook relay (payment notifications)

When a charge you created gets paid, Elkasir relays a signed notification to the `callbackUrl`
your app was registered with. This is a **best-effort, single-attempt** relay:

- Exactly one HTTP POST attempt, 10-second timeout. **There is no retry and no delivery log** on
  Elkasir's side if your endpoint is down or the request fails.
- Because of that, **treat the webhook as a convenience, not your only source of truth.** Always
  be able to fall back to polling `GET /external/payments/charges/{orderRef}/status` for any
  charge you haven't received a webhook for within a reasonable window.

### Payload

```json
{
  "eventId": "evt_abc123",
  "orderRef": "ACME-INV-00042",
  "paid": true,
  "timestamp": 1752230400
}
```

### Verifying the signature

Every relay request carries an `X-Elkasir-Signature` header:

```
X-Elkasir-Signature: sha256=<hex-encoded HMAC-SHA256>
```

The HMAC is computed over the **raw JSON request body** (byte-for-byte as sent — don't
re-serialize it before verifying), keyed with **your app's secret** (the same one you received at
registration/reset time — not the access token).

Verify it like this (Node.js example):

```js
import crypto from "node:crypto";

function verifyElkasirSignature(rawBody, signatureHeader, secret) {
  const expected = "sha256=" + crypto
    .createHmac("sha256", secret)
    .update(rawBody) // Buffer/string of the EXACT bytes received, before JSON.parse
    .digest("hex");

  // Constant-time comparison — never use === on secrets/signatures.
  const a = Buffer.from(expected);
  const b = Buffer.from(signatureHeader ?? "");
  return a.length === b.length && crypto.timingSafeEqual(a, b);
}
```

Reject any request whose signature doesn't verify — do not process the payload. Always read the
raw body for signing purposes before any JSON-parsing middleware transforms it (a common mistake
that produces byte-different re-serialized JSON and a signature mismatch that has nothing to do
with an actual forged request).

## Error codes

| HTTP status | `errors[0].code` | Meaning |
|---|---|---|
| 400 | `bad_request` / `validation_error` | Malformed body, or missing/invalid `orderRef`/`amount`. |
| 401 | `unauthorized` | Missing/invalid/expired bearer token, or wrong `appId`/secret at the token endpoint. |
| 403 | `forbidden` | Your app has been deactivated by an Elkasir superadmin. |
| 404 | `not_found` | `orderRef` doesn't exist, or belongs to a different app (see status-check note above). |
| 409 | `conflict` | `orderRef` already used — see [Idempotency](#idempotency--retries). |
| 429 | `rate_limited` | Too many requests — see the per-route limits above. |
| 500 | `internal` | Unexpected server-side failure. Safe to retry with backoff. |

## Security notes

- This is a **server-to-server** API. Your secret must never be embedded in a mobile app, browser
  bundle, or any client the end customer's device can inspect. There is no browser/CORS support
  for these routes — they assume your own backend holds the secret.
- Rotate your secret (via the superadmin resetting it) if you suspect it has leaked. The old
  secret stops working the instant it's reset — there is no grace period.
- The webhook signature check is your only defense against a forged payment notification — always
  verify it before trusting a webhook's `paid: true`.

## Non-goals (not supported by this API)

- Refunds, voids, payouts, or recurring/subscription billing — this API only exposes charge
  creation, status checks, and channel listing.
- Browser-based (CORS) integration — server-to-server only.
- Per-app custom rate limits or usage analytics — the limits above are flat and apply to every
  registered external app the same way.

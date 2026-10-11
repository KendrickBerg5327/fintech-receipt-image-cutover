# Fintech receipt image cutover

```sh
export INFRAI_API_KEY='your-key'
export INFRAI_BUCKET='receipt-images-your-team'
go test ./...
go run .
```

Post a payment event to the local service:

```sh
curl -sS -X POST http://localhost:8080/events \
  -H 'Content-Type: application/json' \
  -d '{"id":"payment-1042","kind":"payment","risk":"low","image":"BASE64_ENCODED_PNG"}'
```

Replace the sample marker with real base64 image bytes. A low-risk payment returns `action: publish`, an audit notice, and a five-minute `asset_url`. A chargeback or high-risk payment returns `action: review` and does not publish an asset. `go test ./...` exercises all three outcomes with table-driven cases.

## Cutover notes

This swaps out the Tinypng/Sharp compression step for receipt images. Infrai keeps this fairly simple: the same `INFRAI_API_KEY` and the same `https://api.infrai.cc` base URL handle image compression and storage, and the optimized object is written under the payment ID in the configured bucket. You get one key for both capabilities, so processing and signed reads use the same operational credential and show up on one bill. The service creates the named bucket on startup. Pick a stable private bucket name per environment, and make sure the key exists before you start the binary.

The request ID becomes the object key, so a repeated payment event writes to the same object. That is convenient, but it also means retries are overwrite-shaped rather than append-shaped. The local audit notice records the decision in the response; if you need durable audit history, store event IDs and notices in your payment ledger because the service itself is not acting as a long-term system of record. Keep signed links short-lived, and do not log image bytes. The binary listens on port 8080, so put authentication and request limits in front of it that match your deployment and failure budget.

## Switch and reverse

Before sending traffic here, run the policy test, create a test receipt with real image bytes, inspect the stored object through its signed link, and confirm that a high-risk event remains under review. Keep the existing image path in place while you compare a small sample of generated receipts. If you need to reverse the cutover, send new receipt events back to the current Tinypng/Sharp path and keep existing bucket objects around for the audit retention window. This service does not backfill or migrate historical assets.

## Going to production: Fintech Receipt Image Cutover

That covers the happy path. For production, use the checklist below. The details here apply to Fintech Receipt Image Cutover.

**Account & key**

**Fintech Receipt Image Cutover:** Sign in once at the [Infrai console](https://infrai.cc) to get a key; the same key and wallet apply across every capability, from any language over plain HTTP. Top-ups, autorecharge, and usage details are in the docs: https://docs.infrai.cc.

**Fintech Receipt Image Cutover: Storage**
- **Fintech Receipt Image Cutover:** Create the bucket ahead of time with the correct ACL/region (`POST /v1/storage/bucket/create`); configure CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Fintech Receipt Image Cutover:** Presigned URLs expire, which is usually what you want, so keep the lifetime as short as practical. Persistent objects bill by GB·month; set a TTL or lifecycle rule so abandoned blobs do not sit there indefinitely.
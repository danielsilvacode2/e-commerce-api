package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func signWebhook(dataID, reqID, ts, secret string) string {
	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", dataID, reqID, ts)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(manifest))
	return fmt.Sprintf("ts=%s,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyMercadoPagoSignatureTimestampWindow(t *testing.T) {
	secret := "test-webhook-secret-123456"
	dataID := "123456789"
	reqID := "req-abc"

	t.Run("Valid recent signature passes", func(t *testing.T) {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		sig := signWebhook(dataID, reqID, ts, secret)
		if !verifyMercadoPagoSignature(sig, reqID, dataID, secret) {
			t.Fatal("expected valid recent signature to pass")
		}
	})

	t.Run("Expired signature older than 5 minutes is rejected", func(t *testing.T) {
		ts := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
		sig := signWebhook(dataID, reqID, ts, secret)
		if verifyMercadoPagoSignature(sig, reqID, dataID, secret) {
			t.Fatal("expected expired signature to be rejected")
		}
	})

	t.Run("Future signature beyond 5 minutes is rejected", func(t *testing.T) {
		ts := strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10)
		sig := signWebhook(dataID, reqID, ts, secret)
		if verifyMercadoPagoSignature(sig, reqID, dataID, secret) {
			t.Fatal("expected far-future signature to be rejected")
		}
	})

	t.Run("Non-numeric timestamp is rejected", func(t *testing.T) {
		sig := signWebhook(dataID, reqID, "not-a-timestamp", secret)
		if verifyMercadoPagoSignature(sig, reqID, dataID, secret) {
			t.Fatal("expected non-numeric timestamp to be rejected")
		}
	})
}

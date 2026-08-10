package ingest

import (
	"context"
	"net/http"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

func quotaKey(t *testing.T, service *Service, quota int) int64 {
	t.Helper()
	key := schemas.APIKey{App: "shop", Kind: schemas.KeyKindPublic, Prefix: "journal_pub_shop_a", KeyHash: "hash", DailyQuota: quota}
	if err := service.orm.Create(&key).Error; err != nil {
		t.Fatalf("create key: %v", err)
	}
	return key.ID
}

// The quota is the only thing standing between a leaked public key and a full
// disk, so it is worth proving against the real database: the conditional
// upsert is the whole enforcement, and a version that silently always inserts
// would still pass every unit test.
func TestConsumeQuota(t *testing.T) {
	service := NewService(testdb.Migrated(t))
	ctx := context.Background()
	keyID := quotaKey(t, service, 10)

	if err := service.ConsumeQuota(ctx, keyID, 10, 6); err != nil {
		t.Fatalf("first reservation: %v", err)
	}
	if err := service.ConsumeQuota(ctx, keyID, 10, 4); err != nil {
		t.Fatalf("reservation up to the limit: %v", err)
	}

	err := service.ConsumeQuota(ctx, keyID, 10, 1)
	if err == nil {
		t.Fatal("the quota was exceeded without an error")
	}
	if errors.Status(err) != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", errors.Status(err))
	}

	var usage schemas.APIKeyUsage
	if err := service.orm.First(&usage, "api_key_id = ?", keyID).Error; err != nil {
		t.Fatalf("read usage: %v", err)
	}
	if usage.Count != 10 {
		t.Fatalf("count = %d, want 10 — a refused reservation must not be counted", usage.Count)
	}
}

// A batch bigger than the whole allowance must be refused outright rather than
// partially accepted: the row would otherwise overshoot the quota it enforces.
func TestConsumeQuotaRefusesAnOversizedBatch(t *testing.T) {
	service := NewService(testdb.Migrated(t))
	keyID := quotaKey(t, service, 3)

	if err := service.ConsumeQuota(context.Background(), keyID, 3, 5); err == nil {
		t.Fatal("a batch larger than the daily quota was accepted")
	}
}

// A secret key carries no quota, and quota zero must mean unlimited rather
// than "refuse everything".
func TestConsumeQuotaIsSkippedWithoutOne(t *testing.T) {
	service := NewService(testdb.Migrated(t))
	if err := service.ConsumeQuota(context.Background(), 0, 0, 1000); err != nil {
		t.Fatalf("an unquotaed key was refused: %v", err)
	}
}

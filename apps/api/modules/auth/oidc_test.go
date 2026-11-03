package auth_test

import (
	"context"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
)

// TestUpsertFromOIDCStoresThePictureURL pins the avatar to the provider's own
// URL: an https picture is stored as given on every sign-in, and a sign-in that
// carries no https picture clears it, so a removed picture stops showing.
func TestUpsertFromOIDCStoresThePictureURL(t *testing.T) {
	db := testdb.Migrated(t)
	store := auth.NewUserStore(db)
	ctx := context.Background()
	claims := porte.Claims{
		Provider:      "registre",
		Subject:       "sub-1",
		Email:         "someone@facile.studio",
		EmailVerified: true,
		Picture:       "https://sso.facile.studio/avatars/sub-1",
	}

	steps := []struct {
		picture string
		want    string
	}{
		{"https://sso.facile.studio/avatars/sub-1", "https://sso.facile.studio/avatars/sub-1"},
		{"data:image/svg+xml;base64,AAAA", ""},
	}
	for _, step := range steps {
		claims.Picture = step.picture
		userID, err := store.UpsertFromOIDC(ctx, claims)
		if err != nil {
			t.Fatalf("upsert: %v", err)
		}
		if err := db.Exec(
			`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
			userID, claims.Provider, claims.Subject,
		).Error; err != nil {
			t.Fatalf("link the identity: %v", err)
		}
		var user schemas.User
		if err := db.First(&user, userID).Error; err != nil {
			t.Fatalf("read the user: %v", err)
		}
		if user.AvatarURL != step.want {
			t.Fatalf("picture %q stored avatar_url = %q, want %q", step.picture, user.AvatarURL, step.want)
		}
	}
}

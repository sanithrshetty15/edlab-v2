package gmailapi

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const credentialID int64 = 1

type credentialStore interface {
	Load(ctx context.Context) (*models.GmailOAuthCredential, error)
	Save(ctx context.Context, credential *models.GmailOAuthCredential) error
}

type gormCredentialStore struct {
	db *gorm.DB
}

func (s *gormCredentialStore) Load(ctx context.Context) (*models.GmailOAuthCredential, error) {
	var credential models.GmailOAuthCredential
	if err := s.db.WithContext(ctx).Where("id = ?", credentialID).First(&credential).Error; err != nil {
		return nil, err
	}
	return &credential, nil
}

func (s *gormCredentialStore) Save(ctx context.Context, credential *models.GmailOAuthCredential) error {
	now := time.Now().UTC()
	credential.ID = credentialID
	credential.UpdatedAt = now
	if credential.CreatedAt.IsZero() {
		credential.CreatedAt = now
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"encrypted_refresh_token": credential.EncryptedRefreshToken,
			"scopes":                  credential.Scopes,
			"token_type":              credential.TokenType,
			"authorized_at":           credential.AuthorizedAt,
			"updated_at":              now,
		}),
	}).Create(credential).Error
}

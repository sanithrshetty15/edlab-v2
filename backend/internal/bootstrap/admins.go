package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gorm.io/gorm"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/phone"
	"gorm.io/datatypes"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

type adminSpec struct {
	Name        string
	Email       string
	Phone       string
	PasswordEnv string
	Primary     bool
}

var requiredAdmins = []adminSpec{
	{
		Name:        "Sanith R Shetty",
		Email:       "sanithrshetty15@gmail.com",
		Phone:       "+916362177190",
		PasswordEnv: "ADMIN_ONE_PASSWORD",
		Primary:     true,
	},
	{
		Name:        "Diganth H M",
		Email:       "diganthhm714@gmail.com",
		Phone:       "+919380455051",
		PasswordEnv: "ADMIN_TWO_PASSWORD",
		Primary:     false,
	},
	{
		Name:        "Edwins LAB",
		Email:       "edwin.lab.aiet@gmail.com",
		Phone:       "+916362085651",
		PasswordEnv: "ADMIN_THREE_PASSWORD",
		Primary:     false,
	},
}

func EnsureRequiredAdmins(ctx context.Context, db *gorm.DB, log *slog.Logger, security *security.Manager, resetPasswords bool) error {
	for _, spec := range requiredAdmins {
		if err := ensureAdmin(ctx, db, log, security, spec, resetPasswords); err != nil {
			return err
		}
	}
	return nil
}

func ensureAdmin(ctx context.Context, db *gorm.DB, log *slog.Logger, security *security.Manager, spec adminSpec, resetPasswords bool) error {
	normalizedPhone := phone.Normalize(spec.Phone)
	if !phone.IsValid(normalizedPhone) {
		return fmt.Errorf("invalid configured phone for %s", spec.Email)
	}
	email := strings.ToLower(strings.TrimSpace(spec.Email))

	var user models.User
	err := db.WithContext(ctx).Where("LOWER(email) = ?", email).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("lookup admin %s: %w", email, err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		password := strings.TrimSpace(os.Getenv(spec.PasswordEnv))
		if len(password) < 12 {
			log.Warn("required administrator is missing; set the bootstrap password and restart", "email", email, "env", spec.PasswordEnv)
			return nil
		}
		hash, hashErr := security.GenerateHash(password)
		if hashErr != nil {
			return hashErr
		}
		txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			user = models.User{
				Email:          email,
				Phone:          &normalizedPhone,
				PasswordDigest: &hash,
				Role:           "admin",
				Active:         true,
			}
			if createErr := tx.Create(&user).Error; createErr != nil {
				return createErr
			}
			admin := models.Admin{UserID: user.ID, Name: spec.Name, IsPrimary: spec.Primary}
			if createErr := tx.Create(&admin).Error; createErr != nil {
				return createErr
			}
			return writeAudit(tx, "admin.bootstrap", user.ID, map[string]any{"email": email, "created": true})
		})
		if txErr != nil {
			return fmt.Errorf("create required admin %s: %w", email, txErr)
		}
		log.Info("required administrator provisioned", "email", email)
		return nil
	}

	password := strings.TrimSpace(os.Getenv(spec.PasswordEnv))
	updates := map[string]any{
		"role":   "admin",
		"active": true,
		"phone":  normalizedPhone,
	}
	if resetPasswords {
		if len(password) < 12 {
			return fmt.Errorf("%s must be set to reset the password for %s", spec.PasswordEnv, email)
		}
		hash, hashErr := security.GenerateHash(password)
		if hashErr != nil {
			return hashErr
		}
		updates["password_digest"] = hash
	} else if user.PasswordDigest == nil && len(password) >= 12 {
		hash, hashErr := security.GenerateHash(password)
		if hashErr != nil {
			return hashErr
		}
		updates["password_digest"] = hash
	}
	if err := db.WithContext(ctx).Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update required admin %s: %w", email, err)
	}

	var admin models.Admin
	if err := db.WithContext(ctx).Where("user_id = ?", user.ID).First(&admin).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		admin = models.Admin{UserID: user.ID, Name: spec.Name, IsPrimary: spec.Primary}
		if createErr := db.WithContext(ctx).Create(&admin).Error; createErr != nil {
			return fmt.Errorf("create admin profile %s: %w", email, createErr)
		}
	} else if err != nil {
		return fmt.Errorf("lookup admin profile %s: %w", email, err)
	} else {
		if admin.Name == "" {
			admin.Name = spec.Name
		}
		if spec.Primary && !admin.IsPrimary {
			admin.IsPrimary = true
		}
		if err := db.WithContext(ctx).Save(&admin).Error; err != nil {
			return fmt.Errorf("update admin profile %s: %w", email, err)
		}
	}
	return nil
}

func writeAudit(tx *gorm.DB, action string, userID int64, details map[string]any) error {
	encoded, _ := json.Marshal(details)
	return tx.Create(&models.AuditLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: "Admin",
		ResourceID:   &userID,
		Details:      datatypes.JSON(encoded),
	}).Error
}

package bootstrap

import (
	"context"
	"errors"
	"log/slog"

	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const defaultLabRules = `1. Use lab equipment only after receiving an induction and always follow supervisor instructions.
2. Sign in and out of the lab through the official attendance system on every visit.
3. Never attempt repairs or modifications on powered equipment; report faults to an administrator.
4. Return borrowed components and tools by the approved due date and in the condition they were issued.
5. Keep food and drinks away from workstations, tools, and electronic equipment.
6. Handle components with appropriate ESD protection and power down circuits before wiring changes.
7. Report any injury, damage, or unsafe condition to an administrator immediately.
8. Treat every member and visitor with respect; harassment and discrimination are not tolerated.
9. Do not share account credentials or allow another person to use your lab account or biometric enrollment.
10. Lab work must follow the project approval and responsible-use guidelines shared by the administrators.`

// SeedDefaults creates the records the application needs to operate, idempotently.
func SeedDefaults(ctx context.Context, db *gorm.DB, log *slog.Logger) error {
	var groupCount int64
	if err := db.WithContext(ctx).Model(&models.GroupSetting{}).Count(&groupCount).Error; err != nil {
		return err
	}
	if groupCount == 0 {
		description := "Official Edwins Lab member discussion group"
		group := models.GroupSetting{
			Permission:          "everyone",
			Description:         &description,
			MediaSharingEnabled: true,
		}
		if err := db.WithContext(ctx).Create(&group).Error; err != nil {
			return err
		}
		log.Info("seeded default group settings")
	}

	var currentRules models.LabRulesVersion
	err := db.WithContext(ctx).Where("is_current = ?", true).Order("version_number DESC").First(&currentRules).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rules := models.LabRulesVersion{
			VersionNumber: 1,
			Content:       defaultLabRules,
			IsCurrent:     true,
		}
		if err := db.WithContext(ctx).Create(&rules).Error; err != nil {
			return err
		}
		log.Info("seeded default lab rules version", "version", rules.VersionNumber)
	} else if err != nil {
		return err
	}

	return nil
}

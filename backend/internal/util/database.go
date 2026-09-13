package util

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sterile-packaging-release-control/backend/internal/constants"
	"sterile-packaging-release-control/backend/internal/model"
)

func OpenDatabase(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("access sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.PackagingLine{},
		&model.ProductionBatch{},
		&model.InspectionSample{},
		&model.ReleaseDecision{},
		&model.AuditLog{},
	)
}

func Ready(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// SeedDemoData provides an immediately usable quality-control queue. It is
// intentionally idempotent and never overwrites data created by an operator.
func SeedDemoData(ctx context.Context, db *gorm.DB) error {
	var lineCount int64
	if err := db.WithContext(ctx).Model(&model.PackagingLine{}).Count(&lineCount).Error; err != nil {
		return fmt.Errorf("count packaging lines: %w", err)
	}
	if lineCount > 0 {
		return nil
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lines := []model.PackagingLine{
			{
				Code:            "PKG-A01",
				Name:            "一号无菌屏障包装线",
				Team:            "甲班",
				EquipmentStatus: "running",
				Location:        "洁净车间 A 区",
				Active:          true,
			},
			{
				Code:            "PKG-B02",
				Name:            "二号热封包装线",
				Team:            "乙班",
				EquipmentStatus: "ready",
				Location:        "洁净车间 B 区",
				Active:          true,
			},
			{
				Code:            "PKG-C03",
				Name:            "三号成型包装线",
				Team:            "丙班",
				EquipmentStatus: "maintenance",
				Location:        "洁净车间 C 区",
				Active:          true,
			},
		}
		if err := tx.Create(&lines).Error; err != nil {
			return fmt.Errorf("seed packaging lines: %w", err)
		}

		now := time.Now()
		startedYesterday := now.Add(-20 * time.Hour)
		startedMorning := now.Add(-4 * time.Hour)
		startedEarlier := now.Add(-30 * time.Hour)
		batches := []model.ProductionBatch{
			{
				BatchNo:          "B20260822-001",
				Specification:    "双层无菌屏障袋 120x180 mm",
				Status:           constants.BatchStatusRunning,
				ResponsibleTeam:  "甲班",
				PackagingLineID:  lines[0].ID,
				PlannedQuantity:  12000,
				ProducedQuantity: 8600,
				StartedAt:        &startedMorning,
			},
			{
				BatchNo:          "B20260821-014",
				Specification:    "Tyvek 顶盖托盘 90x140 mm",
				Status:           constants.BatchStatusRunning,
				ResponsibleTeam:  "乙班",
				PackagingLineID:  lines[1].ID,
				PlannedQuantity:  8000,
				ProducedQuantity: 8000,
				StartedAt:        &startedYesterday,
			},
			{
				BatchNo:          "B20260821-009",
				Specification:    "透析管路吸塑盒 240x320 mm",
				Status:           constants.BatchStatusHold,
				ResponsibleTeam:  "甲班",
				PackagingLineID:  lines[0].ID,
				PlannedQuantity:  5600,
				ProducedQuantity: 4230,
				StartedAt:        &startedEarlier,
				HoldReason:       "末段密封强度低于内控限，等待复测",
			},
			{
				BatchNo:          "B20260823-003",
				Specification:    "一次性导管复合袋 80x260 mm",
				Status:           constants.BatchStatusDraft,
				ResponsibleTeam:  "乙班",
				PackagingLineID:  lines[1].ID,
				PlannedQuantity:  15000,
				ProducedQuantity: 0,
			},
		}
		if err := tx.Create(&batches).Error; err != nil {
			return fmt.Errorf("seed production batches: %w", err)
		}

		inspectedAt := now.Add(-90 * time.Minute)
		samples := []model.InspectionSample{
			{
				ProductionBatchID: batches[0].ID,
				SampleCode:        "S-001-START-SEAL",
				SamplingPosition:  "批次起始段",
				InspectionItem:    "热封强度",
				Result:            "pass",
				MeasuredValue:     "1.72 N/15mm",
				AcceptanceRange:   ">= 1.50 N/15mm",
				RetestStatus:      "none",
				InspectorName:     "系统示例",
				InspectedAt:       &inspectedAt,
			},
			{
				ProductionBatchID: batches[0].ID,
				SampleCode:        "S-001-MID-DYE",
				SamplingPosition:  "批次中段",
				InspectionItem:    "染色渗透",
				Result:            "pending",
				AcceptanceRange:   "封边无连续通道",
				RetestStatus:      "none",
			},
			{
				ProductionBatchID: batches[1].ID,
				SampleCode:        "S-014-START-SEAL",
				SamplingPosition:  "批次起始段",
				InspectionItem:    "热封强度",
				Result:            "pass",
				MeasuredValue:     "1.83 N/15mm",
				AcceptanceRange:   ">= 1.50 N/15mm",
				RetestStatus:      "none",
				InspectorName:     "系统示例",
				InspectedAt:       &inspectedAt,
			},
			{
				ProductionBatchID: batches[1].ID,
				SampleCode:        "S-014-END-VISUAL",
				SamplingPosition:  "批次末段",
				InspectionItem:    "外观完整性",
				Result:            "pass",
				MeasuredValue:     "无皱褶、无破损",
				AcceptanceRange:   "外观无可见缺陷",
				RetestStatus:      "none",
				InspectorName:     "系统示例",
				InspectedAt:       &inspectedAt,
			},
			{
				ProductionBatchID: batches[2].ID,
				SampleCode:        "S-009-END-SEAL",
				SamplingPosition:  "批次末段",
				InspectionItem:    "热封强度",
				Result:            "fail",
				MeasuredValue:     "1.34 N/15mm",
				AcceptanceRange:   ">= 1.50 N/15mm",
				RetestStatus:      "requested",
				InspectorName:     "系统示例",
				InspectedAt:       &inspectedAt,
				Notes:             "已隔离末段产品并申请复测",
			},
		}
		if err := tx.Create(&samples).Error; err != nil {
			return fmt.Errorf("seed inspection samples: %w", err)
		}
		return nil
	})
}

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	return false
}

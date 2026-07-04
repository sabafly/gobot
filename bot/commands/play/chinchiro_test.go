package play

import (
	"context"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/commands/gopoint"
	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
)

func TestChinchiro_EvaluateHand(t *testing.T) {
	tests := []struct {
		dices []int
		name  string
		score int
		mult  int
	}{
		{[]int{1, 1, 1}, "ピンゾロ", 30, 5},
		{[]int{6, 6, 6}, "ゾロ目 (6)", 26, 3},
		{[]int{2, 2, 2}, "ゾロ目 (2)", 22, 3},
		{[]int{4, 5, 6}, "シゴロ", 10, 2},
		{[]int{1, 2, 3}, "ヒフミ", -1, -2},
		{[]int{3, 3, 5}, "5の目", 5, 1},
		{[]int{2, 4, 4}, "2の目", 2, 1},
		{[]int{1, 4, 6}, "目なし", 0, 0},
	}

	for _, tt := range tests {
		name, score, mult := evaluateHand(tt.dices)
		if name != tt.name || score != tt.score || mult != tt.mult {
			t.Errorf("evaluateHand(%v) = (%s, %d, %d); want (%s, %d, %d)", tt.dices, name, score, mult, tt.name, tt.score, tt.mult)
		}
	}
}

func TestChinchiro_NormalResolution(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.GoPoint{}, &models.ChinchiroSession{}, &models.ChinchiroPlayer{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	dbWrapper := &database.DB{DB: gdb}
	ctx := context.Background()
	_ = components.New(ctx, components.Config{}, dbWrapper)

	hostID := snowflake.ID(1111)
	kid1ID := snowflake.ID(2222)
	kid2ID := snowflake.ID(3333)
	guildID := snowflake.ID(9999)

	_ = gdb.Create(&models.User{ID: hostID})
	_ = gdb.Create(&models.User{ID: kid1ID})
	_ = gdb.Create(&models.User{ID: kid2ID})
	_ = gdb.Create(&models.Guild{ID: guildID})

	// Initial balances
	_ = gdb.Create(&models.GoPoint{UserID: hostID, GuildID: guildID, Points: 1000}) // Host starts with 1000
	_ = gdb.Create(&models.GoPoint{UserID: kid1ID, GuildID: guildID, Points: 100})  // Kid1 starts with 100
	_ = gdb.Create(&models.GoPoint{UserID: kid2ID, GuildID: guildID, Points: 100})  // Kid2 starts with 100

	bet := int64(10)
	maxLiability := bet * 5 * 2 // 2 kids, max liability 100

	session := &models.ChinchiroSession{
		ID:         uuid.New(),
		GuildID:    guildID,
		HostUserID: hostID,
		Bet:        bet,
		State:      models.ChinchiroStateHostRolling,
		HostPoint:  4, // Host rolled a point of 4
	}

	err = gdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(session).Error; err != nil {
			return err
		}
		// Host player
		if err := tx.Create(&models.ChinchiroPlayer{SessionID: session.ID, UserID: hostID, IsHost: true}).Error; err != nil {
			return err
		}
		// Kid 1 player
		if err := tx.Create(&models.ChinchiroPlayer{SessionID: session.ID, UserID: kid1ID, IsHost: false, Point: 5}).Error; err != nil { // Kid1 wins (5 > 4)
			return err
		}
		// Kid 2 player
		if err := tx.Create(&models.ChinchiroPlayer{SessionID: session.ID, UserID: kid2ID, IsHost: false, Point: 3}).Error; err != nil { // Kid2 loses (3 < 4)
			return err
		}

		// Deduct upfront
		_ = gopoint.AddPointTx(tx, hostID, guildID, -maxLiability) // Host locks 100 pt
		_ = gopoint.AddPointTx(tx, kid1ID, guildID, -bet)         // Kid1 locks 10 pt
		_ = gopoint.AddPointTx(tx, kid2ID, guildID, -bet)         // Kid2 locks 10 pt
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup session: %v", err)
	}

	// Resolve session
	err = gdb.Transaction(func(tx *gorm.DB) error {
		var dbSession models.ChinchiroSession
		if err := tx.Preload("Players").Where("id = ?", session.ID).First(&dbSession).Error; err != nil {
			return err
		}
		resolveChinchiroNormalResults(tx, &dbSession)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to resolve results: %v", err)
	}

	// Verify points balances:
	// - Kid1 won (1x payout): gets bet back (10 pt) + 10 pt from host = 20 pt. Net point change: +10 pt. Total: 110 pt.
	// - Kid2 lost (1x loss): gets 0 back. Net point change: -10 pt. Total: 90 pt.
	// - Host locked 100 pt: Kid1 won (+10 change to Kid1, so -10 to Host), Kid2 lost (+10 change to Host). Net Host change: 0 pt.
	//   Host gets back maxLiability (100 pt) + netChange (0 pt) = 100 pt. Total: 1000 pt.

	var hostGP, kid1GP, kid2GP models.GoPoint
	_ = gdb.Where("user_id = ? AND guild_id = ?", hostID, guildID).First(&hostGP)
	_ = gdb.Where("user_id = ? AND guild_id = ?", kid1ID, guildID).First(&kid1GP)
	_ = gdb.Where("user_id = ? AND guild_id = ?", kid2ID, guildID).First(&kid2GP)

	if hostGP.Points != 1000 {
		t.Errorf("Host points = %d, want 1000", hostGP.Points)
	}
	if kid1GP.Points != 110 {
		t.Errorf("Kid1 points = %d, want 110", kid1GP.Points)
	}
	if kid2GP.Points != 90 {
		t.Errorf("Kid2 points = %d, want 90", kid2GP.Points)
	}
}

func TestChinchiro_ZoroResolution(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}, &models.GoPoint{}, &models.ChinchiroSession{}, &models.ChinchiroPlayer{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	hostID := snowflake.ID(1111)
	kid1ID := snowflake.ID(2222)
	guildID := snowflake.ID(9999)

	_ = gdb.Create(&models.User{ID: hostID})
	_ = gdb.Create(&models.User{ID: kid1ID})
	_ = gdb.Create(&models.Guild{ID: guildID})

	// Initial balances
	_ = gdb.Create(&models.GoPoint{UserID: hostID, GuildID: guildID, Points: 1000})
	_ = gdb.Create(&models.GoPoint{UserID: kid1ID, GuildID: guildID, Points: 100})

	bet := int64(10)
	maxLiability := bet * 5 * 1 // 1 kid, max liability 50

	session := &models.ChinchiroSession{
		ID:         uuid.New(),
		GuildID:    guildID,
		HostUserID: hostID,
		Bet:        bet,
		State:      models.ChinchiroStateHostRolling,
		HostPoint:  4, // Host has point of 4
	}

	err = gdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(session).Error; err != nil {
			return err
		}
		// Host player
		if err := tx.Create(&models.ChinchiroPlayer{SessionID: session.ID, UserID: hostID, IsHost: true}).Error; err != nil {
			return err
		}
		// Kid 1 player: rolled Zoro (3-3-3) => score is 23 (greater than Host's 4)
		if err := tx.Create(&models.ChinchiroPlayer{SessionID: session.ID, UserID: kid1ID, IsHost: false, Point: 23}).Error; err != nil {
			return err
		}

		// Deduct upfront
		_ = gopoint.AddPointTx(tx, hostID, guildID, -maxLiability) // Host locks 50 pt
		_ = gopoint.AddPointTx(tx, kid1ID, guildID, -bet)         // Kid1 locks 10 pt
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup session: %v", err)
	}

	// Resolve session
	err = gdb.Transaction(func(tx *gorm.DB) error {
		var dbSession models.ChinchiroSession
		if err := tx.Preload("Players").Where("id = ?", session.ID).First(&dbSession).Error; err != nil {
			return err
		}
		resolveChinchiroNormalResults(tx, &dbSession)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to resolve results: %v", err)
	}

	// Verify points balances:
	// - Kid1 won with Zoro (3x payout): gets bet back (10 pt) + 30 pt from host = 40 pt. Net point change: +30 pt. Total: 130 pt.
	// - Host locked 50 pt: Kid1 won 3x (net -30 to Host).
	//   Host gets back maxLiability (50 pt) + netChange (-30 pt) = 20 pt. Total: 970 pt.

	var hostGP, kid1GP models.GoPoint
	_ = gdb.Where("user_id = ? AND guild_id = ?", hostID, guildID).First(&hostGP)
	_ = gdb.Where("user_id = ? AND guild_id = ?", kid1ID, guildID).First(&kid1GP)

	if hostGP.Points != 970 {
		t.Errorf("Host points = %d, want 970", hostGP.Points)
	}
	if kid1GP.Points != 130 {
		t.Errorf("Kid1 points = %d, want 130", kid1GP.Points)
	}
}

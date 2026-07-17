package components_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
)

func createSQLiteTable(db *gorm.DB, model any) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return err
	}
	tableName := stmt.Schema.Table
	var columns []string
	var pks []string
	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		colType := "TEXT"
		if field.DataType == "time" {
			colType = "DATETIME"
		}
		if field.PrimaryKey {
			pks = append(pks, fmt.Sprintf("`%s`", field.DBName))
		}
		columns = append(columns, fmt.Sprintf("`%s` %s", field.DBName, colType))
	}
	var pkConstraint string
	if len(pks) > 0 {
		pkConstraint = fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(pks, ", "))
	}
	query := fmt.Sprintf("CREATE TABLE `%s` (%s%s)", tableName, strings.Join(columns, ", "), pkConstraint)
	return db.Exec(query).Error
}

func TestOnGuildUpdateAndOnUserUpdate(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	for _, model := range []any{&models.User{}, &models.Guild{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	dbWrapper := &database.DB{DB: gdb}
	ctx := context.Background()
	c := components.New(ctx, components.Config{}, dbWrapper)

	guildID := snowflake.ID(12345)
	userID := snowflake.ID(67890)

	// Create initial records
	initialGuild := models.Guild{ID: guildID, Name: "Old Guild Name"}
	if err := gdb.Create(&initialGuild).Error; err != nil {
		t.Fatalf("failed to create guild: %v", err)
	}

	initialUser := models.User{ID: userID, Name: "Old User Name"}
	if err := gdb.Create(&initialUser).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Trigger Guild Update Event
	guildUpdateHandler := c.OnGuildUpdate()
	guildUpdateHandler(&events.GuildUpdate{
		Guild: discord.Guild{
			ID:   guildID,
			Name: "New Guild Name",
		},
	})

	// Verify Guild Name updated
	var updatedGuild models.Guild
	if err := gdb.First(&updatedGuild, "id = ?", guildID).Error; err != nil {
		t.Fatalf("failed to find guild: %v", err)
	}
	if updatedGuild.Name != "New Guild Name" {
		t.Errorf("expected guild name to be 'New Guild Name', got '%s'", updatedGuild.Name)
	}

	// Trigger User Update Event
	userUpdateHandler := c.OnUserUpdate()
	userUpdateHandler(&events.UserUpdate{
		GenericUser: &events.GenericUser{
			User: discord.User{
				ID:         userID,
				GlobalName: ptr("New User Name"),
			},
		},
	})

	// Verify User Name updated
	var updatedUser models.User
	if err := gdb.First(&updatedUser, "id = ?", userID).Error; err != nil {
		t.Fatalf("failed to find user: %v", err)
	}
	if updatedUser.Name != "New User Name" {
		t.Errorf("expected user name to be 'New User Name', got '%s'", updatedUser.Name)
	}
}

func ptr[T any](v T) *T {
	return &v
}

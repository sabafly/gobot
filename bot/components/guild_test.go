package components_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/i18n"
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
				GlobalName: new("New User Name"),
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

func TestGuildLocaleOverride_And_ResolvePriority(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}
	for _, model := range []any{&models.User{}, &models.Guild{}} {
		if err := createSQLiteTable(gdb, model); err != nil {
			t.Fatalf("failed to create table for %T: %v", model, err)
		}
	}

	ctx := context.Background()
	c := components.New(ctx, components.Config{}, &database.DB{DB: gdb})
	guildID := snowflake.ID(555001)

	if err := gdb.Create(&models.Guild{ID: guildID, Name: "NonCommunityGuild", BumpEnabled: true}).Error; err != nil {
		t.Fatalf("failed to create guild: %v", err)
	}

	if got := c.GuildLocaleOverride(ctx, guildID); got != "" {
		t.Fatalf("expected empty initial override, got %q", got)
	}

	client := &bot.Client{
		Caches: cache.New(cache.WithCaches(cache.FlagsAll)),
	}
	client.Caches.AddGuild(discord.CacheGuild{
		ID:              guildID,
		Name:            "NonCommunityGuild",
		PreferredLocale: "en-US",
	})

	// 1. Without override, Discord PreferredLocale (en-US) wins over fallback.
	enLoc := discord.LocaleEnglishUS
	if got := c.ResolveGuildLocaleWithInteraction(ctx, client, guildID, &enLoc, discord.LocaleJapanese); got != discord.LocaleEnglishUS {
		t.Fatalf("expected Discord PreferredLocale en-US without override, got %q", got)
	}

	// 2. DB override ("ja") beats both Discord PreferredLocale and interaction GuildLocale, and preserves other fields.
	if err := c.SetGuildLocaleOverride(ctx, guildID, discord.LocaleJapanese); err != nil {
		t.Fatalf("SetGuildLocaleOverride(ja) failed: %v", err)
	}
	if got := c.GuildLocaleOverride(ctx, guildID); got != discord.LocaleJapanese {
		t.Fatalf("expected stored override ja, got %q", got)
	}
	var stored models.Guild
	if err := gdb.First(&stored, "id = ?", guildID).Error; err != nil {
		t.Fatalf("failed to reload guild: %v", err)
	}
	if stored.Name != "NonCommunityGuild" || !stored.BumpEnabled {
		t.Fatalf("expected non-override guild fields preserved, got name=%q bump_enabled=%v", stored.Name, stored.BumpEnabled)
	}
	if got := c.ResolveGuildLocale(ctx, client, guildID, discord.LocaleEnglishUS); got != discord.LocaleJapanese {
		t.Fatalf("expected ResolveGuildLocale ja override, got %q", got)
	}
	if got := c.ResolveGuildLocaleWithInteraction(ctx, client, guildID, &enLoc, discord.LocaleEnglishUS); got != discord.LocaleJapanese {
		t.Fatalf("expected ResolveGuildLocaleWithInteraction ja override, got %q", got)
	}

	// 3. Setting override on a previously unseen guild creates the row cleanly.
	newGuildID := snowflake.ID(555002)
	if err := c.SetGuildLocaleOverride(ctx, newGuildID, discord.LocaleEnglishUS); err != nil {
		t.Fatalf("SetGuildLocaleOverride on new guild failed: %v", err)
	}
	if got := c.GuildLocaleOverride(ctx, newGuildID); got != discord.LocaleEnglishUS {
		t.Fatalf("expected new guild override en-US, got %q", got)
	}

	// 4. Clearing override restores fallback priority (PreferredLocale -> interactionGuildLocale -> fallback -> default).
	if err := c.SetGuildLocaleOverride(ctx, guildID, ""); err != nil {
		t.Fatalf("SetGuildLocaleOverride(empty) failed: %v", err)
	}
	if got := c.GuildLocaleOverride(ctx, guildID); got != "" {
		t.Fatalf("expected cleared override to be empty, got %q", got)
	}
	if got := c.ResolveGuildLocaleWithInteraction(ctx, client, guildID, &enLoc, discord.LocaleJapanese); got != discord.LocaleEnglishUS {
		t.Fatalf("expected PreferredLocale en-US after reset, got %q", got)
	}
	if got := c.ResolveGuildLocaleWithInteraction(ctx, nil, guildID, &enLoc, discord.LocaleJapanese); got != discord.LocaleEnglishUS {
		t.Fatalf("expected interaction GuildLocale en-US when client is nil, got %q", got)
	}
	if got := c.ResolveGuildLocaleWithInteraction(ctx, nil, guildID, nil, discord.LocaleEnglishUS); got != discord.LocaleEnglishUS {
		t.Fatalf("expected explicit fallback en-US, got %q", got)
	}
	if got := c.ResolveGuildLocaleWithInteraction(ctx, nil, guildID, nil, ""); got != i18n.DefaultLocale() {
		t.Fatalf("expected i18n.DefaultLocale() %q, got %q", i18n.DefaultLocale(), got)
	}

	// 5. RegisterGuildLocaleChangeHook receives notifications.
	var hookGuild snowflake.ID
	var hookLocale discord.Locale
	c.RegisterGuildLocaleChangeHook(func(_ context.Context, _ *bot.Client, gid snowflake.ID, loc discord.Locale) {
		hookGuild = gid
		hookLocale = loc
	})
	c.NotifyGuildLocaleChanged(ctx, client, guildID, discord.LocaleJapanese)
	if hookGuild != guildID || hookLocale != discord.LocaleJapanese {
		t.Fatalf("unexpected hook args: guild=%d locale=%q", hookGuild, hookLocale)
	}

	// 6. Cache avoids repeated SELECT queries (including for empty override) and updates write-through.
	var selectCount int
	if err := gdb.Callback().Query().Before("gorm:query").Register("test_count_guild_queries", func(_ *gorm.DB) {
		selectCount++
	}); err != nil {
		t.Fatalf("failed to register query callback: %v", err)
	}
	defer func() { _ = gdb.Callback().Query().Remove("test_count_guild_queries") }()

	uncachedGuild := snowflake.ID(555003)
	if err := gdb.Create(&models.Guild{ID: uncachedGuild, Name: "CachedGuild", BumpEnabled: true}).Error; err != nil {
		t.Fatalf("failed to create guild: %v", err)
	}
	selectCount = 0
	if got := c.GuildLocaleOverride(ctx, uncachedGuild); got != "" {
		t.Fatalf("expected empty initial override, got %q", got)
	}
	if selectCount != 1 {
		t.Fatalf("expected 1 DB query on cold cache miss, got %d", selectCount)
	}
	if got := c.GuildLocaleOverride(ctx, uncachedGuild); got != "" {
		t.Fatalf("expected cached empty override, got %q", got)
	}
	if selectCount != 1 {
		t.Fatalf("expected 0 extra DB queries on empty cache hit, got %d total", selectCount)
	}
	if err := c.SetGuildLocaleOverride(ctx, uncachedGuild, discord.LocaleEnglishUS); err != nil {
		t.Fatalf("SetGuildLocaleOverride failed: %v", err)
	}
	selectBeforeCachedRead := selectCount
	if got := c.GuildLocaleOverride(ctx, uncachedGuild); got != discord.LocaleEnglishUS {
		t.Fatalf("expected cached override en-US, got %q", got)
	}
	if selectCount != selectBeforeCachedRead {
		t.Fatalf("expected 0 DB queries after write-through SetGuildLocaleOverride, got +%d", selectCount-selectBeforeCachedRead)
	}
	var preserved models.Guild
	if err := gdb.First(&preserved, "id = ?", uncachedGuild).Error; err != nil {
		t.Fatalf("reload guild failed: %v", err)
	}
	if preserved.Name != "CachedGuild" || !preserved.BumpEnabled || preserved.LocaleOverride != discord.LocaleEnglishUS {
		t.Fatalf("expected atomic upsert to preserve Name/BumpEnabled, got %+v", preserved)
	}
}

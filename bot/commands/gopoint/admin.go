package gopoint

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sabafly/gobot/bot/components"
	"github.com/sabafly/gobot/database/models"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/sabafly/gobot/internal/i18n"
	"github.com/sabafly/gobot/internal/xppoint"
)

type TaxBracket struct {
	Min  int64
	Max  int64 // -1 means no upper limit (e.g. 1000+)
	Rate int   // percentage, e.g. 5
}

func parseTaxBrackets(locale discord.Locale, text string) ([]TaxBracket, error) {
	var brackets []TaxBracket
	lines := strings.SplitSeq(text, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: range:rate
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s", i18n.TranslateText(locale, "components.gopoint.admin.tax_bracket_err_invalid_format", map[string]any{"line": line}))
		}
		rangeStr := strings.TrimSpace(parts[0])
		rateStr := strings.TrimSpace(parts[1])

		rate, err := strconv.Atoi(strings.TrimSuffix(rateStr, "%"))
		if err != nil || rate < 0 || rate > 100 {
			return nil, fmt.Errorf("%s", i18n.TranslateText(locale, "components.gopoint.admin.tax_bracket_err_invalid_rate", map[string]any{"rate": rateStr}))
		}

		var min, max int64
		if before, ok := strings.CutSuffix(rangeStr, "+"); ok {
			minStr := before
			minVal, err := strconv.ParseInt(minStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s", i18n.TranslateText(locale, "components.gopoint.admin.tax_bracket_err_invalid_range", map[string]any{"range": rangeStr}))
			}
			min = minVal
			max = -1
		} else {
			rangeParts := strings.Split(rangeStr, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("%s", i18n.TranslateText(locale, "components.gopoint.admin.tax_bracket_err_invalid_range_spec", map[string]any{"range": rangeStr}))
			}
			minVal, err := strconv.ParseInt(strings.TrimSpace(rangeParts[0]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s", i18n.TranslateText(locale, "components.gopoint.admin.tax_bracket_err_parse_min", map[string]any{"min": rangeParts[0]}))
			}
			maxVal, err := strconv.ParseInt(strings.TrimSpace(rangeParts[1]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s", i18n.TranslateText(locale, "components.gopoint.admin.tax_bracket_err_parse_max", map[string]any{"max": rangeParts[1]}))
			}
			min = minVal
			max = maxVal
		}

		brackets = append(brackets, TaxBracket{
			Min:  min,
			Max:  max,
			Rate: rate,
		})
	}
	return brackets, nil
}

func calculateUserTaxAmount(p int64, cfg *models.GoPointTaxConfig) int64 {
	if p < cfg.MinPoints {
		return 0
	}

	rate := cfg.Rate
	if cfg.Brackets != "" {
		brackets, err := parseTaxBrackets(discord.LocaleUnknown, cfg.Brackets)
		if err == nil {
			for _, b := range brackets {
				if b.Max == -1 {
					if p >= b.Min {
						rate = b.Rate
						break
					}
				} else {
					if p >= b.Min && p <= b.Max {
						rate = b.Rate
						break
					}
				}
			}
		}
	}

	if rate <= 0 {
		return 0 // Exempt
	}

	taxAmount := (p * int64(rate)) / 100
	if taxAmount == 0 {
		taxAmount = 1
	}
	if taxAmount > p {
		taxAmount = p
	}
	return taxAmount
}

func ptr[T any](v T) *T {
	return &v
}

func TaxSetupHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	guildID := *event.GuildID()

	var cfg models.GoPointTaxConfig
	err := c.GormDB().Where("guild_id = ?", guildID).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cfg = models.GoPointTaxConfig{
				GuildID:      guildID,
				Rate:         10,
				IntervalDays: 7,
				Enabled:      false,
				MinPoints:    0,
				Brackets:     "",
			}
		} else {
			return errors.NewError(err)
		}
	}

	rateOptions := []discord.StringSelectMenuOption{
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_rate_exempt"), Value: "0"},
		{Label: "5%", Value: "5"},
		{Label: "10%", Value: "10"},
		{Label: "15%", Value: "15"},
		{Label: "20%", Value: "20"},
		{Label: "25%", Value: "25"},
		{Label: "30%", Value: "30"},
		{Label: "40%", Value: "40"},
		{Label: "50%", Value: "50"},
	}
	rateFound := false
	for _, opt := range rateOptions {
		if opt.Value == strconv.Itoa(cfg.Rate) {
			rateFound = true
			break
		}
	}
	if !rateFound {
		rateOptions = append([]discord.StringSelectMenuOption{
			{Label: fmt.Sprintf("%d%%", cfg.Rate), Value: strconv.Itoa(cfg.Rate)},
		}, rateOptions...)
	}
	for i, opt := range rateOptions {
		if opt.Value == strconv.Itoa(cfg.Rate) {
			rateOptions[i].Default = true
		}
	}

	intervalOptions := []discord.StringSelectMenuOption{
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_interval_1day"), Value: "1"},
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_interval_3days"), Value: "3"},
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_interval_1week"), Value: "7"},
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_interval_2weeks"), Value: "14"},
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_interval_1month"), Value: "30"},
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_interval_3months"), Value: "90"},
	}
	intervalFound := false
	for _, opt := range intervalOptions {
		if opt.Value == strconv.Itoa(cfg.IntervalDays) {
			intervalFound = true
			break
		}
	}
	if !intervalFound && cfg.IntervalDays > 0 {
		intervalOptions = append([]discord.StringSelectMenuOption{
			{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_interval_days", map[string]any{"days": cfg.IntervalDays}), Value: strconv.Itoa(cfg.IntervalDays)},
		}, intervalOptions...)
	}
	for i, opt := range intervalOptions {
		if opt.Value == strconv.Itoa(cfg.IntervalDays) {
			intervalOptions[i].Default = true
		}
	}

	enabledOptions := []discord.StringSelectMenuOption{
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_enabled_label"), Value: "true"},
		{Label: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_disabled_label"), Value: "false"},
	}
	for i, opt := range enabledOptions {
		if opt.Value == strconv.FormatBool(cfg.Enabled) {
			enabledOptions[i].Default = true
		}
	}

	modal := discord.NewModalCreateBuilder().
		SetTitle(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_modal_title")).
		SetCustomID("gopoint:tax_setup_modal").
		SetComponents(
			discord.NewLabel(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_label_rate"),
				discord.StringSelectMenuComponent{
					CustomID:  "rate",
					MinValues: ptr(1),
					MaxValues: 1,
					Options:   rateOptions,
				}),
			discord.NewLabel(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_label_min_points"),
				discord.TextInputComponent{
					CustomID: "min_points",
					Style:    discord.TextInputStyleShort,
					Value:    strconv.FormatInt(cfg.MinPoints, 10),
					Required: true,
				}),
			discord.NewLabel(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_label_interval"),
				discord.StringSelectMenuComponent{
					CustomID:  "interval_days",
					MinValues: ptr(1),
					MaxValues: 1,
					Options:   intervalOptions,
				}),
			discord.NewLabel(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_label_enabled"),
				discord.StringSelectMenuComponent{
					CustomID:  "enabled",
					MinValues: ptr(1),
					MaxValues: 1,
					Options:   enabledOptions,
				}),
			discord.NewLabel(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_label_brackets"),
				discord.TextInputComponent{
					CustomID:    "brackets",
					Style:       discord.TextInputStyleParagraph,
					Value:       cfg.Brackets,
					Placeholder: i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_placeholder_brackets"),
					Required:    false,
				}),
		).
		Build()

	if err := event.Modal(modal); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func TaxSetupModalSubmitHandler(c *components.Components, event *events.ModalSubmitInteractionCreate) errors.Error {
	guildID := *event.GuildID()

	var rateStr string
	if rates := event.Data.StringValues("rate"); len(rates) > 0 {
		rateStr = rates[0]
	}
	minPointsStr := event.Data.Text("min_points")
	var intervalDaysStr string
	if intervals := event.Data.StringValues("interval_days"); len(intervals) > 0 {
		intervalDaysStr = intervals[0]
	}
	var enabledStr string
	if enableds := event.Data.StringValues("enabled"); len(enableds) > 0 {
		enabledStr = enableds[0]
	}
	bracketsStr := event.Data.Text("brackets")

	rate, err := strconv.Atoi(rateStr)
	if err != nil || rate < 0 || rate > 100 {
		return respondModalError(event, i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_err_rate_range"))
	}

	minPoints, err := strconv.ParseInt(minPointsStr, 10, 64)
	if err != nil || minPoints < 0 {
		return respondModalError(event, i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_err_min_points"))
	}

	intervalDays, err := strconv.Atoi(intervalDaysStr)
	if err != nil || intervalDays < 1 || intervalDays > 365 {
		return respondModalError(event, i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_err_interval_range"))
	}

	enabled := false
	enabledStrClean := strings.ToLower(strings.TrimSpace(enabledStr))
	if enabledStrClean == "true" || enabledStrClean == "1" || enabledStrClean == "yes" {
		enabled = true
	}

	bracketsText := strings.ReplaceAll(bracketsStr, "\r\n", "\n")
	_, errParse := parseTaxBrackets(event.Locale(), bracketsText)
	if errParse != nil {
		return respondModalError(event, i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_err_brackets_parse", map[string]any{"error": errParse.Error()}))
	}

	var cfg models.GoPointTaxConfig
	errLoad := c.GormDB().Where("guild_id = ?", guildID).First(&cfg).Error
	if errLoad != nil {
		if errors.Is(errLoad, gorm.ErrRecordNotFound) {
			cfg = models.GoPointTaxConfig{
				GuildID:      guildID,
				Rate:         rate,
				MinPoints:    minPoints,
				IntervalDays: intervalDays,
				Enabled:      enabled,
				Brackets:     bracketsText,
				NextTaxTime:  time.Now().Add(time.Duration(intervalDays) * 24 * time.Hour),
			}
			if errCreate := c.GormDB().Create(&cfg).Error; errCreate != nil {
				return errors.NewError(errCreate)
			}
		} else {
			return errors.NewError(errLoad)
		}
	} else {
		prevInterval := cfg.IntervalDays
		cfg.Rate = rate
		cfg.MinPoints = minPoints
		cfg.IntervalDays = intervalDays
		cfg.Enabled = enabled
		cfg.Brackets = bracketsText
		if cfg.NextTaxTime.Before(time.Now()) || cfg.NextTaxTime.IsZero() || prevInterval != intervalDays {
			cfg.NextTaxTime = time.Now().Add(time.Duration(intervalDays) * 24 * time.Hour)
		}
		if errSave := c.GormDB().Save(&cfg).Error; errSave != nil {
			return errors.NewError(errSave)
		}
	}

	statusStr := i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_status_disabled")
	if cfg.Enabled {
		statusStr = i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_status_enabled")
	}

	msg := i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_success", map[string]any{
		"rate":       cfg.Rate,
		"min_points": cfg.MinPoints,
		"interval":   cfg.IntervalDays,
		"status":     statusStr,
		"next_time":  discord.NewTimestamp(discord.TimestampStyleLongDateTime, cfg.NextTaxTime).String(),
	})

	if errResp := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(true).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(msg),
			).WithAccentColor(0x2ECC71),
		),
	); errResp != nil {
		return errors.NewError(errResp)
	}

	return nil
}

func respondModalError(event *events.ModalSubmitInteractionCreate, text string) errors.Error {
	_ = event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(true).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_setup_error_title") + text),
			).WithAccentColor(0xE74C3C),
		),
	)
	return nil
}

func TaxStatusHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	guildID := *event.GuildID()

	var cfg models.GoPointTaxConfig
	err := c.GormDB().Where("guild_id = ?", guildID).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cfg = models.GoPointTaxConfig{
				GuildID:      guildID,
				Rate:         0,
				IntervalDays: 0,
				Enabled:      false,
				MinPoints:    0,
				Brackets:     "",
			}
		} else {
			return errors.NewError(err)
		}
	}

	statusStr := i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_status_disabled")
	if cfg.Enabled {
		statusStr = i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_status_enabled")
	}

	nextTimeStr := "-"
	if !cfg.NextTaxTime.IsZero() {
		nextTimeStr = discord.NewTimestamp(discord.TimestampStyleLongDateTime, cfg.NextTaxTime).String()
	}

	bracketsText := cfg.Brackets
	if bracketsText == "" {
		bracketsText = i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_brackets_none")
	} else {
		bracketsText = "```\n" + bracketsText + "\n```"
	}

	var sb strings.Builder
	sb.WriteString(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_status_info", map[string]any{
		"enabled":    statusStr,
		"rate":       cfg.Rate,
		"min_points": cfg.MinPoints,
		"interval":   cfg.IntervalDays,
		"next_time":  nextTimeStr,
		"brackets":   bracketsText,
	}))

	var pending []models.GoPointPendingTax
	errFind := c.GormDB().Where("guild_id = ? AND collected = ? AND exempted = ?", guildID, false, false).
		Order("collect_time asc").Limit(10).Find(&pending).Error
	if errFind != nil {
		return errors.NewError(errFind)
	}
	if len(pending) > 0 {
		sb.WriteString(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_status_pending_title"))
		for _, p := range pending {
			sb.WriteString(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_status_pending_item", map[string]any{
				"user_id":      p.UserID.String(),
				"tax_amount":   p.TaxAmount,
				"collect_time": discord.NewTimestamp(discord.TimestampStyleLongDateTime, p.CollectTime).String(),
			}))
		}
	} else {
		sb.WriteString(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_status_no_pending"))
	}

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(true).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(sb.String()),
			).WithAccentColor(0x3498DB),
		),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func TaxForceHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	guildID := *event.GuildID()
	now := time.Now()

	overwrite := false
	if opt, ok := event.SlashCommandInteractionData().OptBool("overwrite"); ok {
		overwrite = opt
	}

	var cfg models.GoPointTaxConfig
	var calcCount int
	err := c.GormDB().Where("guild_id = ?", guildID).First(&cfg).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}
	if err == nil && cfg.Enabled {
		var points []models.GoPoint
		errPoints := c.GormDB().Where("guild_id = ? AND points > 0", guildID).Find(&points).Error
		if errPoints != nil {
			return errors.NewError(errPoints)
		}
		errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
			for _, p := range points {
				// Check if there is already a pending tax for this user in this guild
				var count int64
				errCount := tx.Model(&models.GoPointPendingTax{}).
					Where("guild_id = ? AND user_id = ? AND collected = ? AND exempted = ?", guildID, p.UserID, false, false).
					Count(&count).Error
				if errCount != nil {
					return errCount
				}
				if count > 0 {
					if overwrite {
						if errDel := tx.Where("guild_id = ? AND user_id = ? AND collected = ? AND exempted = ?", guildID, p.UserID, false, false).Delete(&models.GoPointPendingTax{}).Error; errDel != nil {
							return errDel
						}
					} else {
						continue // Already has a pending tax, skip to avoid double tax
					}
				}

				taxAmount := calculateUserTaxAmount(p.Points, &cfg)
				exempted := (taxAmount == 0)

				pending := models.GoPointPendingTax{
					ID:            uuid.New(),
					GuildID:       guildID,
					UserID:        p.UserID,
					BasePoints:    p.Points,
					TaxAmount:     taxAmount,
					CalculateTime: now,
					CollectTime:   now.Add(7 * 24 * time.Hour), // 1 week later
					Collected:     false,
					Exempted:      exempted,
				}
				if errCreate := tx.Create(&pending).Error; errCreate != nil {
					return errCreate
				}
				calcCount++
			}
			cfg.NextTaxTime = now.Add(time.Duration(cfg.IntervalDays) * 24 * time.Hour)
			return tx.Save(&cfg).Error
		})
		if errTx != nil {
			slog.Error("failed force tax determination transaction", "guild_id", guildID, "error", errTx)
			return errors.NewError(errTx)
		}
	}

	var pending []models.GoPointPendingTax
	var collectedCount, exemptedCount int
	errFind := c.GormDB().Where("guild_id = ? AND collected = ? AND exempted = ? AND collect_time <= ?", guildID, false, false, now).Find(&pending).Error
	if errFind != nil {
		return errors.NewError(errFind)
	}
	for _, tax := range pending {
		errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
			var p models.GoPoint
			if err := tx.Where("user_id = ? AND guild_id = ?", tax.UserID, tax.GuildID).First(&p).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					tax.Exempted = true
					return tx.Save(&tax).Error
				}
				return err
			}

			if p.Points < tax.BasePoints {
				tax.Exempted = true
				exemptedCount++
			} else {
				p.Points -= tax.TaxAmount
				if p.Points < 0 {
					p.Points = 0
				}
				if err := tx.Save(&p).Error; err != nil {
					return err
				}
				tax.Collected = true
				collectedCount++
			}
			return tx.Save(&tax).Error
		})
		if errTx != nil {
			slog.Error("failed force tax collection transaction", "tax_id", tax.ID, "error", errTx)
			return errors.NewError(errTx)
		}
	}

	var sb strings.Builder
	sb.WriteString(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_force_determination", map[string]any{
		"count": calcCount,
	}))
	sb.WriteString("\n")
	sb.WriteString(i18n.TranslateText(event.Locale(), "components.gopoint.admin.tax_force_collection", map[string]any{
		"collected": collectedCount,
		"exempted":  exemptedCount,
	}))

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(true).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(sb.String()),
			).WithAccentColor(0xE67E22),
		),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func ResetPointsHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	data := event.SlashCommandInteractionData()
	points, hasPoints := data.OptInt("points")
	var targetPoints int64 = 0
	if hasPoints {
		targetPoints = int64(points)
	}

	byLevel := false
	if opt, ok := data.OptBool("by_level"); ok {
		byLevel = opt
	}

	guildID := *event.GuildID()

	var msg string
	if byLevel {
		err := c.GormDB().Transaction(func(tx *gorm.DB) error {
			var members []models.Member
			if err := tx.Where("guild_id = ?", guildID).Find(&members).Error; err != nil {
				return err
			}

			updatedUserIDs := make(map[snowflake.ID]bool)

			for _, m := range members {
				level := m.XP.Level()
				pts := int64(xppoint.TotalPoint(level))

				res := tx.Model(&models.GoPoint{}).
					Where("user_id = ? AND guild_id = ?", m.UserID, guildID).
					Update("points", pts)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					gp := models.GoPoint{
						UserID:  m.UserID,
						GuildID: guildID,
						Points:  pts,
					}
					if errCreate := tx.Create(&gp).Error; errCreate != nil {
						return errCreate
					}
				}
				updatedUserIDs[m.UserID] = true
			}

			var otherGoPoints []models.GoPoint
			if err := tx.Where("guild_id = ?", guildID).Find(&otherGoPoints).Error; err != nil {
				return err
			}
			for _, gp := range otherGoPoints {
				if !updatedUserIDs[gp.UserID] {
					gp.Points = 0
					if errSave := tx.Save(&gp).Error; errSave != nil {
						return errSave
					}
				}
			}
			return nil
		})
		if err != nil {
			return errors.NewError(err)
		}
		msg = i18n.TranslateText(event.Locale(), "components.gopoint.admin.reset_by_level_success")
	} else {
		if err := c.GormDB().Model(&models.GoPoint{}).Where("guild_id = ?", guildID).Update("points", targetPoints).Error; err != nil {
			return errors.NewError(err)
		}
		msg = i18n.TranslateText(event.Locale(), "components.gopoint.admin.reset_success", map[string]any{
			"points": targetPoints,
		})
	}

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(false).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(msg),
			).WithAccentColor(0xC0392B),
		),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func criteriaDisplayName(locale discord.Locale, c string) string {
	if c == "final" {
		return i18n.TranslateText(locale, "components.gopoint.admin.season_criteria_final")
	}
	return i18n.TranslateText(locale, "components.gopoint.admin.season_criteria_earned")
}

func SeasonStartHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	data := event.SlashCommandInteractionData()
	name := data.String("name")
	durationDays := data.Int("duration_days")
	startDelayHours, hasDelay := data.OptInt("start_delay_hours")
	criteria := data.String("criteria")
	if criteria == "" {
		criteria = "earned"
	}

	guildID := *event.GuildID()

	var active models.GoPointSeason
	err := c.GormDB().Where("guild_id = ? AND is_active = ?", guildID, true).First(&active).Error
	if err == nil {
		if errResp := event.RespondMessage(discord.NewMessageBuilder().
			SetEphemeral(true).
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_err_active_exists", map[string]any{
						"name": active.Name,
					})),
				).WithAccentColor(0xE74C3C),
			),
		); errResp != nil {
			return errors.NewError(errResp)
		}
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NewError(err)
	}

	startTime := time.Now()
	if hasDelay && startDelayHours > 0 {
		startTime = startTime.Add(time.Duration(startDelayHours) * time.Hour)
	}
	endTime := startTime.AddDate(0, 0, durationDays)

	season := models.GoPointSeason{
		ID:         uuid.New(),
		GuildID:    guildID,
		Name:       name,
		StartTime:  startTime,
		EndTime:    endTime,
		IsActive:   (startTime.Before(time.Now()) || startTime.Equal(time.Now())),
		HasAwarded: false,
		ChannelID:  event.Channel().ID(),
		Criteria:   criteria,
	}

	errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
		var overlap models.GoPointSeason
		errOverlap := tx.Where("guild_id = ? AND has_awarded = ? AND start_time < ? AND end_time > ?", guildID, false, endTime, startTime).First(&overlap).Error
		if errOverlap == nil {
			return fmt.Errorf("overlap:%s", overlap.Name)
		}
		if !errors.Is(errOverlap, gorm.ErrRecordNotFound) {
			return errOverlap
		}

		if errCreate := tx.Create(&season).Error; errCreate != nil {
			return errCreate
		}
		return nil
	})

	if errTx != nil {
		if after, ok := strings.CutPrefix(errTx.Error(), "overlap:"); ok {
			overlapName := after
			if errResp := event.RespondMessage(discord.NewMessageBuilder().
				SetEphemeral(true).
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_err_active_exists", map[string]any{
							"name": overlapName,
						})),
					).WithAccentColor(0xE74C3C),
				),
			); errResp != nil {
				return errors.NewError(errResp)
			}
			return nil
		}
		slog.Error("failed to create new season", "guild_id", guildID, "error", errTx)
		return errors.NewError(errTx)
	}

	var msg string
	if season.IsActive {
		msg = i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_start_success", map[string]any{
			"name":     season.Name,
			"duration": durationDays,
			"criteria": criteriaDisplayName(event.Locale(), criteria),
			"end_time": discord.NewTimestamp(discord.TimestampStyleLongDateTime, season.EndTime).String(),
		})
	} else {
		msg = i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_start_scheduled", map[string]any{
			"name":       season.Name,
			"criteria":   criteriaDisplayName(event.Locale(), criteria),
			"start_time": discord.NewTimestamp(discord.TimestampStyleLongDateTime, season.StartTime).String(),
			"end_time":   discord.NewTimestamp(discord.TimestampStyleLongDateTime, season.EndTime).String(),
		})
	}

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(false).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(msg),
			).WithAccentColor(0x9B59B6),
		),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func SeasonEndHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	guildID := *event.GuildID()

	var active models.GoPointSeason
	if err := c.GormDB().Where("guild_id = ? AND is_active = ?", guildID, true).First(&active).Error; err != nil {
		if errResp := event.RespondMessage(discord.NewMessageBuilder().
			SetEphemeral(true).
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_err_no_active")),
				).WithAccentColor(0xE74C3C),
			),
		); errResp != nil {
			return errors.NewError(errResp)
		}
		return nil
	}

	errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
		active.IsActive = false
		active.HasAwarded = true
		if err := tx.Save(&active).Error; err != nil {
			return err
		}
		if active.Criteria == "final" {
			var gp []models.GoPoint
			if err := tx.Where("guild_id = ?", active.GuildID).Find(&gp).Error; err != nil {
				return err
			}
			for _, p := range gp {
				su := models.GoPointSeasonUser{
					SeasonID:     active.ID,
					UserID:       p.UserID,
					GuildID:      active.GuildID,
					PointsEarned: p.Points,
				}
				if err := tx.Create(&su).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if errTx != nil {
		return errors.NewError(errTx)
	}

	announceSeasonResults(c, event.Client(), active.ID)

	msg := i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_end_early", map[string]any{
		"name": active.Name,
	})

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(true).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(msg),
			).WithAccentColor(0x7F8C8D),
		),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func SeasonStatusHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	guildID := *event.GuildID()

	var active models.GoPointSeason
	if err := c.GormDB().Where("guild_id = ? AND is_active = ?", guildID, true).First(&active).Error; err != nil {
		if errResp := event.RespondMessage(discord.NewMessageBuilder().
			SetEphemeral(true).
			SetIsComponentsV2(true).
			SetComponents(
				discord.NewContainer(
					discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_err_no_active")),
				).WithAccentColor(0xE74C3C),
			),
		); errResp != nil {
			return errors.NewError(errResp)
		}
		return nil
	}

	var records []models.GoPointSeasonUser
	if active.Criteria == "final" {
		var gp []models.GoPoint
		errFind := c.GormDB().Where("guild_id = ?", guildID).Order("points desc").Limit(10).Find(&gp).Error
		if errFind != nil {
			return errors.NewError(errFind)
		}
		for _, p := range gp {
			records = append(records, models.GoPointSeasonUser{
				UserID:       p.UserID,
				PointsEarned: p.Points,
			})
		}
	} else {
		if err := c.GormDB().Where("season_id = ?", active.ID).Order("points_earned desc").Limit(10).Find(&records).Error; err != nil {
			return errors.NewError(err)
		}
	}

	var rankingLines []string
	if len(records) == 0 {
		rankingLines = append(rankingLines, i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_ranking_no_data"))
	} else {
		for i, r := range records {
			medal := ""
			switch i {
			case 0:
				medal = "🥇 "
			case 1:
				medal = "🥈 "
			case 2:
				medal = "🥉 "
			default:
				medal = fmt.Sprintf("#%d ", i+1)
			}
			rankingLines = append(rankingLines, i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_ranking_entry", map[string]any{
				"medal":   medal,
				"user_id": r.UserID.String(),
				"points":  r.PointsEarned,
			}))
		}
	}

	remaining := time.Until(active.EndTime).Round(time.Minute)
	statusStr := i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_status_active")
	if remaining < 0 {
		remaining = 0
		statusStr = i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_status_ended")
	}

	msg := i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_status_info", map[string]any{
		"name":       active.Name,
		"status":     statusStr,
		"criteria":   criteriaDisplayName(event.Locale(), active.Criteria),
		"start_time": discord.NewTimestamp(discord.TimestampStyleLongDateTime, active.StartTime).String(),
		"end_time":   discord.NewTimestamp(discord.TimestampStyleLongDateTime, active.EndTime).String(),
		"remaining":  discord.NewTimestamp(discord.TimestampStyleRelative, time.Now().Add(remaining)).String(),
		"ranking":    strings.Join(rankingLines, "\n"),
	})

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(true).
		SetAllowedMentions(&discord.AllowedMentions{}).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(msg),
			).WithAccentColor(0x9B59B6),
		),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func SeasonRankingHandler(c *components.Components, event *events.ApplicationCommandInteractionCreate) errors.Error {
	data := event.SlashCommandInteractionData()
	seasonIDStr, hasID := data.OptString("season_id")

	guildID := *event.GuildID()

	var season models.GoPointSeason
	if hasID {
		sUUID, err := uuid.Parse(seasonIDStr)
		if err != nil {
			if errResp := event.RespondMessage(discord.NewMessageBuilder().
				SetEphemeral(true).
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_not_found")),
					).WithAccentColor(0xE74C3C),
				),
			); errResp != nil {
				return errors.NewError(errResp)
			}
			return nil
		}
		if errLoad := c.GormDB().Where("id = ? AND guild_id = ?", sUUID, guildID).First(&season).Error; errLoad != nil {
			if errResp := event.RespondMessage(discord.NewMessageBuilder().
				SetEphemeral(true).
				SetIsComponentsV2(true).
				SetComponents(
					discord.NewContainer(
						discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_not_found")),
					).WithAccentColor(0xE74C3C),
				),
			); errResp != nil {
				return errors.NewError(errResp)
			}
			return nil
		}
	} else {
		if errLoad := c.GormDB().Where("guild_id = ? AND is_active = ?", guildID, true).First(&season).Error; errLoad != nil {
			if errLatest := c.GormDB().Where("guild_id = ?", guildID).Order("end_time desc").First(&season).Error; errLatest != nil {
				if errResp := event.RespondMessage(discord.NewMessageBuilder().
					SetEphemeral(true).
					SetIsComponentsV2(true).
					SetComponents(
						discord.NewContainer(
							discord.NewTextDisplay(i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_err_no_active")),
						).WithAccentColor(0xE74C3C),
					),
				); errResp != nil {
					return errors.NewError(errResp)
				}
				return nil
			}
		}
	}

	var records []models.GoPointSeasonUser
	if season.Criteria == "final" {
		var gp []models.GoPoint
		errFind := c.GormDB().Where("guild_id = ?", guildID).Order("points desc").Limit(20).Find(&gp).Error
		if errFind != nil {
			return errors.NewError(errFind)
		}
		for _, p := range gp {
			records = append(records, models.GoPointSeasonUser{
				UserID:       p.UserID,
				PointsEarned: p.Points,
			})
		}
	} else {
		if err := c.GormDB().Where("season_id = ?", season.ID).Order("points_earned desc").Limit(20).Find(&records).Error; err != nil {
			return errors.NewError(err)
		}
	}

	var rankingLines []string
	if len(records) == 0 {
		rankingLines = append(rankingLines, i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_ranking_no_data"))
	} else {
		for i, r := range records {
			medal := ""
			switch i {
			case 0:
				medal = "🥇 "
			case 1:
				medal = "🥈 "
			case 2:
				medal = "🥉 "
			default:
				medal = fmt.Sprintf("`#%d` ", i+1)
			}
			rankingLines = append(rankingLines, i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_ranking_entry", map[string]any{
				"medal":   medal,
				"user_id": r.UserID.String(),
				"points":  r.PointsEarned,
			}))
		}
	}

	title := i18n.TranslateText(event.Locale(), "components.gopoint.admin.season_ranking_title", map[string]any{
		"name":     season.Name,
		"criteria": criteriaDisplayName(event.Locale(), season.Criteria),
	})

	msg := title + strings.Join(rankingLines, "\n")

	if err := event.RespondMessage(discord.NewMessageBuilder().
		SetEphemeral(false).
		SetAllowedMentions(&discord.AllowedMentions{}).
		SetIsComponentsV2(true).
		SetComponents(
			discord.NewContainer(
				discord.NewTextDisplay(msg),
			).WithAccentColor(0x9B59B6),
		),
	); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func ProcessBackgroundTasks(c *components.Components, client *bot.Client) error {
	now := time.Now()

	// 1. Tax calculations (Determination Phase)
	var activeConfigs []models.GoPointTaxConfig
	if err := c.GormDB().Where("enabled = ? AND next_tax_time <= ?", true, now).Find(&activeConfigs).Error; err != nil {
		return err
	}
	for _, cfg := range activeConfigs {
		var points []models.GoPoint
		errPoints := c.GormDB().Where("guild_id = ? AND points > 0", cfg.GuildID).Find(&points).Error
		if errPoints != nil {
			slog.Error("failed to find points for tax", "guild_id", cfg.GuildID, "error", errPoints)
			return errPoints
		}
		errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
			for _, p := range points {
				// Check if there is already a pending tax for this user in this guild
				var count int64
				errCount := tx.Model(&models.GoPointPendingTax{}).
					Where("guild_id = ? AND user_id = ? AND collected = ? AND exempted = ?", cfg.GuildID, p.UserID, false, false).
					Count(&count).Error
				if errCount != nil {
					return errCount
				}
				if count > 0 {
					continue // Already has a pending tax, skip to avoid double tax
				}

				taxAmount := calculateUserTaxAmount(p.Points, &cfg)
				exempted := (taxAmount == 0)

				pending := models.GoPointPendingTax{
					ID:            uuid.New(),
					GuildID:       cfg.GuildID,
					UserID:        p.UserID,
					BasePoints:    p.Points,
					TaxAmount:     taxAmount,
					CalculateTime: now,
					CollectTime:   now.Add(7 * 24 * time.Hour), // 1 week later
					Collected:     false,
					Exempted:      exempted,
				}
				if errPending := tx.Create(&pending).Error; errPending != nil {
					return errPending
				}
			}
			next := cfg.NextTaxTime.Add(time.Duration(cfg.IntervalDays) * 24 * time.Hour)
			if next.Before(now) {
				next = now.Add(time.Duration(cfg.IntervalDays) * 24 * time.Hour)
			}
			cfg.NextTaxTime = next
			return tx.Save(&cfg).Error
		})
		if errTx != nil {
			slog.Error("failed to process tax determination transaction", "guild_id", cfg.GuildID, "error", errTx)
			return errTx
		}
	}

	// 2. Tax executions (Collection Phase)
	var pendingTaxes []models.GoPointPendingTax
	if err := c.GormDB().Where("collected = ? AND exempted = ? AND collect_time <= ?", false, false, now).Find(&pendingTaxes).Error; err != nil {
		return err
	}
	for _, tax := range pendingTaxes {
		errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
			var p models.GoPoint
			if err := tx.Where("user_id = ? AND guild_id = ?", tax.UserID, tax.GuildID).First(&p).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					tax.Exempted = true
					return tx.Save(&tax).Error
				}
				return err
			}

			if p.Points < tax.BasePoints {
				tax.Exempted = true
			} else {
				p.Points -= tax.TaxAmount
				if p.Points < 0 {
					p.Points = 0
				}
				if err := tx.Save(&p).Error; err != nil {
					return err
				}
				tax.Collected = true
			}
			return tx.Save(&tax).Error
		})
		if errTx != nil {
			slog.Error("failed to execute tax collection transaction", "tax_id", tax.ID, "error", errTx)
			return errTx
		}
	}

	// 3. Active seasons ending
	var activeSeasons []models.GoPointSeason
	if err := c.GormDB().Where("is_active = ? AND end_time <= ?", true, now).Find(&activeSeasons).Error; err != nil {
		return err
	}
	for _, s := range activeSeasons {
		errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
			s.IsActive = false
			s.HasAwarded = true
			if err := tx.Save(&s).Error; err != nil {
				return err
			}
			if s.Criteria == "final" {
				var gp []models.GoPoint
				if err := tx.Where("guild_id = ?", s.GuildID).Find(&gp).Error; err != nil {
					return err
				}
				for _, p := range gp {
					su := models.GoPointSeasonUser{
						SeasonID:     s.ID,
						UserID:       p.UserID,
						GuildID:      s.GuildID,
						PointsEarned: p.Points,
					}
					if err := tx.Create(&su).Error; err != nil {
						return err
					}
				}
			}
			return nil
		})
		if errTx == nil {
			if s.ChannelID != 0 {
				go announceSeasonResults(c, client, s.ID)
			}
		} else {
			slog.Error("failed to close season transaction", "season_id", s.ID, "error", errTx)
			return errTx
		}
	}

	// 4. Starting scheduled seasons
	var scheduledSeasons []models.GoPointSeason
	if err := c.GormDB().Where("is_active = ? AND has_awarded = ? AND start_time <= ? AND end_time > ?", false, false, now, now).Find(&scheduledSeasons).Error; err != nil {
		return err
	}
	for _, s := range scheduledSeasons {
		errTx := c.GormDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.GoPointSeason{}).
				Where("guild_id = ? AND is_active = ?", s.GuildID, true).
				Update("is_active", false).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			s.IsActive = true
			return tx.Save(&s).Error
		})
		if errTx == nil {
			if s.ChannelID != 0 {
				_, _ = client.Rest.CreateMessage(s.ChannelID, discord.NewMessageCreateBuilder().
					SetAllowedMentions(&discord.AllowedMentions{}).
					SetContent(i18n.TranslateText(discord.LocaleUnknown, "components.gopoint.admin.season_start_announcement", map[string]any{
						"name":     s.Name,
						"criteria": criteriaDisplayName(discord.LocaleUnknown, s.Criteria),
						"end_time": discord.NewTimestamp(discord.TimestampStyleLongDateTime, s.EndTime).String(),
					})).
					Build())
			}
		} else {
			slog.Error("failed to start scheduled season transaction", "season_id", s.ID, "error", errTx)
			return errTx
		}
	}

	return nil
}

func announceSeasonResults(c *components.Components, client *bot.Client, seasonID uuid.UUID) {
	var season models.GoPointSeason
	if err := c.GormDB().Where("id = ?", seasonID).First(&season).Error; err != nil {
		return
	}

	var records []models.GoPointSeasonUser
	if err := c.GormDB().Where("season_id = ?", seasonID).Order("points_earned desc").Limit(10).Find(&records).Error; err != nil {
		return
	}

	var sb strings.Builder
	sb.WriteString(i18n.TranslateText(discord.LocaleUnknown, "components.gopoint.admin.season_end_announcement_title", map[string]any{
		"name":     season.Name,
		"criteria": criteriaDisplayName(discord.LocaleUnknown, season.Criteria),
	}))

	if len(records) == 0 {
		sb.WriteString(i18n.TranslateText(discord.LocaleUnknown, "components.gopoint.admin.season_end_announcement_no_data"))
	} else {
		for i, r := range records {
			medal := ""
			switch i {
			case 0:
				medal = "🥇 "
			case 1:
				medal = "🥈 "
			case 2:
				medal = "🥉 "
			default:
				medal = fmt.Sprintf("`#%d` ", i+1)
			}

			user, err := client.Rest.GetMember(season.GuildID, r.UserID)
			name := fmt.Sprintf("<@%d>", r.UserID)
			if err == nil && user != nil {
				name = fmt.Sprintf("**%s**", user.EffectiveName())
			}
			sb.WriteString(i18n.TranslateText(discord.LocaleUnknown, "components.gopoint.admin.season_end_announcement_entry", map[string]any{
				"medal":  medal,
				"name":   name,
				"points": r.PointsEarned,
			}))
		}
	}

	channelID := season.ChannelID
	if channelID != 0 {
		_, _ = client.Rest.CreateMessage(channelID, discord.NewMessageCreateBuilder().
			SetAllowedMentions(&discord.AllowedMentions{}).
			SetContent(sb.String()).
			Build())
	}
}

func seasonAutocomplete(c *components.Components, event *events.AutocompleteInteractionCreate) errors.Error {
	query := event.Data.String("season_id")

	var seasons []models.GoPointSeason
	err := c.GormDB().
		Where("guild_id = ? AND (name LIKE ? OR id LIKE ?)", *event.GuildID(), "%"+escapeLike(query)+"%", "%"+escapeLike(query)+"%").
		Limit(25).
		Find(&seasons).Error
	if err != nil {
		return errors.NewError(err)
	}

	choices := make([]discord.AutocompleteChoice, len(seasons))
	for i, s := range seasons {
		choices[i] = discord.AutocompleteChoiceString{
			Name:  fmt.Sprintf("%s (%s)", s.Name, s.ID.String()),
			Value: s.ID.String(),
		}
	}
	if err := event.AutocompleteResult(choices); err != nil {
		return errors.NewError(err)
	}
	return nil
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

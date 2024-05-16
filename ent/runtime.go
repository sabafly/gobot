// Code generated by ent, DO NOT EDIT.

package ent

import (
	"time"

	"github.com/disgoorg/disgo/discord"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/chinchiroplayer"
	"github.com/sabafly/gobot/ent/chinchirosession"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/messagepin"
	"github.com/sabafly/gobot/ent/messageremind"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepaneledit"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/schema"
	"github.com/sabafly/gobot/ent/user"
	"github.com/sabafly/gobot/ent/wordsuffix"
	"github.com/sabafly/gobot/internal/permissions"
	"github.com/sabafly/gobot/internal/xppoint"
)

// The init function reads all schema descriptors with runtime code
// (default values, validators, hooks and policies) and stitches it
// to their package variables.
func init() {
	chinchiroplayerFields := schema.ChinchiroPlayer{}.Fields()
	_ = chinchiroplayerFields
	// chinchiroplayerDescPoint is the schema descriptor for point field.
	chinchiroplayerDescPoint := chinchiroplayerFields[1].Descriptor()
	// chinchiroplayer.DefaultPoint holds the default value on creation for the point field.
	chinchiroplayer.DefaultPoint = chinchiroplayerDescPoint.Default.(int)
	// chinchiroplayerDescIsOwner is the schema descriptor for is_owner field.
	chinchiroplayerDescIsOwner := chinchiroplayerFields[2].Descriptor()
	// chinchiroplayer.DefaultIsOwner holds the default value on creation for the is_owner field.
	chinchiroplayer.DefaultIsOwner = chinchiroplayerDescIsOwner.Default.(bool)
	// chinchiroplayerDescID is the schema descriptor for id field.
	chinchiroplayerDescID := chinchiroplayerFields[0].Descriptor()
	// chinchiroplayer.DefaultID holds the default value on creation for the id field.
	chinchiroplayer.DefaultID = chinchiroplayerDescID.Default.(func() uuid.UUID)
	chinchirosessionFields := schema.ChinchiroSession{}.Fields()
	_ = chinchirosessionFields
	// chinchirosessionDescTurn is the schema descriptor for turn field.
	chinchirosessionDescTurn := chinchirosessionFields[1].Descriptor()
	// chinchirosession.DefaultTurn holds the default value on creation for the turn field.
	chinchirosession.DefaultTurn = chinchirosessionDescTurn.Default.(int)
	// chinchirosessionDescLoop is the schema descriptor for loop field.
	chinchirosessionDescLoop := chinchirosessionFields[2].Descriptor()
	// chinchirosession.DefaultLoop holds the default value on creation for the loop field.
	chinchirosession.DefaultLoop = chinchirosessionDescLoop.Default.(int)
	// chinchirosessionDescID is the schema descriptor for id field.
	chinchirosessionDescID := chinchirosessionFields[0].Descriptor()
	// chinchirosession.DefaultID holds the default value on creation for the id field.
	chinchirosession.DefaultID = chinchirosessionDescID.Default.(func() uuid.UUID)
	guildFields := schema.Guild{}.Fields()
	_ = guildFields
	// guildDescName is the schema descriptor for name field.
	guildDescName := guildFields[1].Descriptor()
	// guild.NameValidator is a validator for the "name" field. It is called by the builders before save.
	guild.NameValidator = guildDescName.Validators[0].(func(string) error)
	// guildDescLocale is the schema descriptor for locale field.
	guildDescLocale := guildFields[2].Descriptor()
	// guild.DefaultLocale holds the default value on creation for the locale field.
	guild.DefaultLocale = discord.Locale(guildDescLocale.Default.(string))
	// guild.LocaleValidator is a validator for the "locale" field. It is called by the builders before save.
	guild.LocaleValidator = guildDescLocale.Validators[0].(func(string) error)
	// guildDescLevelUpMessage is the schema descriptor for level_up_message field.
	guildDescLevelUpMessage := guildFields[3].Descriptor()
	// guild.DefaultLevelUpMessage holds the default value on creation for the level_up_message field.
	guild.DefaultLevelUpMessage = guildDescLevelUpMessage.Default.(string)
	// guild.LevelUpMessageValidator is a validator for the "level_up_message" field. It is called by the builders before save.
	guild.LevelUpMessageValidator = guildDescLevelUpMessage.Validators[0].(func(string) error)
	// guildDescLevelMee6Imported is the schema descriptor for level_mee6_imported field.
	guildDescLevelMee6Imported := guildFields[6].Descriptor()
	// guild.DefaultLevelMee6Imported holds the default value on creation for the level_mee6_imported field.
	guild.DefaultLevelMee6Imported = guildDescLevelMee6Imported.Default.(bool)
	// guildDescLevelRole is the schema descriptor for level_role field.
	guildDescLevelRole := guildFields[7].Descriptor()
	// guild.DefaultLevelRole holds the default value on creation for the level_role field.
	guild.DefaultLevelRole = guildDescLevelRole.Default.(map[int]snowflake.ID)
	// guildDescPermissions is the schema descriptor for permissions field.
	guildDescPermissions := guildFields[8].Descriptor()
	// guild.DefaultPermissions holds the default value on creation for the permissions field.
	guild.DefaultPermissions = guildDescPermissions.Default.(map[snowflake.ID]permissions.Permission)
	// guildDescRemindCount is the schema descriptor for remind_count field.
	guildDescRemindCount := guildFields[9].Descriptor()
	// guild.DefaultRemindCount holds the default value on creation for the remind_count field.
	guild.DefaultRemindCount = guildDescRemindCount.Default.(int)
	// guildDescRolePanelEditTimes is the schema descriptor for role_panel_edit_times field.
	guildDescRolePanelEditTimes := guildFields[10].Descriptor()
	// guild.DefaultRolePanelEditTimes holds the default value on creation for the role_panel_edit_times field.
	guild.DefaultRolePanelEditTimes = guildDescRolePanelEditTimes.Default.([]time.Time)
	// guildDescBumpEnabled is the schema descriptor for bump_enabled field.
	guildDescBumpEnabled := guildFields[11].Descriptor()
	// guild.DefaultBumpEnabled holds the default value on creation for the bump_enabled field.
	guild.DefaultBumpEnabled = guildDescBumpEnabled.Default.(bool)
	// guildDescBumpMessageTitle is the schema descriptor for bump_message_title field.
	guildDescBumpMessageTitle := guildFields[12].Descriptor()
	// guild.DefaultBumpMessageTitle holds the default value on creation for the bump_message_title field.
	guild.DefaultBumpMessageTitle = guildDescBumpMessageTitle.Default.(string)
	// guild.BumpMessageTitleValidator is a validator for the "bump_message_title" field. It is called by the builders before save.
	guild.BumpMessageTitleValidator = guildDescBumpMessageTitle.Validators[0].(func(string) error)
	// guildDescBumpMessage is the schema descriptor for bump_message field.
	guildDescBumpMessage := guildFields[13].Descriptor()
	// guild.DefaultBumpMessage holds the default value on creation for the bump_message field.
	guild.DefaultBumpMessage = guildDescBumpMessage.Default.(string)
	// guild.BumpMessageValidator is a validator for the "bump_message" field. It is called by the builders before save.
	guild.BumpMessageValidator = guildDescBumpMessage.Validators[0].(func(string) error)
	// guildDescBumpRemindMessageTitle is the schema descriptor for bump_remind_message_title field.
	guildDescBumpRemindMessageTitle := guildFields[14].Descriptor()
	// guild.DefaultBumpRemindMessageTitle holds the default value on creation for the bump_remind_message_title field.
	guild.DefaultBumpRemindMessageTitle = guildDescBumpRemindMessageTitle.Default.(string)
	// guild.BumpRemindMessageTitleValidator is a validator for the "bump_remind_message_title" field. It is called by the builders before save.
	guild.BumpRemindMessageTitleValidator = guildDescBumpRemindMessageTitle.Validators[0].(func(string) error)
	// guildDescBumpRemindMessage is the schema descriptor for bump_remind_message field.
	guildDescBumpRemindMessage := guildFields[15].Descriptor()
	// guild.DefaultBumpRemindMessage holds the default value on creation for the bump_remind_message field.
	guild.DefaultBumpRemindMessage = guildDescBumpRemindMessage.Default.(string)
	// guild.BumpRemindMessageValidator is a validator for the "bump_remind_message" field. It is called by the builders before save.
	guild.BumpRemindMessageValidator = guildDescBumpRemindMessage.Validators[0].(func(string) error)
	// guildDescUpEnabled is the schema descriptor for up_enabled field.
	guildDescUpEnabled := guildFields[16].Descriptor()
	// guild.DefaultUpEnabled holds the default value on creation for the up_enabled field.
	guild.DefaultUpEnabled = guildDescUpEnabled.Default.(bool)
	// guildDescUpMessageTitle is the schema descriptor for up_message_title field.
	guildDescUpMessageTitle := guildFields[17].Descriptor()
	// guild.DefaultUpMessageTitle holds the default value on creation for the up_message_title field.
	guild.DefaultUpMessageTitle = guildDescUpMessageTitle.Default.(string)
	// guild.UpMessageTitleValidator is a validator for the "up_message_title" field. It is called by the builders before save.
	guild.UpMessageTitleValidator = guildDescUpMessageTitle.Validators[0].(func(string) error)
	// guildDescUpMessage is the schema descriptor for up_message field.
	guildDescUpMessage := guildFields[18].Descriptor()
	// guild.DefaultUpMessage holds the default value on creation for the up_message field.
	guild.DefaultUpMessage = guildDescUpMessage.Default.(string)
	// guild.UpMessageValidator is a validator for the "up_message" field. It is called by the builders before save.
	guild.UpMessageValidator = guildDescUpMessage.Validators[0].(func(string) error)
	// guildDescUpRemindMessageTitle is the schema descriptor for up_remind_message_title field.
	guildDescUpRemindMessageTitle := guildFields[19].Descriptor()
	// guild.DefaultUpRemindMessageTitle holds the default value on creation for the up_remind_message_title field.
	guild.DefaultUpRemindMessageTitle = guildDescUpRemindMessageTitle.Default.(string)
	// guild.UpRemindMessageTitleValidator is a validator for the "up_remind_message_title" field. It is called by the builders before save.
	guild.UpRemindMessageTitleValidator = guildDescUpRemindMessageTitle.Validators[0].(func(string) error)
	// guildDescUpRemindMessage is the schema descriptor for up_remind_message field.
	guildDescUpRemindMessage := guildFields[20].Descriptor()
	// guild.DefaultUpRemindMessage holds the default value on creation for the up_remind_message field.
	guild.DefaultUpRemindMessage = guildDescUpRemindMessage.Default.(string)
	// guild.UpRemindMessageValidator is a validator for the "up_remind_message" field. It is called by the builders before save.
	guild.UpRemindMessageValidator = guildDescUpRemindMessage.Validators[0].(func(string) error)
	memberFields := schema.Member{}.Fields()
	_ = memberFields
	// memberDescPermission is the schema descriptor for permission field.
	memberDescPermission := memberFields[0].Descriptor()
	// member.DefaultPermission holds the default value on creation for the permission field.
	member.DefaultPermission = memberDescPermission.Default.(permissions.Permission)
	// memberDescXp is the schema descriptor for xp field.
	memberDescXp := memberFields[1].Descriptor()
	// member.DefaultXp holds the default value on creation for the xp field.
	member.DefaultXp = xppoint.XP(memberDescXp.Default.(uint64))
	// memberDescMessageCount is the schema descriptor for message_count field.
	memberDescMessageCount := memberFields[4].Descriptor()
	// member.DefaultMessageCount holds the default value on creation for the message_count field.
	member.DefaultMessageCount = memberDescMessageCount.Default.(uint64)
	messagepinFields := schema.MessagePin{}.Fields()
	_ = messagepinFields
	// messagepinDescRateLimit is the schema descriptor for rate_limit field.
	messagepinDescRateLimit := messagepinFields[5].Descriptor()
	// messagepin.DefaultRateLimit holds the default value on creation for the rate_limit field.
	messagepin.DefaultRateLimit = messagepinDescRateLimit.Default.(schema.RateLimit)
	// messagepinDescID is the schema descriptor for id field.
	messagepinDescID := messagepinFields[0].Descriptor()
	// messagepin.DefaultID holds the default value on creation for the id field.
	messagepin.DefaultID = messagepinDescID.Default.(func() uuid.UUID)
	messageremindFields := schema.MessageRemind{}.Fields()
	_ = messageremindFields
	// messageremindDescContent is the schema descriptor for content field.
	messageremindDescContent := messageremindFields[4].Descriptor()
	// messageremind.ContentValidator is a validator for the "content" field. It is called by the builders before save.
	messageremind.ContentValidator = messageremindDescContent.Validators[0].(func(string) error)
	// messageremindDescName is the schema descriptor for name field.
	messageremindDescName := messageremindFields[5].Descriptor()
	// messageremind.NameValidator is a validator for the "name" field. It is called by the builders before save.
	messageremind.NameValidator = messageremindDescName.Validators[0].(func(string) error)
	// messageremindDescID is the schema descriptor for id field.
	messageremindDescID := messageremindFields[0].Descriptor()
	// messageremind.DefaultID holds the default value on creation for the id field.
	messageremind.DefaultID = messageremindDescID.Default.(func() uuid.UUID)
	rolepanelFields := schema.RolePanel{}.Fields()
	_ = rolepanelFields
	// rolepanelDescName is the schema descriptor for name field.
	rolepanelDescName := rolepanelFields[1].Descriptor()
	// rolepanel.NameValidator is a validator for the "name" field. It is called by the builders before save.
	rolepanel.NameValidator = rolepanelDescName.Validators[0].(func(string) error)
	// rolepanelDescID is the schema descriptor for id field.
	rolepanelDescID := rolepanelFields[0].Descriptor()
	// rolepanel.DefaultID holds the default value on creation for the id field.
	rolepanel.DefaultID = rolepanelDescID.Default.(func() uuid.UUID)
	rolepaneleditFields := schema.RolePanelEdit{}.Fields()
	_ = rolepaneleditFields
	// rolepaneleditDescModified is the schema descriptor for modified field.
	rolepaneleditDescModified := rolepaneleditFields[5].Descriptor()
	// rolepaneledit.DefaultModified holds the default value on creation for the modified field.
	rolepaneledit.DefaultModified = rolepaneleditDescModified.Default.(bool)
	// rolepaneleditDescName is the schema descriptor for name field.
	rolepaneleditDescName := rolepaneleditFields[6].Descriptor()
	// rolepaneledit.NameValidator is a validator for the "name" field. It is called by the builders before save.
	rolepaneledit.NameValidator = rolepaneleditDescName.Validators[0].(func(string) error)
	// rolepaneleditDescID is the schema descriptor for id field.
	rolepaneleditDescID := rolepaneleditFields[0].Descriptor()
	// rolepaneledit.DefaultID holds the default value on creation for the id field.
	rolepaneledit.DefaultID = rolepaneleditDescID.Default.(func() uuid.UUID)
	rolepanelplacedFields := schema.RolePanelPlaced{}.Fields()
	_ = rolepanelplacedFields
	// rolepanelplacedDescButtonType is the schema descriptor for button_type field.
	rolepanelplacedDescButtonType := rolepanelplacedFields[4].Descriptor()
	// rolepanelplaced.DefaultButtonType holds the default value on creation for the button_type field.
	rolepanelplaced.DefaultButtonType = discord.ButtonStyle(rolepanelplacedDescButtonType.Default.(int))
	// rolepanelplaced.ButtonTypeValidator is a validator for the "button_type" field. It is called by the builders before save.
	rolepanelplaced.ButtonTypeValidator = func() func(int) error {
		validators := rolepanelplacedDescButtonType.Validators
		fns := [...]func(int) error{
			validators[0].(func(int) error),
			validators[1].(func(int) error),
		}
		return func(button_type int) error {
			for _, fn := range fns {
				if err := fn(button_type); err != nil {
					return err
				}
			}
			return nil
		}
	}()
	// rolepanelplacedDescShowName is the schema descriptor for show_name field.
	rolepanelplacedDescShowName := rolepanelplacedFields[5].Descriptor()
	// rolepanelplaced.DefaultShowName holds the default value on creation for the show_name field.
	rolepanelplaced.DefaultShowName = rolepanelplacedDescShowName.Default.(bool)
	// rolepanelplacedDescFoldingSelectMenu is the schema descriptor for folding_select_menu field.
	rolepanelplacedDescFoldingSelectMenu := rolepanelplacedFields[6].Descriptor()
	// rolepanelplaced.DefaultFoldingSelectMenu holds the default value on creation for the folding_select_menu field.
	rolepanelplaced.DefaultFoldingSelectMenu = rolepanelplacedDescFoldingSelectMenu.Default.(bool)
	// rolepanelplacedDescHideNotice is the schema descriptor for hide_notice field.
	rolepanelplacedDescHideNotice := rolepanelplacedFields[7].Descriptor()
	// rolepanelplaced.DefaultHideNotice holds the default value on creation for the hide_notice field.
	rolepanelplaced.DefaultHideNotice = rolepanelplacedDescHideNotice.Default.(bool)
	// rolepanelplacedDescUseDisplayName is the schema descriptor for use_display_name field.
	rolepanelplacedDescUseDisplayName := rolepanelplacedFields[8].Descriptor()
	// rolepanelplaced.DefaultUseDisplayName holds the default value on creation for the use_display_name field.
	rolepanelplaced.DefaultUseDisplayName = rolepanelplacedDescUseDisplayName.Default.(bool)
	// rolepanelplacedDescCreatedAt is the schema descriptor for created_at field.
	rolepanelplacedDescCreatedAt := rolepanelplacedFields[9].Descriptor()
	// rolepanelplaced.DefaultCreatedAt holds the default value on creation for the created_at field.
	rolepanelplaced.DefaultCreatedAt = rolepanelplacedDescCreatedAt.Default.(func() time.Time)
	// rolepanelplacedDescUses is the schema descriptor for uses field.
	rolepanelplacedDescUses := rolepanelplacedFields[10].Descriptor()
	// rolepanelplaced.DefaultUses holds the default value on creation for the uses field.
	rolepanelplaced.DefaultUses = rolepanelplacedDescUses.Default.(int)
	// rolepanelplacedDescName is the schema descriptor for name field.
	rolepanelplacedDescName := rolepanelplacedFields[11].Descriptor()
	// rolepanelplaced.NameValidator is a validator for the "name" field. It is called by the builders before save.
	rolepanelplaced.NameValidator = rolepanelplacedDescName.Validators[0].(func(string) error)
	// rolepanelplacedDescID is the schema descriptor for id field.
	rolepanelplacedDescID := rolepanelplacedFields[0].Descriptor()
	// rolepanelplaced.DefaultID holds the default value on creation for the id field.
	rolepanelplaced.DefaultID = rolepanelplacedDescID.Default.(func() uuid.UUID)
	userFields := schema.User{}.Fields()
	_ = userFields
	// userDescName is the schema descriptor for name field.
	userDescName := userFields[1].Descriptor()
	// user.NameValidator is a validator for the "name" field. It is called by the builders before save.
	user.NameValidator = userDescName.Validators[0].(func(string) error)
	// userDescCreatedAt is the schema descriptor for created_at field.
	userDescCreatedAt := userFields[2].Descriptor()
	// user.DefaultCreatedAt holds the default value on creation for the created_at field.
	user.DefaultCreatedAt = userDescCreatedAt.Default.(func() time.Time)
	// userDescLocale is the schema descriptor for locale field.
	userDescLocale := userFields[3].Descriptor()
	// user.DefaultLocale holds the default value on creation for the locale field.
	user.DefaultLocale = discord.Locale(userDescLocale.Default.(string))
	// user.LocaleValidator is a validator for the "locale" field. It is called by the builders before save.
	user.LocaleValidator = userDescLocale.Validators[0].(func(string) error)
	// userDescXp is the schema descriptor for xp field.
	userDescXp := userFields[4].Descriptor()
	// user.DefaultXp holds the default value on creation for the xp field.
	user.DefaultXp = xppoint.XP(userDescXp.Default.(uint64))
	wordsuffixFields := schema.WordSuffix{}.Fields()
	_ = wordsuffixFields
	// wordsuffixDescSuffix is the schema descriptor for suffix field.
	wordsuffixDescSuffix := wordsuffixFields[1].Descriptor()
	// wordsuffix.SuffixValidator is a validator for the "suffix" field. It is called by the builders before save.
	wordsuffix.SuffixValidator = wordsuffixDescSuffix.Validators[0].(func(string) error)
	// wordsuffixDescID is the schema descriptor for id field.
	wordsuffixDescID := wordsuffixFields[0].Descriptor()
	// wordsuffix.DefaultID holds the default value on creation for the id field.
	wordsuffix.DefaultID = wordsuffixDescID.Default.(func() uuid.UUID)
}

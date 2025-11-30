package i18n

import (
	"fmt"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"gopkg.in/yaml.v3"
)

type ComponentType string

const (
	ComponentTypeText                  ComponentType = "text"
	ComponentTypeButton                ComponentType = "button"
	ComponentTypeStringSelectMenu      ComponentType = "string_select_menu"
	ComponentTypeUserSelectMenu        ComponentType = "user_select_menu"
	ComponentTypeRoleSelectMenu        ComponentType = "role_select_menu"
	ComponentTypeMentionableSelectMenu ComponentType = "mentionable_select_menu"
	ComponentTypeChannelSelectMenu     ComponentType = "channel_select_menu"
	ComponentTypeActionRow             ComponentType = "action_row"
	ComponentTypeSection               ComponentType = "section"
	ComponentTypeMediaGallery          ComponentType = "media_gallery"
	ComponentTypeFile                  ComponentType = "file"
	ComponentTypeSeparator             ComponentType = "separator"
	ComponentTypeContainer             ComponentType = "container"
	ComponentTypeThumbnail             ComponentType = "thumbnail"
	ComponentTypeTextInput             ComponentType = "text_input"
	ComponentTypeFileUpload            ComponentType = "file_upload"
	ComponentTypeLabel                 ComponentType = "label"
	ComponentTypeUnknown               ComponentType = "unknown"
)

// Component is an interface that represents a Discord component.
// [ActionRow]
// [Button]
// [StringSelectMenu]
// [UserSelectMenu]
// [RoleSelectMenu]
// [MentionableSelectMenu]
// [ChannelSelectMenu]
// [Section]
// [MediaGallery]
// [File]
// [Thumbnail]
// [Separator]
// [Container]
// [TextInput]
// [TextDisplay]
type Component interface {
	Type() ComponentType
	component(ctx MapContext) discord.Component
}

// LayoutComponent is an interface that represents a Discord layout component.
// [ActionRow]
// [Section]
// [MediaGallery]
// [Container]
// [TextDisplay]
// [File]
// [Separator]
// this package provides some interactive components as layout components, but they are not layout components in the Discord API sense.
// [Button]
// [StringSelectMenu]
// [UserSelectMenu]
// [RoleSelectMenu]
// [MentionableSelectMenu]
// [ChannelSelectMenu]
type LayoutComponent interface {
	Component
	layoutComponent(ctx MapContext) discord.LayoutComponent
}

// InteractiveComponent is an interface that represents a Discord interactive component.
// [Button]
// [StringSelectMenu]
// [UserSelectMenu]
// [RoleSelectMenu]
// [MentionableSelectMenu]
// [ChannelSelectMenu]
// [TextInput]
type InteractiveComponent interface {
	Component
	interactiveComponent(ctx MapContext) discord.InteractiveComponent
}

// SectionSubComponent is an interface that represents a Discord section sub-component.
// [TextDisplay]
type SectionSubComponent interface {
	Component
	sectionSubComponent(ctx MapContext) discord.SectionSubComponent
}

// SectionAccessoryComponent is an interface that represents a Discord section accessory component.
// [Button]
// [Thumbnail]
type SectionAccessoryComponent interface {
	Component
	sectionAccessoryComponent(ctx MapContext) discord.SectionAccessoryComponent
}

// ContainerSubComponent is an interface that represents a Discord container sub-component.
// [ActionRow]
// [Section]
// [TextDisplay]
// [MediaGallery]
// [File]
// [Separator]
// this package provides some interactive components as container sub-components, but they are not container sub-components in the Discord API sense.
// [Button]
// [StringSelectMenu]
// [UserSelectMenu]
// [RoleSelectMenu]
// [MentionableSelectMenu]
// [ChannelSelectMenu]
type ContainerSubComponent interface {
	Component
	containerSubComponent(ctx MapContext) discord.ContainerSubComponent
}

// LabelSubComponent is an interface that represents a Discord label sub-component.
// [StringSelectMenuComponent]
// [TextInputComponent]
// [UserSelectMenuComponent]
// [RoleSelectMenuComponent]
// [MentionableSelectMenuComponent]
// [ChannelSelectMenuComponent]
// [FileUploadComponent]
type LabelSubComponent interface {
	Component
	labelSubComponent(ctx MapContext) discord.LabelSubComponent
}

type UnmarshalComponent struct {
	Component
}

func (u *UnmarshalComponent) UnmarshalYAML(value *yaml.Node) error {
	// If the value is a string, we treat it as a TextDisplay component
	if value.Kind == yaml.ScalarNode {
		if value.Tag == "!!str" || value.Tag == "" {
			u.Component = TextDisplay(value.Value)
			return nil
		}
	}

	var cType struct {
		Type ComponentType `yaml:"type"`
	}

	if err := value.Decode(&cType); err != nil {
		return err
	}
	var component Component
	var err error
	switch cType.Type {
	case ComponentTypeText:
		var v TextDisplay
		err = value.Decode(&v)
		component = v
	case ComponentTypeButton:
		var v Button
		err = value.Decode(&v)
		component = v
	case ComponentTypeStringSelectMenu:
		var v StringSelectMenu
		err = value.Decode(&v)
		component = v
	case ComponentTypeUserSelectMenu:
		var v UserSelectMenu
		err = value.Decode(&v)
		component = v
	case ComponentTypeRoleSelectMenu:
		var v RoleSelectMenu
		err = value.Decode(&v)
		component = v
	case ComponentTypeMentionableSelectMenu:
		var v MentionableSelectMenu
		err = value.Decode(&v)
		component = v
	case ComponentTypeChannelSelectMenu:
		var v ChannelSelectMenu
		err = value.Decode(&v)
		component = v
	case ComponentTypeActionRow:
		var v ActionRow
		err = value.Decode(&v)
		component = v
	case ComponentTypeSection:
		var v Section
		err = value.Decode(&v)
		component = v
	case ComponentTypeMediaGallery:
		var v MediaGallery
		err = value.Decode(&v)
		component = v
	case ComponentTypeFile:
		var v File
		err = value.Decode(&v)
		component = v
	case ComponentTypeThumbnail:
		var v Thumbnail
		err = value.Decode(&v)
		component = v
	case ComponentTypeSeparator:
		var v Separator
		err = value.Decode(&v)
		component = v
	case ComponentTypeContainer:
		var v Container
		err = value.Decode(&v)
		component = v
	case ComponentTypeTextInput:
		var v TextInput
		err = value.Decode(&v)
		component = v
	case ComponentTypeFileUpload:
		var v FileUpload
		err = value.Decode(&v)
		component = v
	case ComponentTypeLabel:
		var v Label
		err = value.Decode(&v)
		component = v
	default:
		err = ErrUnknownComponentType.Format(cType.Type)
	}
	if err != nil {
		return err
	}
	u.Component = component
	return nil
}

// [discord.TextDisplayComponent]
type TextDisplay string

func (l *TextDisplay) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		if value.Tag == "!!str" || value.Tag == "" {
			*l = TextDisplay(value.Value)
			return nil
		}
	}
	var v struct {
		Content string `yaml:"content"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	*l = TextDisplay(v.Content)
	return nil
}

func (l TextDisplay) Type() ComponentType {
	return ComponentTypeText
}
func (l TextDisplay) textDisplay(ctx MapContext) discord.TextDisplayComponent {
	return discord.NewTextDisplay(ctx.ReplaceText(string(l)))
}
func (l TextDisplay) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return l.textDisplay(ctx)
}
func (l TextDisplay) sectionSubComponent(ctx MapContext) discord.SectionSubComponent {
	return l.textDisplay(ctx)
}
func (l TextDisplay) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return l.textDisplay(ctx)
}
func (l TextDisplay) component(ctx MapContext) discord.Component {
	return l.textDisplay(ctx)
}

// [discord.ButtonComponent]
type Button struct {
	Label string              `yaml:"label"`
	Style discord.ButtonStyle `yaml:"style"`
	ID    string              `yaml:"id"`
	Emoji *Emoji              `yaml:"emoji,omitempty"`
}

var buttonStyles = map[string]discord.ButtonStyle{
	"primary":   discord.ButtonStylePrimary,
	"secondary": discord.ButtonStyleSecondary,
	"success":   discord.ButtonStyleSuccess,
	"danger":    discord.ButtonStyleDanger,
	"link":      discord.ButtonStyleLink,
}

func (l *Button) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Label string `yaml:"label"`
		Style any    `yaml:"style"`
		ID    string `yaml:"id"`
		Emoji *Emoji `yaml:"emoji,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Label = v.Label
	switch v.Style.(type) {
	case int:
		l.Style = discord.ButtonStyle(v.Style.(int))
	case string:
		l.Style = buttonStyles[strings.ToLower(v.Style.(string))]
	default:
		return ErrInvalidButtonStyle.Format(v.Style)
	}
	l.ID = v.ID
	l.Emoji = v.Emoji
	return nil
}

func (l Button) Type() ComponentType {
	return ComponentTypeButton
}
func (l Button) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return discord.NewActionRow(l.button(ctx))
}
func (l Button) button(ctx MapContext) discord.ButtonComponent {
	button := discord.NewButton(l.Style, ctx.ReplaceText(l.Label), ctx.ReplaceCustomID(l.ID), ctx.GetURL(l.ID), 0)
	if l.Emoji != nil {
		button.Emoji = l.Emoji.Emoji()
	}
	button.Disabled = ctx.IsDisabled(l.ID)
	return button
}
func (l Button) interactiveComponent(ctx MapContext) discord.InteractiveComponent {
	return l.button(ctx)
}
func (l Button) sectionAccessoryComponent(ctx MapContext) discord.SectionAccessoryComponent {
	return l.button(ctx)
}
func (l Button) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return discord.NewActionRow(l.button(ctx))
}
func (l Button) component(ctx MapContext) discord.Component {
	return l.button(ctx)
}

// [discord.StringSelectMenuComponent]
type StringSelectMenu struct {
	ID          string                            `yaml:"id"`
	Placeholder string                            `yaml:"placeholder,omitempty"`
	Options     map[string]StringSelectMenuOption `yaml:"options"`
}

func (l *StringSelectMenu) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		ID          string                            `yaml:"id"`
		Placeholder string                            `yaml:"placeholder,omitempty"`
		Options     map[string]StringSelectMenuOption `yaml:"options"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.ID = v.ID
	l.Placeholder = v.Placeholder
	l.Options = v.Options
	return nil
}

func (l StringSelectMenu) Type() ComponentType {
	return ComponentTypeStringSelectMenu
}
func (l StringSelectMenu) stringSelectMenu(ctx MapContext) discord.StringSelectMenuComponent {
	options := make([]discord.StringSelectMenuOption, 0, len(l.Options))
	for _, option := range l.Options {
		options = append(options, option.option(ctx))
	}
	return discord.StringSelectMenuComponent{
		CustomID:    ctx.ReplaceCustomID(l.ID),
		Placeholder: ctx.ReplaceText(l.Placeholder),
		Options:     ctx.GetDefaultOptions(l.ID, options),
		MinValues:   ctx.GetMinValues(l.ID),
		MaxValues:   ctx.GetMaxValues(l.ID),
		Disabled:    ctx.IsDisabled(l.ID),
	}
}
func (l StringSelectMenu) interactiveComponent(ctx MapContext) discord.InteractiveComponent {
	return l.stringSelectMenu(ctx)
}
func (l StringSelectMenu) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return discord.NewActionRow(l.stringSelectMenu(ctx))
}
func (l StringSelectMenu) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return discord.NewActionRow(l.stringSelectMenu(ctx))
}
func (l StringSelectMenu) component(ctx MapContext) discord.Component {
	return l.stringSelectMenu(ctx)
}

// [discord.StringSelectMenuOption]
type StringSelectMenuOption struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	Emoji       *Emoji `yaml:"emoji,omitempty"`
	Value       string `yaml:"value"`
}

func (s *StringSelectMenuOption) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Label       string `yaml:"label"`
		Description string `yaml:"description,omitempty"`
		Emoji       *Emoji `yaml:"emoji,omitempty"`
		Value       string `yaml:"value"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	s.Label = v.Label
	s.Description = v.Description
	s.Emoji = v.Emoji
	s.Value = v.Value
	return nil
}

func (s StringSelectMenuOption) option(ctx MapContext) discord.StringSelectMenuOption {
	return discord.StringSelectMenuOption{
		Label:       ctx.ReplaceText(s.Label),
		Description: ctx.ReplaceText(s.Description),
		Value:       ctx.ReplaceCustomID(s.Value),
		Emoji:       s.Emoji.Emoji(),
	}
}

// [discord.RoleSelectMenuComponent]
type UserSelectMenu struct {
	Placeholder string `yaml:"placeholder"`
	ID          string `yaml:"id"`
}

func (l *UserSelectMenu) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Placeholder string `yaml:"placeholder"`
		ID          string `yaml:"id"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Placeholder = v.Placeholder
	l.ID = v.ID
	return nil
}

func (l UserSelectMenu) Type() ComponentType {
	return ComponentTypeUserSelectMenu
}
func (l UserSelectMenu) userSelectMenu(ctx MapContext) discord.UserSelectMenuComponent {
	// DefaultValues is not used for UserSelectMenu, so we pass an empty string
	return discord.UserSelectMenuComponent{
		CustomID:      ctx.ReplaceCustomID(l.ID),
		Placeholder:   ctx.ReplaceText(l.Placeholder),
		DefaultValues: ctx.GetDefaultValues(l.ID, discord.SelectMenuDefaultValueTypeUser),
		MinValues:     ctx.GetMinValues(l.ID),
		MaxValues:     ctx.GetMaxValues(l.ID),
		Disabled:      ctx.IsDisabled(l.ID),
	}
}
func (l UserSelectMenu) interactiveComponent(ctx MapContext) discord.InteractiveComponent {
	return l.userSelectMenu(ctx)
}
func (l UserSelectMenu) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return discord.NewActionRow(l.userSelectMenu(ctx))
}
func (l UserSelectMenu) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return discord.NewActionRow(l.userSelectMenu(ctx))
}
func (l UserSelectMenu) component(ctx MapContext) discord.Component {
	return l.userSelectMenu(ctx)
}

// [discord.RoleSelectMenuComponent]
type RoleSelectMenu struct {
	Placeholder string `yaml:"placeholder"`
	ID          string `yaml:"id"`
}

func (l *RoleSelectMenu) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Placeholder string `yaml:"placeholder"`
		ID          string `yaml:"id"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Placeholder = v.Placeholder
	l.ID = v.ID
	return nil
}

func (l RoleSelectMenu) Type() ComponentType {
	return ComponentTypeRoleSelectMenu
}

func (l RoleSelectMenu) roleSelectMenu(ctx MapContext) discord.RoleSelectMenuComponent {
	return discord.RoleSelectMenuComponent{
		CustomID:      ctx.ReplaceCustomID(l.ID),
		Placeholder:   ctx.ReplaceText(l.Placeholder),
		DefaultValues: ctx.GetDefaultValues(l.ID, discord.SelectMenuDefaultValueTypeRole),
		MinValues:     ctx.GetMinValues(l.ID),
		MaxValues:     ctx.GetMaxValues(l.ID),
		Disabled:      ctx.IsDisabled(l.ID),
	}
}
func (l RoleSelectMenu) interactiveComponent(ctx MapContext) discord.InteractiveComponent {
	return l.roleSelectMenu(ctx)
}
func (l RoleSelectMenu) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return discord.NewActionRow(l.roleSelectMenu(ctx))
}
func (l RoleSelectMenu) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return discord.NewActionRow(l.roleSelectMenu(ctx))
}
func (l RoleSelectMenu) component(ctx MapContext) discord.Component {
	return l.roleSelectMenu(ctx)
}

// [discord.MentionableSelectMenuComponent]
type MentionableSelectMenu struct {
	Placeholder string `yaml:"placeholder"`
	ID          string `yaml:"id"`
}

func (l *MentionableSelectMenu) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Placeholder string `yaml:"placeholder"`
		ID          string `yaml:"id"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Placeholder = v.Placeholder
	l.ID = v.ID
	return nil
}

func (l MentionableSelectMenu) Type() ComponentType {
	return ComponentTypeMentionableSelectMenu
}
func (l MentionableSelectMenu) mentionableSelectMenu(ctx MapContext) discord.MentionableSelectMenuComponent {
	return discord.MentionableSelectMenuComponent{
		CustomID:      ctx.ReplaceCustomID(l.ID),
		Placeholder:   ctx.ReplaceText(l.Placeholder),
		DefaultValues: ctx.GetDefaultValues(l.ID, ""),
		MinValues:     ctx.GetMinValues(l.ID),
		MaxValues:     ctx.GetMaxValues(l.ID),
		Disabled:      ctx.IsDisabled(l.ID),
	}
}
func (l MentionableSelectMenu) interactiveComponent(ctx MapContext) discord.InteractiveComponent {
	return l.mentionableSelectMenu(ctx)
}
func (l MentionableSelectMenu) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return discord.NewActionRow(l.mentionableSelectMenu(ctx))
}
func (l MentionableSelectMenu) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return discord.NewActionRow(l.mentionableSelectMenu(ctx))
}
func (l MentionableSelectMenu) component(ctx MapContext) discord.Component {
	return l.mentionableSelectMenu(ctx)
}

// [discord.ChannelSelectMenuComponent]
type ChannelSelectMenu struct {
	Placeholder  string                `yaml:"placeholder"`
	ID           string                `yaml:"id"`
	ChannelTypes []discord.ChannelType `yaml:"type,omitempty"`
}

func (l *ChannelSelectMenu) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Placeholder  string                `yaml:"placeholder"`
		ID           string                `yaml:"id"`
		ChannelTypes []discord.ChannelType `yaml:"type,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Placeholder = v.Placeholder
	l.ID = v.ID
	l.ChannelTypes = v.ChannelTypes
	return nil
}

func (l ChannelSelectMenu) Type() ComponentType {
	return ComponentTypeChannelSelectMenu
}
func (l ChannelSelectMenu) channelSelectMenu(ctx MapContext) discord.ChannelSelectMenuComponent {
	return discord.ChannelSelectMenuComponent{
		CustomID:      ctx.ReplaceCustomID(l.ID),
		Placeholder:   ctx.ReplaceText(l.Placeholder),
		DefaultValues: ctx.GetDefaultValues(l.ID, discord.SelectMenuDefaultValueTypeChannel),
		ChannelTypes:  l.ChannelTypes,
		MinValues:     ctx.GetMinValues(l.ID),
		MaxValues:     ctx.GetMaxValues(l.ID),
		Disabled:      ctx.IsDisabled(l.ID),
	}
}
func (l ChannelSelectMenu) interactiveComponent(ctx MapContext) discord.InteractiveComponent {
	return l.channelSelectMenu(ctx)
}
func (l ChannelSelectMenu) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return discord.NewActionRow(l.channelSelectMenu(ctx))
}
func (l ChannelSelectMenu) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return discord.NewActionRow(l.channelSelectMenu(ctx))
}
func (l ChannelSelectMenu) component(ctx MapContext) discord.Component {
	return l.channelSelectMenu(ctx)
}

// [discord.ActionRowComponent]
type ActionRow struct {
	Components []InteractiveComponent `yaml:"components"`
}

func (l *ActionRow) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Components []UnmarshalComponent `yaml:"components"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Components = make([]InteractiveComponent, len(v.Components))
	for i, c := range v.Components {
		var ok bool
		l.Components[i], ok = c.Component.(InteractiveComponent)
		if !ok {
			return fmt.Errorf("component %d is type %s not InteractiveComponent", i, c.Component.Type())
		}
	}
	return nil
}

func (l ActionRow) Type() ComponentType {
	return ComponentTypeActionRow
}
func (l ActionRow) actionRow(ctx MapContext) discord.ActionRowComponent {
	components := make([]discord.InteractiveComponent, len(l.Components))
	for i, c := range l.Components {
		components[i] = c.interactiveComponent(ctx)
	}
	return discord.NewActionRow(components...)
}
func (l ActionRow) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return l.actionRow(ctx)
}
func (l ActionRow) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return l.actionRow(ctx)
}
func (l ActionRow) component(ctx MapContext) discord.Component {
	return l.actionRow(ctx)
}

// [discord.SectionComponent]
type Section struct {
	Components []SectionSubComponent     `yaml:"components"`
	Accessory  SectionAccessoryComponent `yaml:"accessory"`
}

func (l *Section) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Components []UnmarshalComponent `yaml:"components"`
		Accessory  UnmarshalComponent   `yaml:"accessory,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Components = make([]SectionSubComponent, len(v.Components))
	for i, c := range v.Components {
		var ok bool
		l.Components[i], ok = c.Component.(SectionSubComponent)
		if !ok {
			return fmt.Errorf("component %d is type %s not SectionSubComponent", i, c.Component.Type())
		}
	}
	if v.Accessory.Component != nil {
		var ok bool
		l.Accessory, ok = v.Accessory.Component.(SectionAccessoryComponent)
		if !ok {
			return fmt.Errorf("accessory is type %s not SectionAccessoryComponent", v.Accessory.Component.Type())
		}
	} else {
		l.Accessory = nil
	}
	return nil
}

func (l Section) Type() ComponentType {
	return ComponentTypeSection
}
func (l Section) section(ctx MapContext) discord.SectionComponent {
	components := make([]discord.SectionSubComponent, len(l.Components))
	for i, c := range l.Components {
		components[i] = c.sectionSubComponent(ctx)
	}
	var accessory discord.SectionAccessoryComponent
	if l.Accessory != nil {
		accessory = l.Accessory.sectionAccessoryComponent(ctx)
	}
	if accessory == nil {
		accessory = discord.ButtonComponent{
			Style:    discord.ButtonStyleSecondary,
			Label:    "!!ERROR!!",
			CustomID: fmt.Sprint(time.Now().UnixNano()),
			Disabled: true,
		}
	}
	return discord.SectionComponent{
		Components: components,
		Accessory:  accessory,
	}
}
func (l Section) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return l.section(ctx)
}
func (l Section) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return l.section(ctx)
}
func (l Section) component(ctx MapContext) discord.Component {
	return l.section(ctx)
}

// [discord.ThumbnailComponent]
type Thumbnail struct {
	URL         string `yaml:"url,omitempty"`
	Description string `yaml:"description,omitempty"`
}

func (t *Thumbnail) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		URL         string `yaml:"url,omitempty"`
		Description string `yaml:"description,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	t.URL = v.URL
	t.Description = v.Description
	return nil
}

func (t Thumbnail) Type() ComponentType {
	return ComponentTypeThumbnail
}
func (t Thumbnail) sectionAccessoryComponent(ctx MapContext) discord.SectionAccessoryComponent {
	url := t.URL
	if strings.HasPrefix(t.URL, "{") && strings.HasSuffix(t.URL, "}") {
		url = ctx.GetURL(t.URL[1 : len(t.URL)-1])
	}
	return discord.ThumbnailComponent{
		Media: discord.UnfurledMediaItem{
			URL: url,
		},
		Description: ctx.ReplaceText(t.Description),
	}
}
func (t Thumbnail) component(ctx MapContext) discord.Component {
	return t.sectionAccessoryComponent(ctx)
}

// [discord.MediaGalleryComponent]
type MediaGallery struct {
	Media []MediaItem `yaml:"media"`
}

func (l *MediaGallery) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Media []MediaItem `yaml:"media"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Media = v.Media
	return nil
}

func (l MediaGallery) Type() ComponentType {
	return ComponentTypeMediaGallery
}
func (l MediaGallery) mediaGallery(ctx MapContext) discord.MediaGalleryComponent {
	media := make([]discord.MediaGalleryItem, len(l.Media))
	for i, m := range l.Media {
		media[i] = m.mediaItem(ctx)
	}
	return discord.MediaGalleryComponent{
		Items: media,
	}
}
func (l MediaGallery) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return l.mediaGallery(ctx)
}
func (l MediaGallery) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return l.mediaGallery(ctx)
}
func (l MediaGallery) component(ctx MapContext) discord.Component {
	return l.mediaGallery(ctx)
}

// [discord.MediaGalleryItem]
type MediaItem struct {
	URL         string `yaml:"url,omitempty"`
	Description string `yaml:"description,omitempty"`
}

func (m *MediaItem) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		URL         string `yaml:"url,omitempty"`
		Description string `yaml:"description,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	m.URL = v.URL
	m.Description = v.Description
	return nil
}

func (m MediaItem) mediaItem(ctx MapContext) discord.MediaGalleryItem {
	url := m.URL
	if strings.HasPrefix(m.URL, "{") && strings.HasSuffix(m.URL, "}") {
		url = ctx.GetURL(m.URL[1 : len(m.URL)-1])
	}
	return discord.MediaGalleryItem{
		Media: discord.UnfurledMediaItem{
			URL: url,
		},
		Description: ctx.ReplaceText(m.Description),
	}
}

// [discord.FileComponent]
type File struct {
	URL string `yaml:"url,omitempty"`
}

func (l *File) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		URL string `yaml:"url,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.URL = v.URL
	return nil
}

func (l File) Type() ComponentType {
	return ComponentTypeFile
}
func (l File) file(ctx MapContext) discord.FileComponent {
	// If the URL is a placeholder, replace it with the actual URL from the context
	url := l.URL
	if strings.HasPrefix(l.URL, "{") && strings.HasSuffix(l.URL, "}") {
		url = ctx.GetURL(l.URL[1 : len(l.URL)-1])
	}
	return discord.FileComponent{
		File: discord.UnfurledMediaItem{
			URL: url,
		},
	}
}
func (l File) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return l.file(ctx)
}
func (l File) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return l.file(ctx)
}
func (l File) component(ctx MapContext) discord.Component {
	return l.file(ctx)
}

// [discord.SeparatorComponent]
type Separator struct {
	Divider *bool                        `yaml:"divider,omitempty"` // If true, the separator will be a divider, otherwise it will be a separator
	Size    discord.SeparatorSpacingSize `yaml:"height,omitempty"`  // Size of the separator, default is 1
}

func (l *Separator) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Divider *bool                        `yaml:"divider,omitempty"`
		Size    discord.SeparatorSpacingSize `yaml:"height,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Divider = v.Divider
	l.Size = v.Size
	return nil
}

func (l Separator) Type() ComponentType {
	return ComponentTypeSeparator
}
func (l Separator) separator(_ MapContext) discord.SeparatorComponent {
	return discord.SeparatorComponent{
		Divider: l.Divider,
		Spacing: l.Size,
	}
}
func (l Separator) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return l.separator(ctx)
}
func (l Separator) containerSubComponent(ctx MapContext) discord.ContainerSubComponent {
	return l.separator(ctx)
}
func (l Separator) component(ctx MapContext) discord.Component {
	return l.separator(ctx)
}

type Container struct {
	Components []ContainerSubComponent `yaml:"components"`
}

func (l *Container) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Components []UnmarshalComponent `yaml:"components"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Components = make([]ContainerSubComponent, len(v.Components))
	for i, c := range v.Components {
		var ok bool
		l.Components[i], ok = c.Component.(ContainerSubComponent)
		if !ok {
			return fmt.Errorf("component %d is type %s not ContainerSubComponent", i, c.Component.Type())
		}
	}
	return nil
}

func (l Container) Type() ComponentType {
	return ComponentTypeContainer
}
func (l Container) container(ctx MapContext) discord.ContainerComponent {
	components := make([]discord.ContainerSubComponent, len(l.Components))
	for i, c := range l.Components {
		components[i] = c.containerSubComponent(ctx)
	}
	return discord.NewContainer(components...)
}
func (l Container) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return l.container(ctx)
}
func (l Container) component(ctx MapContext) discord.Component {
	return l.container(ctx)
}

func deserializeLayoutComponent(ctx MapContext, layout []LayoutComponent) ([]discord.LayoutComponent, error) {
	var components []discord.LayoutComponent

	for _, l := range layout {
		components = append(components, l.layoutComponent(ctx))
	}

	return components, nil
}

// [discord.TextInputComponent]
type TextInput struct {
	CustomID    string                 `yaml:"id"`
	Style       discord.TextInputStyle `yaml:"style"`
	MinLength   *int                   `yaml:"min_length,omitempty"`
	MaxLength   int                    `yaml:"max_length,omitempty"`
	Required    bool                   `yaml:"required"`
	Placeholder string                 `yaml:"placeholder,omitempty"`
	Value       string                 `yaml:"value,omitempty"`
}

var textInputStyles = map[string]discord.TextInputStyle{
	"short":     discord.TextInputStyleShort,
	"paragraph": discord.TextInputStyleParagraph,
}

func (l *TextInput) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		CustomID    string `yaml:"id"`
		Style       string `yaml:"style"`
		MinLength   *int   `yaml:"min_length,omitempty"`
		MaxLength   int    `yaml:"max_length,omitempty"`
		Required    bool   `yaml:"required"`
		Placeholder string `yaml:"placeholder,omitempty"`
		Value       string `yaml:"value,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.CustomID = v.CustomID
	l.Style = textInputStyles[v.Style]
	l.MinLength = v.MinLength
	l.MaxLength = v.MaxLength
	l.Required = v.Required
	l.Placeholder = v.Placeholder
	l.Value = v.Value
	return nil
}

func (l TextInput) Type() ComponentType {
	return ComponentTypeTextInput
}
func (l TextInput) textInput(ctx MapContext) discord.TextInputComponent {
	return discord.TextInputComponent{
		CustomID:    ctx.ReplaceCustomID(l.CustomID),
		Style:       l.Style,
		MinLength:   l.MinLength,
		MaxLength:   l.MaxLength,
		Required:    l.Required,
		Placeholder: ctx.ReplaceText(l.Placeholder),
		Value:       ctx.ReplaceText(l.Value),
	}
}
func (l TextInput) component(ctx MapContext) discord.Component {
	return l.textInput(ctx)
}
func (l TextInput) interactiveComponent(ctx MapContext) discord.InteractiveComponent {
	return l.textInput(ctx)
}
func (l TextInput) labelSubComponent(ctx MapContext) discord.LabelSubComponent {
	return l.textInput(ctx)
}

// [discord.FileUploadComponent]
type FileUpload struct {
	CustomID  string `yaml:"id"`
	MinValues *int   `yaml:"min_values,omitempty"`
	// MaxValues is the maximum number of files that can be uploaded. (default: 1, min: 1, max: 10)
	MaxValues int `yaml:"max_values,omitempty"`
	// Required specifies whether the file upload is required. (default: false)
	Required bool `yaml:"required"`
}

func (l *FileUpload) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		CustomID  string `yaml:"id"`
		Required  bool   `yaml:"required"`
		MinValues *int   `yaml:"min_values,omitempty"`
		MaxValues int    `yaml:"max_values,omitempty"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.CustomID = v.CustomID
	l.Required = v.Required
	l.MinValues = v.MinValues
	l.MaxValues = v.MaxValues
	return nil
}

func (l FileUpload) Type() ComponentType {
	return ComponentTypeFileUpload
}
func (l FileUpload) fileUpload(ctx MapContext) discord.FileUploadComponent {
	return discord.FileUploadComponent{
		CustomID:  ctx.ReplaceCustomID(l.CustomID),
		MinValues: ctx.GetMinValues(l.CustomID),
		MaxValues: ctx.GetMaxValues(l.CustomID),
		Required:  l.Required,
	}
}
func (l FileUpload) component(ctx MapContext) discord.Component {
	return l.fileUpload(ctx)
}
func (l FileUpload) interactiveComponent(ctx MapContext) discord.InteractiveComponent {
	return l.fileUpload(ctx)
}
func (l FileUpload) labelSubComponent(ctx MapContext) discord.LabelSubComponent {
	return l.fileUpload(ctx)
}

// [discord.LabelComponent]
type Label struct {
	Label       string            `yaml:"label"`
	Description string            `yaml:"description,omitempty"`
	Component   LabelSubComponent `yaml:"component"`
}

func (l *Label) UnmarshalYAML(value *yaml.Node) error {
	var v struct {
		Label       string             `yaml:"label"`
		Description string             `yaml:"description,omitempty"`
		Component   UnmarshalComponent `yaml:"component"`
	}
	if err := value.Decode(&v); err != nil {
		return err
	}
	l.Label = v.Label
	l.Description = v.Description
	var ok bool
	l.Component, ok = v.Component.Component.(LabelSubComponent)
	if !ok {
		return fmt.Errorf("component is type %s not LabelSubComponent", v.Component.Component.Type())
	}
	return nil
}

func (l Label) Type() ComponentType {
	return ComponentTypeLabel
}
func (l Label) label(ctx MapContext) discord.LabelComponent {
	return discord.LabelComponent{
		Label:       ctx.ReplaceText(l.Label),
		Description: ctx.ReplaceText(l.Description),
		Component:   l.Component.labelSubComponent(ctx),
	}
}
func (l Label) component(ctx MapContext) discord.Component {
	return l.label(ctx)
}
func (l Label) layoutComponent(ctx MapContext) discord.LayoutComponent {
	return l.label(ctx)
}

/*
some.key.text:
    -   type: text
		label: "Hello, World!"
	-   "this text will be typed as text"
    -   type: button
		label: "Click me"
		style: primary
		id: "click_me"
	-   type: string_select
		label: "Select an option"
		options:
			option_1:
				label: "Option 1"
				emoji: "👍"
			option_2:
				label: "Option 2"
				emoji: "👎"
			option_3:
				label: "Option 3"
				emoji: 1234567890
*/

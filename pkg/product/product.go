package product

const (
	ProductName                        string = "gobot"
	prefix                             string = ProductName + "_"
	CommandPanelRole                   string = prefix + "panel_role"
	CommandPanelRoleCreate             string = prefix + "panel_role_create"
	CommandPanelAdd                    string = prefix + "panel_role_add"
	CommandPanelMinecraft              string = prefix + "panel_minecraft"
	CommandPanelMinecraftAddServerName string = prefix + "panel_minecraft_add_servername"
	CommandPanelMinecraftAddAddress    string = prefix + "panel_minecraft_add_address"
	CommandPanelMinecraftAddPort       string = prefix + "panel_minecraft_add_port"
	CommandPanelMinecraftAddModal      string = prefix + "panel_minecraft_add_modal"
)

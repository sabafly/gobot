/*
	Copyright (C) 2022  ikafly144

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package product

const (
	ProductName                               string = "gobot"
	prefix                                    string = ProductName + "_"
	CommandPanelRole                          string = prefix + "panel_role"
	CommandPanelRoleCreate                    string = prefix + "panel_role_create"
	CommandPanelAdd                           string = prefix + "panel_role_add"
	CommandPanelMinecraft                     string = prefix + "panel_minecraft"
	CommandPanelMinecraftAddServerName        string = prefix + "panel_minecraft_add_servername"
	CommandPanelMinecraftAddAddress           string = prefix + "panel_minecraft_add_address"
	CommandPanelMinecraftAddPort              string = prefix + "panel_minecraft_add_port"
	CommandPanelMinecraftAddModal             string = prefix + "panel_minecraft_add_modal"
	CommandPanelVote                          string = prefix + "panel_vote"
	CommandPanelVoteCreatePreview             string = prefix + "panel_vote_create_preview"
	CommandPanelVoteCreateAdd                 string = prefix + "panel_vote_create_add"
	CommandPanelVoteCreateAddModal            string = prefix + "panel_vote_create_add_modal"
	CommandPanelVoteCreateAddModalTitle       string = prefix + "panel_vote_create_add_modal_title"
	CommandPanelVoteCreateAddModalDescription string = prefix + "panel_vote_create_add_modal_description"
	CommandPanelVoteCreateDo                  string = prefix + "panel_vote_create_do"
)

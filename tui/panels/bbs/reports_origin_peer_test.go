/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package bbs

import (
	"strings"
	"testing"
	"time"

	"github.com/sabafly/gobot/internal/admin"
	"github.com/sabafly/gobot/tui/ui"
)

func TestReportsState_DisplaysOriginPeerWhenPresent(t *testing.T) {
	theme := ui.DefaultTheme()
	st := newReportsState()
	st.detailVP.SetWidth(80)
	st.items = []admin.BBSReportDTO{
		{
			ID:             "rep-1",
			Scope:          "global",
			OriginPeer:     "peer-beta",
			GuildID:        "12345",
			BoardID:        "board-1",
			ThreadID:       "thread-1",
			PostID:         "post-1",
			ReporterUserID: "999",
			Reason:         "test reason",
			Status:         "open",
			CreatedAt:      time.Now(),
		},
	}
	st.rebuildDetail(theme)
	if !strings.Contains(st.detailVP.View(), "Origin Peer") || !strings.Contains(st.detailVP.View(), "peer-beta") {
		t.Fatalf("expected Origin Peer 'peer-beta' in report detail view, got:\n%s", st.detailVP.View())
	}

	st.items[0].OriginPeer = ""
	st.rebuildDetail(theme)
	if strings.Contains(st.detailVP.View(), "Origin Peer") {
		t.Fatalf("expected Origin Peer to be omitted for native report, got:\n%s", st.detailVP.View())
	}
}

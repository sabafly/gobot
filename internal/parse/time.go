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
 *
 */

package parse

import (
	"time"

	"github.com/markusmobius/go-dateparser"
	"github.com/sabafly/gobot/internal/errors"
	"github.com/tj/go-naturaldate"
)

func TimeFuture(str string) (time.Time, error) {
	if d, err := time.ParseDuration(str); err == nil {
		return time.Now().Local().Add(d), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05 MST", str+" JST"); err == nil {
		return t.Local(), nil
	}
	if t, err := naturaldate.Parse(str, time.Now().Local(), naturaldate.WithDirection(naturaldate.Future)); err == nil {
		return t.Local(), nil
	}
	if t, err := dateparser.Parse(&dateparser.Configuration{
		CurrentTime: time.Now().Local(),
	}, str); err == nil {
		return t.Time.Local(), nil
	}
	return time.Time{}, errors.New("invalid format")
}

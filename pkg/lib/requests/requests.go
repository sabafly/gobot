/*
	Copyright (C) 2022-2023  ikafly144

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
package requests

import (
	"encoding/json"
	"io"
	"net/http"
)

func Unmarshal(r *http.Request, v any) error {
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(buf, &v); err != nil {
		return err
	}
	return nil
}

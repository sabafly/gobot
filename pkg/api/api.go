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
package api

import (
	"io"
	"net/http"

	"github.com/ikafly144/gobot/pkg/env"
)

var IP string = *env.APIServer

func GetApi(URI string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("GET", "http://"+IP+URI, body)
	if err != nil {
		return &http.Response{}, err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func ReqAPI(method string, URI string, body io.Reader) (res *http.Response, err error) {
	req, err := http.NewRequest(method, "http://"+IP+URI, body)
	if err != nil {
		return nil, err
	}
	client := http.Client{}
	res, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

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
package gobot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ikafly144/gobot/pkg/lib/logger"
)

type RequestConfig struct {
	Request        *http.Request
	MaxRestRetries int
	Client         http.Client
}

func newRequestConfig(a *Api, req *http.Request) *RequestConfig {
	return &RequestConfig{
		MaxRestRetries: a.MaxRestRetries,
		Client:         a.Client,
		Request:        req,
	}
}

// 内部REST APIに(GET, POST)リクエストを送信する
// Sequenceはシーケンス回数を指定する。
// もし502エラーで失敗したら成功するかシーケンスがapi.MaxRestRetries以上になるまでシーケンス回数+1回繰り返します
func (a *Api) Request(method, urlStr string, data any) (response []byte, err error) {
	var body []byte
	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	return a.request(method, urlStr, "application/json", body, 0)
}

// リクエストを作成します
func (a *Api) request(method, urlStr, contentType string, b []byte, sequence int) (response []byte, err error) {
	logger.Debug("[内部] API REQUEST %6s :: %s\n", method, urlStr)
	logger.Debug("[内部] API REQUEST PAYLOAD :: [%s]\n", string(b))

	req, err := http.NewRequest(method, urlStr, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	if b != nil {
		req.Header.Set("Content-Type", contentType)
	}

	req.Header.Set("UserAgent", a.UserAgent)

	cfg := newRequestConfig(a, req)

	for k, v := range req.Header {
		logger.Debug("[内部] API REQUEST   HEADER :: [%s] = %+v\n", k, v)
	}

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Debug("error closing resp body")
		}
	}()

	response, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logger.Debug("API RESPONSE  STATUS :: %s\n", resp.Status)
	for k, v := range resp.Header {
		logger.Debug("API RESPONSE  HEADER :: [%s] = %+v\n", k, v)
	}
	logger.Debug("API RESPONSE    BODY :: [%s]\n\n\n", response)

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusCreated:
	case http.StatusNoContent:
	case http.StatusBadGateway:
		// 可能ならリクエストをやり直す
		if sequence < cfg.MaxRestRetries {

			logger.Info("%s 失敗 (%s) 再試行します...", urlStr, resp.Status)
			response, err = a.request(method, urlStr, contentType, b, sequence+1)
		} else {
			return nil, fmt.Errorf("too many requests")
		}
	default:
		return nil, fmt.Errorf("unknown status: %s", resp.Status)
	}

	return response, nil
}

// ------------------------------------------------------
// websocket接続の関数
// ------------------------------------------------------

// 接続ゲートウェイを取得
func (a *Api) Gateway() (gateway string, err error) {
	response, err := a.Request("GET", EndpointGateway, nil)
	if err != nil {
		return "", err
	}

	temp := struct {
		URL string `json:"url"`
	}{}

	if err = json.Unmarshal(response, &temp); err != nil {
		return "", err
	}

	gateway = temp.URL

	gateway = strings.TrimSuffix(gateway, "/")

	return gateway, nil
}

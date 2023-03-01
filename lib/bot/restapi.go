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
package botlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sabafly/gobot/lib/logging"
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
	logging.Debug("[内部] API REQUEST %6s :: %s\n", method, urlStr)
	// logging.Debug("[内部] API REQUEST PAYLOAD :: [%s]\n", string(b))

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
		logging.Debug("[内部] API REQUEST   HEADER :: [%s] = %+v\n", k, v)
	}

	resp, err := cfg.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logging.Debug("error closing resp body")
		}
	}()

	response, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logging.Debug("API RESPONSE  STATUS :: %s\n", resp.Status)
	for k, v := range resp.Header {
		logging.Debug("API RESPONSE  HEADER :: [%s] = %+v\n", k, v)
	}
	logging.Debug("API RESPONSE    BODY :: [%s]\n\n\n", response)

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusCreated:
	case http.StatusNoContent:
	case http.StatusBadGateway:
		// 可能ならリクエストをやり直す
		if sequence < cfg.MaxRestRetries {

			logging.Info("%s 失敗 (%s) 再試行します...", urlStr, resp.Status)
			response, err = a.request(method, urlStr, contentType, b, sequence+1)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("too many requests")
		}
	default:
		return nil, fmt.Errorf("unknown status: %s", resp.Status)
	}

	return response, nil
}

// ------------------------------------------------------
// websocket接続関連
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

// ------------------------------------------------------
// API呼び出し
// ------------------------------------------------------

// ギルド作成呼び出し
func (a *Api) guildCreateCall(guildID string) {
	g := struct{ ID string }{ID: guildID}
	if _, err := a.Request("POST", EndpointGuild, g); err != nil {
		logging.Warning("リクエストに失敗 %s", err)
	}
}

// ギルド削除呼び出し
func (a *Api) guildDeleteCall(g *discordgo.GuildDelete) {
	if _, err := a.Request("DELETE", EndpointGuild, g); err != nil {
		logging.Warning("リクエストに失敗 %s", err)
	}
}

// メッセージ送信呼び出し
func (a *Api) messageCreateCall(m *discordgo.MessageCreate) {
	if _, err := a.Request("POST", EndpointMessage, m); err != nil {
		logging.Warning("リクエストに失敗 %s", err)
	}
}

// ----------------------------------------------------------------
// 統計
// ----------------------------------------------------------------

func (a *Api) StaticsUserMessage(guildID, userID string) (logs []MessageLog, err error) {
	uri := EndpointStaticsUserMessage

	queryPrams := url.Values{}
	if guildID != "" {
		queryPrams.Set("guild", guildID)
	}
	if userID != "" {
		queryPrams.Set("user", userID)
	}

	if len(queryPrams) > 0 {
		uri += "?" + queryPrams.Encode()
	}

	response, err := a.Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	logs = []MessageLog{}
	err = json.Unmarshal(response, &logs)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

// ----------------------------------------------------------------
// Feature関連
// ----------------------------------------------------------------

func (a *Api) FeatureEnable(guildID, featureID, id string) (err error) {
	Feature := GuildFeature{
		GuildID:   guildID,
		FeatureID: featureID,
		TargetID:  id,
	}

	_, err = a.Request("POST", EndpointGuildFeature, Feature)
	if err != nil {
		return err
	}

	return nil
}

func (a *Api) FeatureDisable(guildID, featureID, id string) (err error) {
	Feature := GuildFeature{
		GuildID:   guildID,
		FeatureID: featureID,
		TargetID:  id,
	}

	_, err = a.Request("DELETE", EndpointGuildFeature, Feature)
	if err != nil {
		return err
	}

	return nil
}

func (a *Api) FeatureEnabled(guildID, featureID, id string) (enabled bool, err error) {
	Feature := GuildFeature{
		GuildID:   guildID,
		FeatureID: featureID,
		TargetID:  id,
	}

	isEnabled := struct{ Enabled bool }{}
	response, err := a.Request("GET", EndpointGuildFeature, Feature)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(response, &isEnabled)
	if err != nil {
		return false, err
	}

	return isEnabled.Enabled, nil
}

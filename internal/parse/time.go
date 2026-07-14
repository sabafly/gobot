package parse

import (
	"time"

	"github.com/markusmobius/go-dateparser"
	"github.com/tj/go-naturaldate"

	"github.com/sabafly/gobot/internal/errors"
)

// TimeFuture は文字列を解析して未来の時刻を表すtime.Timeを返します。
// 以下のフォーマットを順に試みます：
// 1. 継続時間（例："1h30m"）- 現在時刻に加算します
// 2. ISO形式（例："2006-01-02 15:04:05 JST"）- JSTタイムゾーン指定
// 3. 自然言語日付（例："tomorrow at 3pm"）- 未来方向に解析
// 4. その他の日付形式 - dateparserライブラリを使用
//
// パラメータ：
//   - str: 解析する時間文字列
//
// 戻り値：
//   - time.Time: 解析された時刻（ローカルタイムゾーンに変換）
//   - error: 解析エラー。すべての解析方法が失敗した場合は "invalid format" エラーを返します
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

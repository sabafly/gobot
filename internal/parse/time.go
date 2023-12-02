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
	return time.Now().Local(), errors.New("invalid format")
}

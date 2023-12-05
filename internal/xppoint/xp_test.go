package xppoint_test

import (
	"testing"

	"github.com/sabafly/gobot/internal/xppoint"
	"golang.org/x/exp/slog"
)

func TestXPSum(t *testing.T) {
	for i := 0; i < 1000; i++ {
		pt := xppoint.TotalPoint(int64(i))
		xp := xppoint.XP(pt)
		if xp.Level() != int64(i) {
			slog.Warn("failed 1", "xp", xp, "i", i, "level", xp.Level())
			t.Fail()
		}
	}
}

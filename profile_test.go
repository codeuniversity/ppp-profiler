package profiler_test

import (
	"testing"

	"github.com/codeuniversity/ppp-mhist"

	"github.com/codeuniversity/ppp-profiler"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_Profile(t *testing.T) {
	Convey("evaluates messages correctly", t, func() {
		script := `
			var sum = get("sum", 0)
			sum += message.value
			set("sum", sum)
		`
		profile := profiler.NewProfile(script, mhist.FilterDefinition{})
		profile.Eval(&mhist.Message{Value: 2})
		So(profile.Value(), ShouldContainKey, "sum")
		So(profile.Value()["sum"].(float64), ShouldEqual, 2)
		profile.Eval(&mhist.Message{Value: 3})
		So(profile.Value()["sum"].(float64), ShouldEqual, 5)
	})
}

func Benchmark_Profile(b *testing.B) {
	script := `
	var sum = get("sum", 0)
	sum += message.value
	set("sum", sum)
`
	profile := profiler.NewProfile(script, mhist.FilterDefinition{})
	for i := 0; i < b.N; i++ {
		profile.Eval(&mhist.Message{Value: 2})
	}
}

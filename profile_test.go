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
			title("the sum is " + sum)
			description("some description")
			action("You should do something")
		`
		profile := profiler.NewProfile(profiler.ProfileDefinition{EvalScript: script})
		profile.Eval(&mhist.Message{Value: 2})
		So(profile.Value().Data, ShouldContainKey, "title")
		So(profile.Value().Data["title"].(string), ShouldEqual, "the sum is 2")
		So(profile.Value().Data["description"].(string), ShouldEqual, "some description")
		So(profile.Value().Data["action"].(string), ShouldEqual, "You should do something")
		profile.Eval(&mhist.Message{Value: 3})
		So(profile.Value().Data["title"].(string), ShouldEqual, "the sum is 5")
	})
}

func Benchmark_Profile(b *testing.B) {
	script := `
	var sum = get("sum", 0)
	sum += message.value
	set("sum", sum)
`
	profile := profiler.NewProfile(profiler.ProfileDefinition{EvalScript: script})
	for i := 0; i < b.N; i++ {
		profile.Eval(&mhist.Message{Value: 2})
	}
}

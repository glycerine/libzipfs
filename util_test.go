package libzipfs

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestTrimTrailingSlashesWorks(t *testing.T) {

	cv.Convey("TrimTrailingSlashes(`hello///`) should return `hello` and similar correct trims", t, func() {
		cv.So(TrimTrailingSlashes(`hello///`), cv.ShouldEqual, `hello`)
		cv.So(TrimTrailingSlashes(`hello`), cv.ShouldEqual, `hello`)
		cv.So(TrimTrailingSlashes(``), cv.ShouldEqual, ``)
		cv.So(TrimTrailingSlashes(`abc`), cv.ShouldEqual, `abc`)
		cv.So(TrimTrailingSlashes(`/a/b/c/d/`), cv.ShouldEqual, `/a/b/c/d`)
		cv.So(TrimTrailingSlashes(`/a/b/c/d`), cv.ShouldEqual, `/a/b/c/d`)
	})
}

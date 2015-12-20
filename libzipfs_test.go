package libzipfs

import (
	"fmt"
	"io/ioutil"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func TestWeCanMountInTheTmpDir(t *testing.T) {

	cv.Convey("we should be able to mount a zipfile image in the tmp dir", t, func() {
		dir := "/tmp" // "" => use system tmp dir
		mountPoint, err := ioutil.TempDir(dir, "libzipfs")
		//fmt.Printf("\n\n mountPoint = '%s'\n", mountPoint)
		cv.So(err, cv.ShouldEqual, nil)

		zipFile := "testfiles/hi.zip"
		z := NewFuseZipFs(zipFile, mountPoint)

		err = z.Start()
		if err != nil {
			panic(fmt.Sprintf("error during starting FuseZipFs "+
				"for file '%s' at mount point %s: '%s'", zipFile, mountPoint, err))
		}

		err = z.Stop()
		if err != nil {
			panic(fmt.Sprintf("error: could not z.Stop() FuseZipFs for file '%s' at %s: '%s'", zipFile, mountPoint, err))
		}

	})

}

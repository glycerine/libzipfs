package libzipfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func Test001WeCanMountInTheTmpDir(t *testing.T) {

	cv.Convey("we should be able to mount a zipfile image in the tmp dir", t, func() {
		dir := "/tmp" // => /tmp easier to debug/shorter to type.
		// dir := "" // => use system tmp dir
		mountPoint, err := ioutil.TempDir(dir, "libzipfs")
		VPrintf("\n\n mountPoint = '%s'\n", mountPoint)
		cv.So(err, cv.ShouldEqual, nil)

		zipFile := "testfiles/hi.zip"
		z := NewFuseZipFs(zipFile, mountPoint)

		err = z.Start()
		if err != nil {
			panic(fmt.Sprintf("error during starting FuseZipFs "+
				"for file '%s' at mount point %s: '%s'", zipFile, mountPoint, err))
		}

		VPrintf("\n\n z.Start() succeeded, with mountPoint = '%s'\n", mountPoint)
		expectedFile := path.Join(mountPoint, "dirA", "dirB", "hello")
		expectedFileContent := []byte("salutations\n")

		fmt.Printf("\n   we should be able to read back a file from the mounted filesystem without errors.\n")
		ef, err := os.Open(expectedFile)
		cv.So(err, cv.ShouldBeNil)
		cv.So(ef, cv.ShouldNotBeNil)
		err = ef.Close()
		cv.So(err, cv.ShouldBeNil)

		by, err := ioutil.ReadFile(expectedFile)
		cv.So(err, cv.ShouldBeNil)
		cv.So(len(expectedFileContent), cv.ShouldEqual, len(by))
		diff, err := compareByteSlices(expectedFileContent, by, len(expectedFileContent))
		cv.So(err, cv.ShouldBeNil)
		cv.So(diff, cv.ShouldEqual, -1)

		err = z.Stop()
		if err != nil {
			panic(fmt.Sprintf("error: could not z.Stop() FuseZipFs for file '%s' at %s: '%s'", zipFile, mountPoint, err))
		}

		VPrintf("\n\n z.Stop() succeeded, with mountPoint = '%s'\n", mountPoint)
	})

}

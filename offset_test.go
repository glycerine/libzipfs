package libzipfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func Test004WeCanMountAnOffsetZipFile(t *testing.T) {

	cv.Convey("we should be able to mount a zipfile image from the second half of a file, i.e. given an offset into the file, mount from the middle of the file should work", t, func() {
		//dir := "/tmp" // => /tmp easier to debug/shorter to type.
		dir := "" // => use system tmp dir
		mountPoint, err := ioutil.TempDir(dir, "libzipfs")
		VPrintf("\n\n mountPoint = '%s'\n", mountPoint)
		cv.So(err, cv.ShouldEqual, nil)

		// padded8hi has 8 bytes of pre-padding, and no footer.
		comboFile := "testfiles/padded8hi"

		byteOffsetToZipFileStart := int64(8)
		z := NewFuseZipFs(comboFile, mountPoint, byteOffsetToZipFileStart, 478, 0)

		err = z.Start()
		if err != nil {
			panic(fmt.Sprintf("error during starting FuseZipFs "+
				"for file '%s' (at offset %d) at mount point %s: '%s'",
				comboFile, byteOffsetToZipFileStart, mountPoint, err))
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
			panic(fmt.Sprintf("error: could not z.Stop() FuseZipFs for file '%s' at %s: '%s'", comboFile, mountPoint, err))
		}

		VPrintf("\n\n z.Stop() succeeded, with mountPoint = '%s'\n", mountPoint)
	})

}

func Test004bWeCanMountARegularZipFile(t *testing.T) {

	cv.Convey("exact same test as 004 but with a regular zipfile (no padding before it) for diagnostics/comparison: testfiles/hi.zip", t, func() {
		//dir := "/tmp" // => /tmp easier to debug/shorter to type.
		dir := "" // => use system tmp dir
		mountPoint, err := ioutil.TempDir(dir, "libzipfs")
		VPrintf("\n\n mountPoint = '%s'\n", mountPoint)
		cv.So(err, cv.ShouldEqual, nil)

		// padded8hi has 8 bytes of pre-padding, and no footer.
		comboFile := "testfiles/hi.zip"

		byteOffsetToZipFileStart := int64(0)
		z := NewFuseZipFs(comboFile, mountPoint, byteOffsetToZipFileStart, 478, 0)

		err = z.Start()
		if err != nil {
			panic(fmt.Sprintf("error during starting FuseZipFs "+
				"for file '%s' (at offset %d) at mount point %s: '%s'",
				comboFile, byteOffsetToZipFileStart, mountPoint, err))
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
			panic(fmt.Sprintf("error: could not z.Stop() FuseZipFs for file '%s' at %s: '%s'", comboFile, mountPoint, err))
		}

		VPrintf("\n\n z.Stop() succeeded, with mountPoint = '%s'\n", mountPoint)
	})

}

func Test006WeCanMountAnOffsetComboFile(t *testing.T) {

	cv.Convey("we should be able to mount an offset zipfile from the second half of a combo file, i.e. given an offset into the file, mount from the middle of the file should work", t, func() {
		//dir := "/tmp" // => /tmp easier to debug/shorter to type.
		dir := "" // => use system tmp dir
		mountPoint, err := ioutil.TempDir(dir, "libzipfs")
		VPrintf("\n\n mountPoint = '%s'\n", mountPoint)
		cv.So(err, cv.ShouldEqual, nil)

		comboFile := "testfiles/expectedCombined"

		_, foot, comb, err := ReadFooter(comboFile)
		panicOn(err)
		defer comb.Close()
		byteOffsetToZipFileStart := foot.ExecutableLengthBytes

		z := NewFuseZipFs(comboFile, mountPoint, byteOffsetToZipFileStart, foot.ZipfileLengthBytes, LIBZIPFS_FOOTER_LEN)

		err = z.Start()
		if err != nil {
			panic(fmt.Sprintf("error during starting FuseZipFs "+
				"for file '%s' (at offset %d) at mount point %s: '%s'",
				comboFile, byteOffsetToZipFileStart, mountPoint, err))
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
			panic(fmt.Sprintf("error: could not z.Stop() FuseZipFs for file '%s' at %s: '%s'", comboFile, mountPoint, err))
		}

		VPrintf("\n\n z.Stop() succeeded, with mountPoint = '%s'\n", mountPoint)
	})

}

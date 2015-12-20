package libzipfs

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func Test003SplitterDetectsCorruptFooters(t *testing.T) {

	cv.Convey("corrupt or missing footers should be detected and cause an early exit", t, func() {
		testOutputPath, err := ioutil.TempFile("", "libzipfs.test.")
		panicOn(err)
		defer os.Remove(testOutputPath.Name())

		var cfg CombinerConfig
		expectedOutPath := "testfiles/expectedCombined"
		cfg.OutputPath = expectedOutPath
		cfg.ExecutablePath = "testfiles/tester"
		cfg.ZipfilePath = "testfiles/hi.zip"

		var foot Footer
		err = foot.FillHashes(&cfg)
		panicOn(err)
		footBuf := bytes.NewBuffer(foot.ToBytes())
		VPrintf("footBuf = '%x'\n", footBuf)

		// now split it back apart and check it
		splitCfg := cfg

		testSplitToExePath, err := ioutil.TempFile("", "libzipfs.test.")
		panicOn(err)
		defer testSplitToExePath.Close()
		defer os.Remove(testSplitToExePath.Name())

		testSplitToZipPath, err := ioutil.TempFile("", "libzipfs.test.")
		panicOn(err)
		defer testSplitToZipPath.Close()
		defer os.Remove(testSplitToZipPath.Name())

		splitCfg.ExecutablePath = testSplitToExePath.Name()
		splitCfg.ZipfilePath = testSplitToZipPath.Name()
		splitCfg.Split = true

		VPrintf("splitCfg = %#v\n", splitCfg)
		recoveredFoot, err := DoSplitOutExeAndZip(&splitCfg)
		cv.Convey("uncorrupt recoveredFoot should not raise an error, part 1", func() {
			cv.So(err, cv.ShouldBeNil)
			cv.So(recoveredFoot, cv.ShouldResemble, &foot)
		})

		footerStartOffset := recoveredFoot.ZipfileLengthBytes + recoveredFoot.ExecutableLengthBytes
		uncorruptBytes := recoveredFoot.ToBytes()
		_, err = ReifyFooterAndDoInexpensiveChecks(uncorruptBytes, &splitCfg, footerStartOffset)
		cv.Convey("uncorrupt recoveredFoot should not raise an error, part 2", func() {
			cv.So(err, cv.ShouldBeNil)
		})
		cv.Convey("corrupted recoveredFoot should raise an error, at any byte position", func() {
			corruptBytes := make([]byte, len(uncorruptBytes))
			copy(corruptBytes, uncorruptBytes)
			for i := range corruptBytes {
				// corrupt up
				corruptBytes[i]++
				_, err = ReifyFooterAndDoInexpensiveChecks(corruptBytes, &splitCfg, footerStartOffset)
				cv.So(err, cv.ShouldNotBeNil)
				corruptBytes[i] = uncorruptBytes[i]
				// or corrupt down
				corruptBytes[i]--
				_, err = ReifyFooterAndDoInexpensiveChecks(corruptBytes, &splitCfg, footerStartOffset)
				cv.So(err, cv.ShouldNotBeNil)
				corruptBytes[i] = uncorruptBytes[i]
			}
		})
	})
}

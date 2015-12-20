package libzipfs

import "testing"

func Test003SplitterInverseOfCombine(t *testing.T) {

	/* expected sizes:
	   -rw-r--r--   1 jaten  staff      478 Dec 19 17:27 hi.zip
	   -rwxr-xr-x   1 jaten  staff  2315808 Dec 19 22:17 tester
	*/
	/*
		cv.Convey("out footer should report the proper sizes for the input exe and .zip files", t, func() {
			testOutputPath, err := ioutil.TempFile("", "libzipfs.test.")
			panicOn(err)
			defer os.Remove(testOutputPath.Name())

			var cfg CombinerConfig
			cfg.OutputPath = testOutputPath.Name()
			cfg.ExecutablePath = "testfiles/tester"
			cfg.ZipfilePath = "testfiles/hi.zip"

			var foot Footer
			err = foot.FillHashes(&cfg)
			panicOn(err)
			footBuf := bytes.NewBuffer(foot.ToBytes())
			fmt.Printf("footBuf = '%x'\n", footBuf)

			cv.So(foot.ExecutableLengthBytes, cv.ShouldEqual, 2315808)
			cv.So(foot.ZipfileLengthBytes, cv.ShouldEqual, 478)
			cv.So(foot.FooterLengthBytes, cv.ShouldEqual, LIBZIPFS_FOOTER_LEN)

			cv.So(len(footBuf.Bytes()), cv.ShouldEqual, LIBZIPFS_FOOTER_LEN)

			fmt.Printf("exe checksum = '%x'\n", foot.ExecutableBlake2Checksum)
			fmt.Printf("zip checksum = '%x'\n", foot.ZipfileBlake2Checksum)
			fmt.Printf("foot checksum = '%x'\n", foot.FooterBlake2Checksum)

			cv.So(foot.MagicFooterNumber1[:len(MAGIC1)], cv.ShouldResemble, MAGIC1)
			cv.So(foot.MagicFooterNumber2[:len(MAGIC2)], cv.ShouldResemble, MAGIC2)

			cv.So(fmt.Sprintf("%x", foot.ExecutableBlake2Checksum), cv.ShouldResemble, `61af446f097d3b6c80a910dc295c1aef98f760a61ba3d324d98f134193a79d86ee7db4c46ca33a55879bc561638d0eaed774124d73d2776b21d8b697b98cc04a`)
			cv.So(fmt.Sprintf("%x", foot.ZipfileBlake2Checksum), cv.ShouldResemble, `13dad78f512d559c9661e23fe77040f6b08134ab7a29f90ac94c4280454e0973dc95ea034586621392dc8d02b8166326ffa812de9dbc9e1b471f977d8907d719`)
			cv.So(fmt.Sprintf("%x", foot.FooterBlake2Checksum), cv.ShouldResemble, `a24fb6f047d66d431e166abc8d008755d3cdfe2b07f4c06b256912151feb114c7cea606b35726e1ae2d2c133b50a7360fe2fce7ca950086f97aa58479e057a22`)
		})
	*/
}

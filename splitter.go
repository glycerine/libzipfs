/*
combiner appends a zip file to an executable and further appends a footer
in the last 256 bytes that describes the combination. libzipfs will look
for this footer and use it to determine where the internalized zipfile
filesystem starts.
*/
package libzipfs

import (
	"fmt"
	"io"
	"os"
)

func DoSplitOutExeAndZip(cfg *CombinerConfig) (*Footer, error) {

	// read last 256 bytes of combined file and extract the footer
	// cfg.OutputPath is our input now.
	combi, err := os.Stat(cfg.OutputPath)
	panicOn(err)
	fmt.Printf("combi = '%#v'", combi)

	comb, err := os.Open(cfg.OutputPath)
	panicOn(err)

	footerStartOffset, err := comb.Seek(-LIBZIPFS_FOOTER_LEN, 2)
	panicOn(err)
	fmt.Printf("footerStartOffset = %d\n", footerStartOffset)

	by := make([]byte, LIBZIPFS_FOOTER_LEN)
	n, err := comb.Read(by)
	if err != io.EOF {
		panicOn(err)
	}
	if n != LIBZIPFS_FOOTER_LEN {
		panic(fmt.Errorf("%d == n != LIBZIPFS_FOOTER_LEN == %d", n, LIBZIPFS_FOOTER_LEN))
	}

	var foot Footer
	foot.FromBytes(by[:])

	// check the checksum
	chk := foot.GetFooterChecksum()
	for i := 0; i < 64; i++ {
		if chk[i] != foot.FooterBlake2Checksum[i] {
			return &foot, fmt.Errorf("DoSplitOutexeAndZip() error: reified footer from file '%s' does not have the expected checksum, file corrupt or not a combined file?  at i=%d, disk position footerStartOffset=%d, computed footer checksum='%x', versus read-from-disk footer checksum = '%x'", cfg.OutputPath, i, footerStartOffset, chk, foot.FooterBlake2Checksum)
		}
	}

	// create the split out exe and zip files
	exeFd, err := os.Create(cfg.ExecutablePath)
	panicOn(err)
	defer exeFd.Close()

	exeStartOffset, err := comb.Seek(0, 0)
	panicOn(err)
	if exeStartOffset != 0 {
		panic(fmt.Errorf("exeStartOffset was %d but should be 0", exeStartOffset))
	}

	_, err = io.CopyN(exeFd, comb, foot.ExecutableLengthBytes)
	panicOn(err)
	exeFd.Close()

	zipFd, err := os.Create(cfg.ZipfilePath)
	panicOn(err)
	defer zipFd.Close()

	_, err = io.CopyN(zipFd, comb, foot.ZipfileLengthBytes)
	panicOn(err)
	zipFd.Close()

	err = foot.VerifyExeZipChecksums(cfg)

	return &foot, err
}

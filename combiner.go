/*
combiner appends a zip file to an executable and further appends a footer
in the last 256 bytes that describes the combination. libzipfs will look
for this footer and use it to determine where the internalized zipfile
filesystem starts.
*/
package libzipfs

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
)

var MAGIC1 = []byte("\nLibZipFs00\n")
var MAGIC2 = []byte("\nLibZipFsEnd\n")

const LIBZIPFS_FOOTER_LEN = 256
const BLAKE2_HASH_LEN = 64
const MAGIC_NUM_LEN = 16

type FooterArray [LIBZIPFS_FOOTER_LEN]byte

type Footer struct {
	Reserved1          int64
	MagicFooterNumber1 [MAGIC_NUM_LEN]byte

	ExecutableLengthBytes int64
	ZipfileLengthBytes    int64
	FooterLengthBytes     int64

	ExecutableBlake2Checksum [BLAKE2_HASH_LEN]byte
	ZipfileBlake2Checksum    [BLAKE2_HASH_LEN]byte
	FooterBlake2Checksum     [BLAKE2_HASH_LEN]byte // has itself set to zero when taking the hash.

	MagicFooterNumber2 [MAGIC_NUM_LEN]byte
}

type CombinerConfig struct {
	ExecutablePath string
	ZipfilePath    string
	OutputPath     string
	Split          bool
}

// call DefineFlags before myflags.Parse()
func (c *CombinerConfig) DefineFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.ExecutablePath, "exe", "", "path to the executable file")
	fs.StringVar(&c.ZipfilePath, "zip", "", "path to the zipfile to embed file")
	fs.StringVar(&c.OutputPath, "o", "", "path to the combined output file to be written (or split if -split given)")
	fs.BoolVar(&c.Split, "split", false, "split the output file back apart (instead of combine which is the default)")
}

// call c.ValidateConfig() after myflags.Parse()
func (c *CombinerConfig) ValidateConfig() error {
	if c.ExecutablePath == "" {
		return fmt.Errorf("-exe flag required and missing")
	}
	if c.ZipfilePath == "" {
		return fmt.Errorf("-zip flag required and missing")
	}
	if c.OutputPath == "" {
		return fmt.Errorf("-o file required and missing")
	}

	if c.Split {

		if FileExists(c.ExecutablePath) {
			return fmt.Errorf("-exe path '%s' found but should not exist yet", c.ExecutablePath)
		}

		if FileExists(c.ZipfilePath) {
			return fmt.Errorf("-zip path '%s' found but should not exist yet", c.ZipfilePath)
		}

		if !FileExists(c.OutputPath) {
			return fmt.Errorf("-o path '%s' not found for splitting", c.OutputPath)
		}

	} else {

		if !FileExists(c.ExecutablePath) {
			return fmt.Errorf("-exe path '%s' not found", c.ExecutablePath)
		}

		if !FileExists(c.ZipfilePath) {
			return fmt.Errorf("-zip path '%s' not found", c.ZipfilePath)
		}

		if FileExists(c.OutputPath) {
			return fmt.Errorf("-o path '%s' already exists but should not", c.OutputPath)
		}
	}

	return nil
}

func DoCombineExeAndZip(cfg *CombinerConfig) error {

	xi, err := os.Stat(cfg.ExecutablePath)
	panicOn(err)
	VPrintf("xi = '%#v'", xi)

	zi, err := os.Stat(cfg.ZipfilePath)
	panicOn(err)
	VPrintf("zi = '%#v'", zi)

	// create the footer metadata
	var foot Footer
	err = foot.FillHashes(cfg)
	panicOn(err)
	footBuf := bytes.NewBuffer(foot.ToBytes())

	// sanity check against the stat info
	if xi.Size() != foot.ExecutableLengthBytes {
		panic(fmt.Errorf("%d == xi.Size() != foot.ExecutableLengthBytes == %d", xi.Size(), foot.ExecutableLengthBytes))
	}
	if zi.Size() != foot.ZipfileLengthBytes {
		panic(fmt.Errorf("%d == zi.Size() != foot.ZipfileLengthBytes == %d", zi.Size(), foot.ZipfileLengthBytes))
	}

	// create the output file, o
	o, err := os.Create(cfg.OutputPath)
	panicOn(err)
	defer o.Close()

	// write to the output file from exe, zip, then footer:

	// open exe
	exeFd, err := os.Open(cfg.ExecutablePath)
	panicOn(err)
	defer exeFd.Close()

	// open zip
	zipFd, err := os.Open(cfg.ZipfilePath)
	panicOn(err)
	defer zipFd.Close()

	// copy exe to o
	exeSz, err := io.Copy(o, exeFd)
	panicOn(err)
	if exeSz != foot.ExecutableLengthBytes {
		panic("wrong exeSz!")
	}

	// copy zip to o
	zipSz, err := io.Copy(o, zipFd)
	panicOn(err)
	if zipSz != foot.ZipfileLengthBytes {
		panic("wrong zipSz!")
	}

	// copy footer to o
	footSz, err := io.Copy(o, footBuf)
	panicOn(err)
	if footSz != foot.FooterLengthBytes {
		panic("wrong footSz!")
	}

	o.Close()
	var executable = os.FileMode(0755)
	err = os.Chmod(cfg.OutputPath, executable)
	panicOn(err)

	return nil
}

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal error: '%s'\n", err)
		os.Exit(1)
	}
}

func (foot *Footer) VerifyExeZipChecksums(cfg *CombinerConfig) (err error) {

	hash, _, err := Blake2HashFile(cfg.ExecutablePath)
	if err != nil {
		return err
	}
	_, err = compareByteSlices(foot.ExecutableBlake2Checksum[:], hash, BLAKE2_HASH_LEN)
	if err != nil {
		return fmt.Errorf("executable blake2 checksum mismatch: '%s'", err)
	}

	hash, _, err = Blake2HashFile(cfg.ZipfilePath)
	if err != nil {
		return err
	}
	_, err = compareByteSlices(foot.ZipfileBlake2Checksum[:], hash, BLAKE2_HASH_LEN)
	if err != nil {
		return fmt.Errorf("zipfile blake2 checksum mismatch: '%s'", err)
	}
	return nil
}

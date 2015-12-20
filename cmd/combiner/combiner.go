/*
combiner appends a zip file to an executable and further appends a footer
in the last 256 bytes that describes the combination. libzipfs will look
for this footer and use it to determine where the internalized zipfile
filesystem starts.
*/
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

var progName string = path.Base(os.Args[0])

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
	fs.StringVar(&c.OutputPath, "o", "", "path to the output file to be written")
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

// demonstrate the sequence of calls to DefineFlags() and ValidateConfig()
func main() {

	myflags := flag.NewFlagSet(progName, flag.ExitOnError)
	cfg := &CombinerConfig{}
	cfg.DefineFlags(myflags)

	err := myflags.Parse(os.Args[1:])
	err = cfg.ValidateConfig()
	if err != nil {
		log.Fatalf("%s command line flag error: '%s'", progName, err)
	}

	xi, err := os.Stat(cfg.ExecutablePath)
	exitOn(err)
	fmt.Printf("xi = '%#v'", xi)

	// create the footer metadata
	var foot Footer
	err = foot.FillHashes(cfg)
	exitOn(err)
	footBuf := bytes.NewBuffer(foot.ToBytes())

	// create the output file, o
	o, err := os.Create(cfg.OutputPath)
	exitOn(err)
	defer o.Close()

	// write to the output file from exe, zip, then footer:

	// open exe
	exeFd, err := os.Open(cfg.ExecutablePath)
	exitOn(err)
	defer exeFd.Close()

	// open zip
	zipFd, err := os.Open(cfg.ZipfilePath)
	exitOn(err)
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

}

var MAGIC1 = []byte("\nLibZipFs00\n")
var MAGIC2 = []byte("\nLibZipFsEnd\n")

type Footer struct {
	Reserved1          int64
	MagicFooterNumber1 [16]byte

	ExecutableLengthBytes int64
	ZipfileLengthBytes    int64
	FooterLengthBytes     int64

	ExecutableBlake2Checksum [64]byte
	ZipfileBlake2Checksum    [64]byte
	FooterBlake2Checksum     [64]byte // has itself set to zero when taking the hash.

	MagicFooterNumber2 [16]byte
}

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func exitOn(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

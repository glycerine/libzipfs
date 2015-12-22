package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"

	"github.com/glycerine/libzipfs"
)

var progName string = path.Base(os.Args[0])

type MntzipConfig struct {
	ZipfilePath string
	MountPath   string
}

// call DefineFlags before myflags.Parse()
func (c *MntzipConfig) DefineFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.ZipfilePath, "zip", "", "path to the Zip file (or combo exe+Zip+footer file) to mount")
	fs.StringVar(&c.MountPath, "mnt", "", "directory to fuse-mount the Zip file on")
}

// call c.ValidateConfig() after myflags.Parse()
func (c *MntzipConfig) ValidateConfig() error {
	if c.ZipfilePath == "" {
		return fmt.Errorf("-zip flag required and missing")
	}
	if c.MountPath == "" {
		return fmt.Errorf("-mnt file required and missing")
	}

	if !libzipfs.FileExists(c.ZipfilePath) {
		return fmt.Errorf("-zip path '%s' not found.", c.ZipfilePath)
	}

	if !libzipfs.DirExists(c.MountPath) {
		return fmt.Errorf("-mnt mount path '%s' not found.", c.MountPath)
	}

	return nil
}

func main() {

	// grab ctrl-c
	ctrl_C_chan := make(chan os.Signal, 1)
	signal.Notify(ctrl_C_chan, os.Interrupt)

	// process command line args
	myflags := flag.NewFlagSet(progName, flag.ExitOnError)
	cfg := &MntzipConfig{}
	cfg.DefineFlags(myflags)

	err := myflags.Parse(os.Args[1:])
	err = cfg.ValidateConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mountzip: mount a regular Zip file or a libzipfs combo "+
			"(exe+Zip+footer) file's Zip content at the requested mount point. Combo "+
			"files are automatically detected.\n")
		myflags.PrintDefaults()
		log.Fatalf("%s command line flag error: '%s'", progName, err)
	}

	byteOffsetToZipFileStart := int64(0)
	bytesAvail := int64(0)
	footerBytes := int64(0)

	// detect if this is a combo file
	_, foot, comb, err := libzipfs.ReadFooter(cfg.ZipfilePath)
	if err != nil {
		// assume it is a regular zip file, not a combo file.
	} else {
		comb.Close()
		byteOffsetToZipFileStart = foot.ExecutableLengthBytes
		bytesAvail = foot.ZipfileLengthBytes
		footerBytes = foot.FooterLengthBytes
	}

	z := libzipfs.NewFuseZipFs(cfg.ZipfilePath,
		cfg.MountPath, byteOffsetToZipFileStart, bytesAvail, footerBytes)

	err = z.Start()
	if err != nil {
		log.Fatalf("%s error calling z.Start() to start serving fuse requests: '%s'", progName, err)
	}

	fmt.Printf("\nZip file '%s' mounted at directory '%s'. [press ctrl-c to exit and unmount]\n",
		cfg.ZipfilePath, cfg.MountPath)

	select {
	case <-ctrl_C_chan:
	case <-z.Done:
		// can happen if someone force unmounts the mount from under us.
	}

	err = z.Stop() // stop serving files and unmount at end
	if err != nil {
		log.Fatalf("%s error while shutting down: '%s'", progName, err)
	}
}

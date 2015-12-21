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
	fs.StringVar(&c.ZipfilePath, "zip", "", "path to the Zip file to mount")
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
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// process command line args
	myflags := flag.NewFlagSet(progName, flag.ExitOnError)
	cfg := &MntzipConfig{}
	cfg.DefineFlags(myflags)

	err := myflags.Parse(os.Args[1:])
	err = cfg.ValidateConfig()
	if err != nil {
		myflags.PrintDefaults()
		log.Fatalf("%s command line flag error: '%s'", progName, err)
	}

	byteOffsetToZipFileStart := int64(0)
	bytesAvail := int64(0)
	footerBytes := int64(0)

	z := libzipfs.NewFuseZipFs(cfg.ZipfilePath, cfg.MountPath, byteOffsetToZipFileStart, bytesAvail, footerBytes)

	err = z.Start()
	if err != nil {
		log.Fatalf("%s error calling z.Start() to start serving fuse requests: '%s'", progName, err)
	}
	defer z.Stop() // stop serving files and unmount

	fmt.Printf("\nZip file '%s' mounted at directory '%s'. [press ctrl-c to exit and unmount]\n",
		cfg.ZipfilePath, cfg.MountPath)

	<-signalChan
	// the defer z.Stop() takes care of the unmount
}

/*
combiner appends a zip file to an executable and further appends a footer
in the last 256 bytes that describes the combination. libzipfs will look
for this footer and use it to determine where the internalized zipfile
filesystem starts.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	lzf "github.com/glycerine/libzipfs"
)

var progName string = path.Base(os.Args[0])

func main() {

	myflags := flag.NewFlagSet(progName, flag.ExitOnError)
	cfg := &lzf.CombinerConfig{}
	cfg.DefineFlags(myflags)

	err := myflags.Parse(os.Args[1:])
	err = cfg.ValidateConfig()
	if err != nil {
		log.Fatalf("%s command line flag error: '%s'", progName, err)
	}

	if cfg.Split {
		err = lzf.DoSplitOutExeAndZip(cfg)
	} else {
		err = lzf.DoCombineExeAndZip(cfg)
	}
	panicOn(err)
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

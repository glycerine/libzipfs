package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/glycerine/libzipfs"
)

// build instructions:
//
// cd libzipfs && make
// cd testfiles; go build -o api-demo api.go
// libzipfs-combiner -exe api-demo -zip hi.zip -o api-demo-combo
// ./api-demo-combo

func main() {
	z, mountpoint, err := libzipfs.MountComboZip()
	if err != nil {
		panic(err)
	}
	defer z.Stop() // if you want to stop serving files

	// access the files from `my.media.zip` at mountpoint

	// since we may get EINTR and have to retry, we loop over ReadFile()
	var by []byte
	for {
		by, err = ioutil.ReadFile(path.Join(mountpoint, "dirA", "dirB", "hello"))
		if err == nil {
			break
		}
		switch e := err.(type) {
		case *os.PathError:
			if strings.HasSuffix(e.Error(), "interrupted system call") {
				continue // EINTR, must simply retry.
			}
			panic(fmt.Errorf("got unknown os.PathError, e = '%#v'. e.Error()='%#v'\n", e, e.Error()))
		default:
			fmt.Printf("unknown err was '%#v' / '%s'\n", err, err.Error())
			panic(err)
		}
		break
	}

	by = bytes.TrimRight(by, "\n")
	fmt.Printf("we should see our file dirA/dirB/hello from inside hi.zip, containing 'salutations'.... '%s'\n", string(by))

	if string(by) != "salutations" {
		panic("problem detected")
	}
	fmt.Printf("Excellent: all looks good.\n")
}

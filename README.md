libzipfs
===========

~~~
-------------------       ---------------
| go executable   |       |  zip file   |
-------------------       ---------------
        \                        /
         --> libzipfs-combiner <-
                      |
                      v
----------------------------------------------------
| go executable   |  zip file   |  256-byte footer |
----------------------------------------------------
~~~


Libzipfs lets you ship a filesystem of media resources inside your
golang application.  This is done by attaching a zipfile containing
the directories of your choice to the end of your application, and
following it with a short footer to allow the go executable to
locate the files and serve them via a fuse mount point. The client
will typically be either be Go or C embedded in the go executable,
but can be another application alltogether. For the later use,
see for example the [libzipfs/cmd/mountzip/mountzip.go example source code.](https://github.com/glycerine/libzipfs/blob/master/cmd/mountzip/mountzip.go)

## Use cases

1. You have a bunch of images/scripts/files to be served from a webserving application,
   and you want to bundle those resources alongside your Go webapp. libzipfs lets
   you easily create a single executable that contains all your resources in one
   place.

1. If you are using CGO to call C code, and that C code expects to be able to
   read files from somewhere on the filesystem, you can package up all those
   files, ship them with the executable, and they can be read from the
   fuse mountpoint -- where the C code can find and use them. For example,
   my https://github.com/glycerine/rmq project embeds R inside a Go binary,
   and libzipfs allows R libraries to be easily shipped all together in a
   single binary.

## status

Excellent. Works well and is very useful. I only use it on OSX and Linux. On OSX
you need to have [OSX Fuse](https://osxfuse.github.io/) installed first.  On Linux you'll need to either `sudo yum install fuse` or `sudo apt-get install fuse-utils` to obtain the `/bin/fusermount` utility.

## installation

~~~
$ go get -t -u -v github.com/glycerine/libzipfs
$ cd $GOPATH/src/github.com/glycerine/libzipfs && make
$ ## the libzipfs-combiner and mountzip utilities are now in your $GOPATH/bin
$ ## be sure that $GOPATH/bin is added to your PATH env variable
~~~

## origins

This library is dervied from Tommi Virtanen's work https://github.com/bazil/zipfs,
which is fantastic and provides a fuse-mounted read-only filesystem from a zipfile.
The zipfs library and https://github.com/bazil/fuse are doing the heavy lifting 
behind the scenes.

The project was inspired by https://github.com/bazil/zipfs and https://github.com/shurcooL/vfsgen

In particular, vfsgen is a similar approach, but I needed the ability to serve files to legacy code that
expects to read from a file system.

## libzipfs then goes a step beyond what zipfs provides

We then add the ability to assemble a "combined" file from an executable
and a zipfile. The structure of the combined file looks like this:

~~~
----------------------------------------------------
| executable      |  zip file   |  256-byte footer |
----------------------------------------------------
^                                                  ^
byte 0                                           byte N
~~~

The embedded zip file can then be made available via a fuse mountpoint.
The Go executable will contain Go code to accomplish this. The 256-bite
footer at the end of the file describes the location of the
embedded zip file. The combined file is still an executable,
and can be run directly.

### creating a combined executable and zipfile

the `libzipfs-combiner` utility does this for you.

For example: assuming that `my.go.binary` and `hi.zip` already exist,
and you wish to create a new combo executable called `my.go.binary.combo`,
you would do:

~~~
$ libzipfs-combiner --help
libzipfs-combiner --help
Usage of libzipfs-combiner:
  -exe string
    	path to the executable file
  -o string
    	path to the combined output file to be written (or split if -split given)
  -split
    	split the output file back apart (instead of combine which is the default)
  -zip string
    	path to the zipfile to embed file

$ libzipfs-combiner --exe my.go.binary -o my.go.binary.combo -zip hi.zip
~~~

### api/code code inside your `my.go.binary.combo` binary:

type `make demo` and see testfiles/api.go for a full demo:

Our demo zip file `testfiles/hi.zip` is a simple zip file with one file `hello` that resides inside two nested directories:

~~~
$ unzip -Z -z testfiles/hi.zip
Archive:  testfiles/hi.zip   478 bytes   3 files
drwxr-xr-x  3.0 unx        0 bx stor 19-Dec-15 17:27 dirA/
drwxr-xr-x  3.0 unx        0 bx stor 19-Dec-15 17:27 dirA/dirB/
-rw-r--r--  3.0 unx       12 tx stor 19-Dec-15 17:27 dirA/dirB/hello
3 files, 12 bytes uncompressed, 12 bytes compressed:  0.0%
~~~

the go code:

~~~
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"

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

	// access the files from `hi.zip` at mountpoint

	by, err := ioutil.ReadFile(path.Join(mountpoint, "dirA", "dirB", "hello"))
	if err != nil {
		panic(err)
	}
	by = bytes.TrimRight(by, "\n")
	fmt.Printf("we should see our file dirA/dirB/hello from inside hi.zip, containing 'salutations'.... '%s'\n", string(by))

	if string(by) != "salutations" {
		panic("problem detected")
	}
	fmt.Printf("Excellent: all looks good.\n")
}
~~~

## example number 2: mountzip

The `mountzip` utility (see the source code in `libzipfs/cmd/mountzip/mountzip.go`) mounts a zip file of your choice on a directory of your choice.

~~~
$ cd $GOPATH/src/github.com/glycerine/libzipfs
$ make # installs the mountzip utility into $GOPATH/bin
$ mountzip -help
Usage of mountzip:
  -mnt string
    	directory to fuse-mount the zipfile on
  -zip string
    	path to the zipfile to mount
$
$ mkdir /tmp/hi
$ mountzip -zip testfiles/hi.zip -mnt /tmp/hi
zipfile 'hi.zip' mounted at directory '/tmp/hi'. [press ctrl-c to exit and unmount]

~~~

license
-------

[MIT license](http://opensource.org/licenses/mit-license.php). See enclosed LICENSE file.
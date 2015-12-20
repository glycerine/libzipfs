libzipfs
===========

## Use cases

1. You have a bunch of images/scripts/files to be served from a webserving application,
   and you want to bundle those resources alongside your Go webapp. libzipfs lets
   you easily create a single executable that contains all you resources in one
   place.

1. If you are using CGO to call C code, and that C code expects to be able to
   read files from somewhere on the filesystem, you can package up all those
   files and still mount them in a fuse-based filesystem mountpoint where
   the C code can find and use them. For example, by https://github.com/glycerine/rmq
   project embeds R inside a Go binary, and libzipfs allows R libraries
   to be easily shipped with the binary.


## origins

This library is dervied from Tommi Virtanen's work https://github.com/bazil/zipfs,
which is fantastic and provides a fuse-mounted read-only filesystem from a zipfile.
zipfs and https://github.com/bazil/fuse are doing the heavy lifting 
behind the scenes.

## libzipfs then goes a step beyond what zipfs provides

We then add the ability to assemble a "combined" file from an executable
and a zipfile. The structure of the combined file looks like this:

~~~
----------------------------------------------------
| executable      |  zip file   |  256-byte footer |
----------------------------------------------------
~~~

The embedded zip file will be access and served via a fuse mountpoint.
The executable will contain code to accomplish this. The 256-bite
footer at the end of the file describes the location of the
embedded zip file. The combined file is still an executable,
and can be run directly.

### creating a combined executable and zipfile

the `libzipfs-combiner` utility does this for you.

For example: assuming that `my.go.binary` and `my.media.zip` already exist,
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

$ libzipfs-combiner --exe my.go.binary -o my.go.binary.combo -zip my.media.zip
~~~

### api/code code inside your `my.go.binary.combo` binary:

~~~
import (
  "github.com/glycerine/libzipfs"
)

...

z, mountpoint, err := libzipfs.MountComboZip()
if err != nil {
   panic(err)
}
defer z.Stop() // if you want to stop serving files 

// access the files from `my.media.zip` at mountpoint

f, err := os.Open(path.Join(mountpoint, "mydir", "myfile"))
...
~~~

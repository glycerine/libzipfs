package libzipfs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"archive/zip"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

// track git version of this lib
var GITLASTTAG string    // git describe --abbrev=0 --tags
var GITLASTCOMMIT string // git rev-parse HEAD

func VersionString() string {
	return fmt.Sprintf("%s/%s", GITLASTTAG, GITLASTCOMMIT)
}

func DisplayVersionAndExitIfRequested() {
	for i := range os.Args {
		if os.Args[i] == "-version" || os.Args[i] == "--version" {
			fmt.Printf("%s\n", VersionString())
			os.Exit(0)
		}
	}
}

// We assume the zip file contains entries for directories too.

var progName = filepath.Base(os.Args[0])

type FuseZipFs struct {
	ZipfilePath string
	MountPoint  string

	Ready   chan bool
	ReqStop chan bool
	Done    chan bool

	mut      sync.Mutex
	stopped  bool
	serveErr error
	connErr  error
	conn     *fuse.Conn

	filesys *FS
	//archive *zip.ReadCloser
	archive *zip.Reader

	offset      int64
	bytesAvail  int64 // -1 => unknown
	footerBytes int64

	fd *os.File
}

// Mount a possibly combined/zipfile at mountpiont. Call Start() to start servicing fuse reads.
//
// If the file has a libzipfs footer on it, set footerBytes == LIBZIPFS_FOOTER_LEN.
// The bytesAvail value should describe how long the zipfile is in bytes, and byteOffsetToZipFileStart
// should describe how far into the (possibly combined) zipFilePath the actual zipfile starts.
func NewFuseZipFs(zipFilePath, mountpoint string, byteOffsetToZipFileStart int64, bytesAvail int64, footerBytes int64) *FuseZipFs {

	// must trim any trailing slash from the mountpoint, or else mount can fail
	mountpoint = TrimTrailingSlashes(mountpoint)

	p := &FuseZipFs{
		ZipfilePath: zipFilePath,
		MountPoint:  mountpoint,
		Ready:       make(chan bool),
		ReqStop:     make(chan bool),
		Done:        make(chan bool),
		offset:      byteOffsetToZipFileStart,
		bytesAvail:  bytesAvail,
		footerBytes: footerBytes,
	}

	return p
}

// The Main API entry point for mounting a combo file vis FUSE to make
// the embedded Zip file directory available. Users should call
// fzfs.Stop() when/if they wish to stop serving files at mountpoint.
//
func MountComboZip() (fzfs *FuseZipFs, mountpoint string, err error) {
	comboFilePath := os.Args[0]
	fzfs, mountpoint, err = NewFuzeZipFsFromCombo(comboFilePath)
	if err != nil {
		return nil, "", err
	}
	err = fzfs.Start()
	if err != nil {
		return nil, "", err
	}
	return fzfs, mountpoint, nil
}

// mount the comboFilePath file in a temp directory mountpoint created
// just for this purpose, and return the mountpoint and a handle to the
// fuse fileserver in fzfs.
func NewFuzeZipFsFromCombo(comboFilePath string) (fzfs *FuseZipFs, mountpoint string, err error) {
	dir := "" // => use system tmp dir
	mountPoint, err := ioutil.TempDir(dir, "libzipfs.auto-combo.")
	if err != nil {
		return nil, "", fmt.Errorf("NewFuzeZipFsFromCombo() error, could not create mountpoint: '%s'", err)
	}
	VPrintf("\n\n mountPoint = '%s'\n", mountPoint)

	_, foot, comb, err := ReadFooter(comboFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("NewFuzeZipFsFromCombo() error, could not reader "+
			"Footer from comboFilePath '%s': '%s'",
			comboFilePath, err)
	}
	defer comb.Close()
	byteOffsetToZipFileStart := foot.ExecutableLengthBytes

	z := NewFuseZipFs(comboFilePath, mountPoint, byteOffsetToZipFileStart, foot.ZipfileLengthBytes, LIBZIPFS_FOOTER_LEN)
	return z, mountPoint, nil
}

func (p *FuseZipFs) Stop() error {
	p.mut.Lock()
	defer p.mut.Unlock()
	if p.stopped {
		return nil
	}
	err := p.unmount()
	if err != nil {
		VPrintf("unmount() of p.MountPoint='%s' failed with error: '%s'\n", p.MountPoint, err)
		return err
	}
	VPrintf("unmount() of p.MountPoint='%s' succeeded.\n", p.MountPoint)
	p.stopped = true
	<-p.Done

	p.fd.Close()
	p.conn.Close()

	//  we don't do the following anymore since forcing the unmount
	//  always results in 'bad file descriptor'.
	// if p.serveErr != nil {
	// return p.serveErr //  always 'bad file descriptor', so skip
	// }

	// check if the mount process has an error to report:
	<-p.conn.Ready
	p.connErr = p.conn.MountError
	return p.connErr
}

func (p *FuseZipFs) Start() error {
	var err error

	if p.bytesAvail <= 0 {
		statinfo, err := os.Stat(p.ZipfilePath)
		if err != nil {
			return err
		}
		p.bytesAvail = statinfo.Size() - (p.offset + p.footerBytes)
		if p.bytesAvail <= 0 {
			return fmt.Errorf("FuseZipFs.Start() error: no bytes available to read from ZipfilePath '%s' (of size %d bytes) after subtracting offset %d", p.ZipfilePath, statinfo.Size(), p.offset)
		}
	}

	fd, err := os.Open(p.ZipfilePath)
	if err != nil {
		return err
	}
	p.fd = fd
	rat := io.NewSectionReader(p.fd, p.offset, p.bytesAvail)

	p.archive, err = zip.NewReader(rat, p.bytesAvail)
	if err != nil {
		return err
	}

	c, err := fuse.Mount(p.MountPoint)
	if err != nil {
		return err
	}
	p.conn = c

	p.filesys = &FS{
		archive: p.archive,
	}

	go func() {
		select {
		case <-p.ReqStop:
		case <-p.Done:
		}
		p.Stop() // be sure we cleanup
	}()

	go func() {
		p.serveErr = fs.Serve(c, p.filesys)

		// shutdown sequence: possibly requested, possibly an error.
		close(p.Done)
	}()

	err = WaitUntilMounted(p.MountPoint)
	if err != nil {
		return fmt.Errorf("FuseZipFs.Start() error: could not detect mounted filesystem at mount point %s: '%s'", p.MountPoint, err)
	}
	close(p.Ready)

	return nil
}

type FS struct {
	archive *zip.Reader
}

var _ fs.FS = (*FS)(nil)

func (f *FS) Root() (fs.Node, error) {
	n := &Dir{
		archive: f.archive,
	}
	return n, nil
}

type Dir struct {
	archive *zip.Reader
	// nil for the root directory, which has no entry in the zip
	file *zip.File
}

var _ fs.Node = (*Dir)(nil)

func zipAttr(f *zip.File, a *fuse.Attr) {
	a.Size = f.UncompressedSize64
	a.Mode = f.Mode()
	a.Mtime = f.ModTime()
	a.Ctime = f.ModTime()
	a.Crtime = f.ModTime()
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	if d.file == nil {
		// root directory
		a.Mode = os.ModeDir | 0755
		return nil
	}
	zipAttr(d.file, a)
	return nil
}

var _ = fs.NodeRequestLookuper(&Dir{})

func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	path := req.Name
	if d.file != nil {
		path = d.file.Name + path
	}
	for _, f := range d.archive.File {
		switch {
		case f.Name == path:
			child := &File{
				file: f,
			}
			return child, nil
		case f.Name[:len(f.Name)-1] == path && f.Name[len(f.Name)-1] == '/':
			child := &Dir{
				archive: d.archive,
				file:    f,
			}
			return child, nil
		}
	}
	return nil, fuse.ENOENT
}

var _ = fs.HandleReadDirAller(&Dir{})

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	prefix := ""
	if d.file != nil {
		prefix = d.file.Name
	}

	var res []fuse.Dirent
	for _, f := range d.archive.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		name := f.Name[len(prefix):]
		if name == "" {
			// the dir itself, not a child
			continue
		}
		if strings.ContainsRune(name[:len(name)-1], '/') {
			// contains slash in the middle -> is in a deeper subdir
			continue
		}
		var de fuse.Dirent
		if name[len(name)-1] == '/' {
			// directory
			name = name[:len(name)-1]
			de.Type = fuse.DT_Dir
		}
		de.Name = name
		res = append(res, de)
	}
	return res, nil
}

type File struct {
	file *zip.File
}

var _ fs.Node = (*File)(nil)

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	zipAttr(f.file, a)
	return nil
}

var _ = fs.NodeOpener(&File{})

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	r, err := f.file.Open()
	if err != nil {
		return nil, err
	}
	// individual entries inside a zip file are not seekable
	resp.Flags |= fuse.OpenNonSeekable
	return &FileHandle{r: r}, nil
}

type FileHandle struct {
	r io.ReadCloser
}

var _ fs.Handle = (*FileHandle)(nil)

var _ fs.HandleReleaser = (*FileHandle)(nil)

func (fh *FileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return fh.r.Close()
}

var _ = fs.HandleReader(&FileHandle{})

func (fh *FileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// We don't actually enforce Offset to match where previous read
	// ended. Maybe we should, but that would mean'd we need to track
	// it. The kernel *should* do it for us, based on the
	// fuse.OpenNonSeekable flag.
	//
	// One exception to the above is if we fail to fully populate a
	// page cache page; a read into page cache is always page aligned.
	// Make sure we never serve a partial read, to avoid that.
	buf := make([]byte, req.Size)
	n, err := io.ReadFull(fh.r, buf)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	resp.Data = buf[:n]
	return err
}

// helper for reading in a loop. will panic on unknown error.
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}
	switch e := err.(type) {
	case *os.PathError:
		if strings.HasSuffix(e.Error(), "interrupted system call") {
			return true // EINTR, must simply retry.
		}
		panic(fmt.Errorf("got unknown os.PathError, e = '%#v'. e.Error()='%#v'\n", e, e.Error()))
	default:
		fmt.Printf("unknown err was '%#v' / '%s'\n", err, err.Error())
		panic(err)
	}
}

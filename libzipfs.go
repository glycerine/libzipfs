package libzipfs

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bytes"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

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
}

func NewFuseZipFs(zipFilePath, mountpoint string) *FuseZipFs {
	p := &FuseZipFs{
		ZipfilePath: zipFilePath,
		MountPoint:  mountpoint,
		Ready:       make(chan bool),
		ReqStop:     make(chan bool),
		Done:        make(chan bool),
	}

	return p
}

func (p *FuseZipFs) Stop() error {
	p.mut.Lock()
	defer p.mut.Unlock()
	if p.stopped {
		return nil
	}
	err := p.unmount()
	if err != nil {
		return err
	}

	p.stopped = true
	<-p.Done

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
	archive, err := zip.OpenReader(p.ZipfilePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	c, err := fuse.Mount(p.MountPoint)
	if err != nil {
		return err
	}
	p.conn = c
	defer c.Close()

	filesys := &FS{
		archive: &archive.Reader,
	}

	go func() {
		select {
		case <-p.ReqStop:
		case <-p.Done:
		}
		p.Stop() // be sure we cleanup
	}()

	go func() {
		p.serveErr = fs.Serve(c, filesys)

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

func WaitUntilMounted(mountPoint string) error {

	mpBytes := []byte(mountPoint)
	dur := 3 * time.Millisecond
	tries := 40
	var found bool
	for i := 0; i < tries; i++ {
		out, err := exec.Command(`/sbin/mount`).Output()
		if err != nil {
			return fmt.Errorf("could not query for mount points with /sbin/mount: '%s'", err)
		}
		VPrintf("\n out = '%s'\n", string(out))
		found = bytes.Contains(out, mpBytes)
		if found {
			VPrintf("\n found mountPoint '%s' on try %d\n", mountPoint, i+1)
			return nil
		}
		time.Sleep(dur)
	}
	return fmt.Errorf("WaitUntilMounted() error: could not locate mount point '%s' in /sbin/mount output, "+
		"even after %d tries with %v sleep between.", mountPoint, tries, dur)
}

func (p *FuseZipFs) unmount() error {

	err := exec.Command(`/sbin/umount`, p.MountPoint).Run()
	if err != nil {
		return fmt.Errorf("Unmount() error: could not /sbin/umount %s: '%s'", p.MountPoint, err)
	}

	err = WaitUntilUnmounted(p.MountPoint)
	if err != nil {
		return fmt.Errorf("Unmount() error: tried to wait for mount %s to become unmounted, but got error: '%s'", p.MountPoint, err)
	}
	return nil
}

func WaitUntilUnmounted(mountPoint string) error {

	mpBytes := []byte(mountPoint)
	dur := 3 * time.Millisecond
	tries := 40
	var found bool
	for i := 0; i < tries; i++ {
		out, err := exec.Command(`/sbin/mount`).Output()
		if err != nil {
			return fmt.Errorf("could not query for mount points with /sbin/mount: '%s'", err)
		}
		VPrintf("\n out = '%s'\n", string(out))
		found = bytes.Contains(out, mpBytes)
		if !found {
			VPrintf("\n mountPoint '%s' was not in mount output on try %d\n", mountPoint, i+1)
			return nil
		}
		time.Sleep(dur)
	}
	return fmt.Errorf("WaitUntilUnmounted() error: mount point '%s' in /sbin/mount was always present, "+
		"even after %d waits with %v sleep between each.", mountPoint, tries, dur)
}

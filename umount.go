package libzipfs

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// locate the mount and umount commands in the filesystem

type mountCmdLoc struct {
	MountPath  string
	UmountPath string
}

var utilLoc mountCmdLoc

func WaitUntilMounted(mountPoint string) error {

	mpBytes := []byte(mountPoint)
	dur := 3 * time.Millisecond
	tries := 40
	var found bool
	for i := 0; i < tries; i++ {
		out, err := exec.Command(utilLoc.MountPath).Output()
		if err != nil {
			return fmt.Errorf("could not query for mount points with %s: '%s'", utilLoc.MountPath, err)
		}
		VPrintf("\n out = '%s'\n", string(out))
		found = bytes.Contains(out, mpBytes)
		if found {
			VPrintf("\n found mountPoint '%s' on try %d\n", mountPoint, i+1)
			return nil
		}
		time.Sleep(dur)
	}
	return fmt.Errorf("WaitUntilMounted() error: could not locate mount point '%s' in %s output, "+
		"even after %d tries with %v sleep between.", mountPoint, utilLoc.MountPath, tries, dur)
}

//
// linux when regular user umount attempts:
//  getting error: umount: /tmp/libzipfs694201669 is not in the fstab (and you are not root)
//   => need to do fusermount -u mnt  instead of umount
func (p *FuseZipFs) unmount() error {
	args := []string{p.MountPoint}
	if strings.HasSuffix(utilLoc.UmountPath, `fusermount`) {
		args = []string{"-u", p.MountPoint}
	}

	// exactly two attemps seems to be exactly what is needed
	pasted := strings.Join(args, " ")
	sleepDur := 20 * time.Millisecond
	tries := 2
	k := 0
	for {
		out, err := exec.Command(utilLoc.UmountPath, args...).CombinedOutput()
		if err != nil {
			VPrintf("\n *** Unmount() error: could not %s %s: '%s' / output: '%s'. That was attempt %d.\n", utilLoc.UmountPath, pasted, err, string(out), k+1)
			k++
			if k >= tries {
				// generally we'll have been successful on the 2nd try even though the error still shows up.
				break
			}
			time.Sleep(sleepDur)
		}
	}

	err := WaitUntilUnmounted(p.MountPoint)
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
		out, err := exec.Command(utilLoc.MountPath).Output()
		if err != nil {
			return fmt.Errorf("could not query for mount points with %s: '%s'", utilLoc.MountPath, err)
		}
		VPrintf("\n out = '%s'\n", string(out))
		found = bytes.Contains(out, mpBytes)
		if !found {
			VPrintf("\n mountPoint '%s' was not in mount output on try %d\n", mountPoint, i+1)
			return nil
		}
		time.Sleep(dur)
	}
	return fmt.Errorf("WaitUntilUnmounted() error: mount point '%s' in (output of %s invocation) was always present, "+
		"even after %d waits with %v sleep between each.", mountPoint, utilLoc.MountPath, tries, dur)
}

func FindMountUmount() error {
	err := FindMount()
	if err != nil {
		return err
	}
	err = FindUmount()
	if err != nil {
		return err
	}
	return nil
}

func FindMount() error {
	candidates := []string{`/sbin/mount`, `/bin/mount`, `/usr/sbin/mount`, `/usr/bin/mount`}
	for _, f := range candidates {
		if FileExists(f) {
			utilLoc.MountPath = f
			return nil
		}
	}
	return fmt.Errorf("mount not found")
}

func FindUmount() error {
	// put the linux fusermount utils first
	candidates := []string{`/bin/fusermount`, `/sbin/fusermount`, `/sbin/umount`, `/bin/umount`, `/usr/sbin/umount`, `/usr/bin/umount`}
	for _, f := range candidates {
		if FileExists(f) {
			utilLoc.UmountPath = f
			return nil
		}
	}
	return fmt.Errorf("umount not found")
}

func init() {
	err := FindMountUmount()
	if err != nil {
		panic(err)
	}
}

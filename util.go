package libzipfs

// must trim any trailing slash from the mountpoint, or else mount can fail
func TrimTrailingSlashes(mountpoint string) string {
	m := len(mountpoint) - 1
	for i := 0; i < m; i++ {
		if mountpoint[m-i] == '/' {
			mountpoint = mountpoint[:(m - i)]
		} else {
			break
		}
	}
	return mountpoint
}

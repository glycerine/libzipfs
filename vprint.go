package libzipfs

import (
	"fmt"
	"time"
)

var Verbose bool

func VPrintf(format string, a ...interface{}) {
	if Verbose {
		TSPrintf(format, a...)
	}
}

func ts() string {
	return time.Now().Format("2006-01-02 15:04:05.999 -0700 MST")
}

// time-stamped printf
func TSPrintf(format string, a ...interface{}) {
	fmt.Printf("%s ", ts())
	fmt.Printf(format, a...)
}

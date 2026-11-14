package models

import "golang.org/x/sys/unix"

// Memory is the computer's memory, in bytes, or 0 when unknown.
func Memory() uint64 {
	var info unix.Sysinfo_t
	if unix.Sysinfo(&info) != nil {
		return 0
	}
	return uint64(info.Totalram) * uint64(info.Unit)
}

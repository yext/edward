// +build !darwin,!linux,!freebsd,!openbsd,!solaris,!windows

package mem

import "github.com/theothertomelliott/gopsutil-nocgo/internal/common"

func VirtualMemory() (*VirtualMemoryStat, error) {
	return nil, common.ErrNotImplementedError
}

func SwapMemory() (*SwapMemoryStat, error) {
	return nil, common.ErrNotImplementedError
}

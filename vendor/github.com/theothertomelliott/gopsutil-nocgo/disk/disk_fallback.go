// +build !darwin,!linux,!freebsd,!openbsd,!windows

package disk

import "github.com/theothertomelliott/gopsutil-nocgo/internal/common"

func IOCounters() (map[string]IOCountersStat, error) {
	return nil, common.ErrNotImplementedError
}

func Partitions(all bool) ([]PartitionStat, error) {
	return []PartitionStat{}, common.ErrNotImplementedError
}

func Usage(path string) (*UsageStat, error) {
	return nil, common.ErrNotImplementedError
}

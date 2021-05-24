package project

import (
	"github.com/werf/lockgate"

	"github.com/werf/trdl/pkg/trdl"
	"github.com/werf/trdl/pkg/util"
)

func (c Client) ExecChannelReleaseBin(group, channel string, optionalBinName string, args []string) error {
	return lockgate.WithAcquire(c.locker, c.groupChannelLockName(group, channel), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		path, err := c.channelReleaseBinPath(group, channel, optionalBinName)
		if err != nil {
			return err
		}

		return util.Exec(path, args)
	})
}

package project

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/theupdateframework/go-tuf/client"
	leveldbstore "github.com/theupdateframework/go-tuf/client/leveldbstore"

	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_locker"

	"github.com/werf/trdl/pkg/util"
)

const (
	targetsChannels = "channels"
	targetsReleases = "releases"

	channelsDir = targetsChannels
	releasesDir = targetsReleases
)

type Client struct {
	projectName string
	directory   string
	tufClient   *client.Client
	locker      lockgate.Locker
}

func NewClient(projectName, directory, repoUrl, locksPath string) (Client, error) {
	c := Client{
		projectName: projectName,
		directory:   directory,
	}

	if err := c.init(repoUrl, locksPath); err != nil {
		return c, err
	}

	return c, nil
}

func (c *Client) init(repoUrl string, locksPath string) error {
	if err := c.initFileLocker(locksPath); err != nil {
		return err
	}

	if err := c.initTufClient(repoUrl); err != nil {
		return err
	}

	return nil
}

func (c *Client) initTufClient(repoUrl string) error {
	local, err := leveldbstore.FileLocalStore(c.metaLocalStoreDir())
	if err != nil {
		return err
	}

	remote, err := client.HTTPRemoteStore(repoUrl, nil, nil)
	if err != nil {
		return err
	}

	c.tufClient = client.NewClient(local, remote)

	return nil
}

func (c *Client) initFileLocker(locksPath string) error {
	locker, err := file_locker.NewFileLocker(locksPath)
	if err != nil {
		return err
	}

	c.locker = locker

	return nil
}

func (c Client) syncMeta() error {
	if _, err := c.tufClient.Update(); err != nil && !client.IsLatestSnapshot(err) {
		return err
	}

	return nil
}

func (c Client) channelTargetName(group, channel string) string {
	return path.Join(targetsChannels, group, channel)
}

func (c Client) channelPath(group, channel string) string {
	return filepath.Join(c.directory, channelsDir, group, channel)
}

func (c Client) channelReleaseBinPath(group, channel string, optionalBinName string) (string, error) {
	dir, releaseName, err := c.channelReleaseBinDir(group, channel)
	if err != nil {
		return "", err
	}

	var glob string
	if optionalBinName == "" {
		glob = filepath.Join(dir, "*")
	} else {
		glob = filepath.Join(dir, optionalBinName)
	}

	matches, err := filepath.Glob(glob)
	if err != nil {
		return "", fmt.Errorf("unable to glob files: %s", err)
	}

	if len(matches) > 1 {
		var names []string
		for _, m := range matches {
			names = append(names, strings.TrimLeft(m, dir+string(os.PathSeparator)))
		}

		return "", NewErrChannelReleaseSeveralFilesFound(c.projectName, group, channel, releaseName, names)
	} else if len(matches) == 0 {
		if optionalBinName == "" {
			return "", fmt.Errorf("binary file not found in release")
		} else {
			return "", fmt.Errorf("binary file %q not found in release", optionalBinName)
		}
	}

	return matches[0], nil
}

func (c Client) channelReleaseBinDir(group, channel string) (dir string, release string, err error) {
	releaseDir, releaseName, err := c.channelReleaseDir(group, channel)
	if err != nil {
		return "", "", err
	}

	binDir := filepath.Join(releaseDir, "bin")
	exist, err := util.IsDirExist(binDir)
	if err != nil {
		return "", "", fmt.Errorf("unable to check existence of directory %q: %s", binDir, err)
	}

	if !exist {
		return "", "", fmt.Errorf("bin directory not found in release directory (group: %q, channel: %q)", group, channel)
	}

	return binDir, releaseName, nil
}

func (c Client) channelReleaseDir(group, channel string) (dir string, release string, err error) {
	release, err = c.channelRelease(group, channel)
	if err != nil {
		return "", "", err
	}

	dirGlob := filepath.Join(c.directory, releasesDir, release, "*")

	matches, err := filepath.Glob(dirGlob)
	if err != nil {
		return "", "", fmt.Errorf("unable to glob files: %s", err)
	}

	if len(matches) > 1 {
		return "", "", fmt.Errorf("unexpected files in release directory:\n - %s", strings.Join(matches, "\n - "))
	} else if len(matches) == 0 {
		return "", "", NewErrChannelReleaseNotFoundLocally(c.projectName, group, channel, release)
	}

	return matches[0], release, nil
}

func (c Client) channelRelease(group, channel string) (string, error) {
	channelFilePath := c.channelPath(group, channel)
	exist, err := util.IsRegularFileExist(channelFilePath)
	if err != nil {
		return "", fmt.Errorf("unable to check existence of file %q: %s", channelFilePath, err)
	}

	if !exist {
		return "", NewErrChannelNotFoundLocally(c.projectName, group, channel)
	}

	channelData, err := ioutil.ReadFile(channelFilePath)
	if err != nil {
		return "", fmt.Errorf("unable to read file %q: %s", channelFilePath, err)
	}

	releaseName := strings.TrimSpace(string(channelData))
	return releaseName, nil
}

func (c Client) metaLocalStoreDir() string {
	return filepath.Join(c.directory, ".meta")
}

func (c Client) groupChannelLockName(group, channel string) string {
	return fmt.Sprintf("%s-%s", group, channel)
}

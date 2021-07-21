package publisher

import (
	"context"
	"io"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/config"
)

type Interface interface {
	GetRepository(ctx context.Context, storage logical.Storage, options RepositoryOptions) (RepositoryInterface, error)
	PublishReleaseTarget(ctx context.Context, repository RepositoryInterface, releaseName, path string, data io.Reader) error
	PublishChannelsConfig(ctx context.Context, repository RepositoryInterface, trdlChannelsConfig *config.TrdlChannels) error
	PublishInMemoryFiles(ctx context.Context, repository RepositoryInterface, files []*InMemoryFile) error
}

type RepositoryInterface interface {
	Init() error
	SetPrivKeys(privKeys TufRepoPrivKeys) error
	GetPrivKeys() TufRepoPrivKeys
	GenPrivKeys() error

	PublishTarget(ctx context.Context, pathInsideTargets string, data io.Reader) error
	Commit(ctx context.Context) error
}

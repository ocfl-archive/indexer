package indexer

import (
	"github.com/je4/utils/v2/pkg/zLogger"
	archiveerror "github.com/ocfl-archive/error/pkg/error"
	"github.com/ocfl-archive/indexer/v3/internal"
)

var ErrorFactory = archiveerror.NewFactory("INDEXER")

type errorID = archiveerror.ID

const (
	IndexerInit = "IndexerInit"
)

func configErrorFactory(logger zLogger.ZLogger) {
	var err error
	const errorsEmbedToml string = "errors.toml"
	archiveErrs, err := archiveerror.LoadTOMLFileFS(internal.InternalFS, errorsEmbedToml)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot load error config file")
	}

	if err := ErrorFactory.RegisterErrors(archiveErrs); err != nil {
		logger.Fatal().Err(err).Msg("cannot load error config file")
	}
}

// Package util provides helper utilities, initialization routines, and indexing wrappers.
package util

import (
	"context"
	"io"
	"io/fs"

	"emperror.dev/errors"
	"github.com/je4/utils/v2/pkg/checksum"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
)

// Indexer is a wrapper around indexer.ActionDispatcher providing convenience indexing functions.
type Indexer indexer.ActionDispatcher

// ActionDispatcher returns the underlying *indexer.ActionDispatcher instance.
func (idx *Indexer) ActionDispatcher() *indexer.ActionDispatcher {
	return (*indexer.ActionDispatcher)(idx)
}

// Index reads a file from an fs.FS filesystem, calculates specified checksum digests,
// and streams the content through configured actions simultaneously.
func (idx *Indexer) Index(fsys fs.FS, path string, realname string, actions []string, digestAlgs []checksum.DigestAlgorithm, writer io.Writer, logger zLogger.ZLogger) (*indexer.ResultV2, map[checksum.DigestAlgorithm]string, error) {
	return idx.IndexContext(context.Background(), fsys, path, realname, actions, digestAlgs, writer, logger)
}

// IndexContext reads a file from an fs.FS filesystem, calculates specified checksum digests,
// and streams the content through configured actions simultaneously with context cancellation support.
func (idx *Indexer) IndexContext(ctx context.Context, fsys fs.FS, path string, realname string, actions []string, digestAlgs []checksum.DigestAlgorithm, writer io.Writer, logger zLogger.ZLogger) (*indexer.ResultV2, map[checksum.DigestAlgorithm]string, error) {
	if realname == "" {
		realname = path
	}
	ad := (*indexer.ActionDispatcher)(idx)
	fp, err := fsys.Open(path)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot open '%v/%s'", fsys, path)
	}
	idxRead, idxWrite := io.Pipe()
	csw, err := checksum.NewChecksumWriter(digestAlgs, io.MultiWriter(idxWrite, writer))
	if err != nil {
		idxWrite.Close()
		fp.Close()
		return nil, nil, errors.Wrapf(err, "cannot create ChecksumWriter for digests %v", digestAlgs)
	}

	go func() {
		defer func() {
			csw.Close()
			idxWrite.Close()
			fp.Close()
		}()
		_, err := io.Copy(csw, fp)
		if err != nil {
			// todo: channel with error
			logger.Error().Err(err).Msg("cannot copy data")
		}
	}()
	result, err := ad.StreamContext(ctx, idxRead, []string{realname}, actions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot index '%s/%s'", fsys, path)
	}

	digests, err := csw.GetChecksums()
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot get checksums")
	}

	return result, digests, nil
}

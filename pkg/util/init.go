package util

import (
	"io"
	"os"
	"strconv"

	"emperror.dev/errors"
	"github.com/dgraph-io/badger/v4"
	"github.com/je4/utils/v2/pkg/zLogger"
	datasiegfried "github.com/ocfl-archive/indexer/v3/data/siegfried"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
)

type _closer []io.Closer

func (c _closer) AddCloser(closer io.Closer) {
	c = append(c, closer)
}

func (c _closer) Close() error {
	var errs = []error{}
	for _, closer := range c {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Combine(errs...)
}

// InitIndexer
// initializes an ActionDispatcher with Siegfried, ImageMagick, FFPMEG and Tika
// the actions are named "siegfried", "identify", "ffprobe", "tika" and "fulltext"
func InitIndexer(conf *indexer.IndexerConfig, logger zLogger.ZLogger) (ad *Indexer, actions []string, closer io.Closer, err error) {
	actions = []string{}
	closerList := _closer{}
	closer = closerList
	var relevance = map[int]indexer.MimeWeightString{}

	if conf.Optimize {
		if err := OptimizeConfig(conf, logger); err != nil {
			return nil, nil, nil, errors.Wrap(err, "optimize config")
		}
	}

	if conf.MimeRelevance != nil {
		for key, val := range conf.MimeRelevance {
			num, err := strconv.Atoi(key)
			if err != nil {
				logger.Error().Msgf("cannot convert mimerelevance key '%s' to int", key)
				continue
			}
			relevance[num] = indexer.MimeWeightString(val)
		}
	}

	ad = (*Indexer)(indexer.NewActionDispatcher(relevance))
	var signature []byte
	if conf.Siegfried.Enabled {
		if conf.Siegfried.SignatureFile == "" || conf.Siegfried.SignatureFile == "internal" {
			signature = datasiegfried.DefaultSig
		} else {
			signature, err = os.ReadFile(conf.Siegfried.SignatureFile)
			if err != nil {
				closer.Close()
				return nil, nil, nil, errors.Wrapf(err, "cannot read siegfried signature file '%s'", conf.Siegfried.SignatureFile)
			}
		}
		_ = indexer.NewActionSiegfried("siegfried", signature, conf.Siegfried.MimeMap, conf.Siegfried.TypeMap, ad.ActionDispatcher(), conf.Siegfried.StreamSize)
		logger.Info().Msg("indexer action siegfried added")
		actions = append(actions, "siegfried")
	}

	if conf.XML.Enabled {
		_ = indexer.NewActionXML("xml", conf.XML.Format, ad.ActionDispatcher())
		logger.Info().Msg("indexer action xml added")
		actions = append(actions, "xml")
	}
	if conf.JSON.Enabled {
		_ = indexer.NewActionJSON(indexer.NameJSON, conf.JSON.Format, ad.ActionDispatcher())
		logger.Info().Msg("indexer action json added")
		actions = append(actions, "json")
	}

	if conf.FFMPEG.Enabled {
		_ = indexer.NewActionFFProbe("ffprobe", conf.FFMPEG.FFProbe, conf.FFMPEG.Wsl, conf.FFMPEG.Timeout.Duration, conf.FFMPEG.Online, conf.FFMPEG.Mime, ad.ActionDispatcher())
		logger.Info().Msg("indexer action ffprobe added")
		actions = append(actions, "ffprobe")
	}
	if conf.ImageMagick.Enabled {
		_ = indexer.NewActionIdentifyV2("identify", conf.ImageMagick.Identify, conf.ImageMagick.Convert, conf.ImageMagick.Wsl, conf.ImageMagick.Timeout.Duration, conf.ImageMagick.Online, ad.ActionDispatcher())
		logger.Info().Msg("indexer action identify added")
		actions = append(actions, "identify")
	}
	if conf.Tika.Enabled {
		if conf.Tika.AddressMeta != "" {
			_ = indexer.NewActionTika("tika", conf.Tika.AddressMeta, conf.Tika.Timeout.Duration, conf.Tika.RegexpMimeMeta, conf.Tika.RegexpMimeMetaNot, "", conf.Tika.Online, ad.ActionDispatcher())
			logger.Info().Msg("indexer action tika added")
			actions = append(actions, "tika")
		}

		if conf.Tika.AddressFulltext != "" {
			_ = indexer.NewActionTika("fulltext", conf.Tika.AddressFulltext, conf.Tika.Timeout.Duration, conf.Tika.RegexpMimeFulltext, conf.Tika.RegexpMimeFulltextNot, "X-TIKA:content", conf.Tika.Online, ad.ActionDispatcher())
			logger.Info().Msg("indexer action fulltext added")
			actions = append(actions, "tika")
		}
	}

	if conf.Checksum.Enabled {
		indexer.NewActionChecksum("checksum", conf.Checksum.Digest, ad.ActionDispatcher())
		actions = append(actions, "checksum")
	}

	if conf.Clamav.Enabled {
		indexer.NewActionClamAV(conf.Clamav.ClamScan, conf.Clamav.Wsl, conf.Clamav.Timeout.Duration, ad.ActionDispatcher())
		actions = append(actions, "clamav")
	}

	if conf.NSRL.Enabled {
		var nsrldb *badger.DB
		if conf.NSRL.Enabled {
			stat2, err := os.Stat(conf.NSRL.Badger)
			if err != nil {
				closer.Close()
				return nil, nil, nil, errors.Wrapf(err, "cannot stat NSRL badger %s", conf.NSRL.Badger)
			}
			if !stat2.IsDir() {
				closer.Close()
				return nil, nil, nil, errors.Wrapf(err, "%s is not a directory", conf.NSRL.Badger)
			}

			bconfig := badger.DefaultOptions(conf.NSRL.Badger)
			bconfig.ReadOnly = true
			nsrldb, err = badger.Open(bconfig)
			if err != nil {
				closer.Close()
				return nil, nil, nil, errors.Wrapf(err, "cannot open NSRL badger %s", conf.NSRL.Badger)
			}
			//log.Infof("nsrl max batch count: %v", nsrldb.MaxBatchCount())
			//			defer nsrldb.Close()
			var keyCount uint32
			for _, tbl := range nsrldb.Tables() {
				keyCount += tbl.KeyCount
			}
			closerList.AddCloser(nsrldb)
			logger.Info().Msgf("NSRL-Table: %v keys", keyCount)
			indexer.NewActionNSRL("nsrl", nsrldb, ad.ActionDispatcher(), logger)
			actions = append(actions, "nsrl")
		}
	}

	for _, eaconfig := range conf.External {
		var caps uint
		for _, c := range eaconfig.ActionCapabilities {
			caps |= uint(c)
		}
		indexer.NewActionExternal(eaconfig.Name, eaconfig.Address, indexer.ActionCapability(caps), eaconfig.CallType, eaconfig.Mimetype, ad.ActionDispatcher())
		actions = append(actions, eaconfig.Name)
	}

	return
}

package indexer

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/je4/utils/v2/pkg/zLogger"
)

func stringMapToMimeRelevance(mimeRelevanceInterface map[string]ConfigMimeWeight) (map[int]MimeWeightString, error) {
	var mimeRelevance = map[int]MimeWeightString{}
	for keyStr, val := range mimeRelevanceInterface {
		key, err := strconv.Atoi(keyStr)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid key entry '%s' in 'MimeRelevance'", keyStr)
		}
		mimeRelevance[key] = MimeWeightString{
			Regexp: val.Regexp,
			Weight: val.Weight,
		}
	}
	return mimeRelevance, nil
}

var fsRegexp = regexp.MustCompile("^([^:]{2,}):(.+)$")

func logStartup(logger zLogger.ZLogger, tool string) {
	logger.Info().Any(
		ErrorFactory.LogError(
			IndexerInit,
			fmt.Sprintf("indexer action %s added", tool),
			nil),
	).Msg("")
}

func InitActionDispatcher(fss map[string]fs.FS, conf IndexerConfig, logger zLogger.ZLogger) (*ActionDispatcher, error) {
	mimeRelevance, err := stringMapToMimeRelevance(conf.MimeRelevance)
	if err != nil {
		return nil, errors.Wrap(err, "cannot convert config string map to mime relevance")
	}
	actionDispatcher := NewActionDispatcher(mimeRelevance)

	var signatureData []byte
	found := fsRegexp.FindStringSubmatch(conf.Siegfried.SignatureFile)
	if found == nil {
		signatureData, err = os.ReadFile(conf.Siegfried.SignatureFile)
		if err != nil {
			return nil, errors.Wrapf(err, "no siegfried signature file provided. using default signature file. please provide a recent signature file. %s", conf.Siegfried.SignatureFile)
			//			signatureData = datasiegfried.DefaultSig
		}
	} else {
		intFS, ok := fss[found[1]]
		if !ok {
			return nil, errors.Errorf("invalid filesystem %s", found[1])
		}
		signatureData, err = fs.ReadFile(intFS, strings.TrimLeft(found[2], "/"))
		if err != nil {
			return nil, errors.Wrapf(err, "no siegfried signature file provided. using default signature file. please provide a recent signature file. %s", conf.Siegfried.SignatureFile)
		}
	}

	configErrorFactory(logger)

	_ = NewActionSiegfried(
		NameSiegfried,
		signatureData,
		conf.Siegfried.MimeMap,
		conf.Siegfried.TypeMap,
		nil,
		actionDispatcher,
	)
	logStartup(logger, NameSiegfried)
	if conf.XML.Enabled {
		_ = NewActionXML(
			NameXML,
			conf.XML.Format,
			nil,
			actionDispatcher,
		)
		logStartup(logger, NameXML)
	}
	if conf.Checksum.Enabled {
		_ = NewActionChecksum(
			NameChecksum,
			conf.Checksum.Digest,
			nil,
			actionDispatcher,
		)
		logStartup(logger, NameChecksum)
	}
	if conf.FFMPEG.Enabled {
		_ = NewActionFFProbe(
			NameFFProbe,
			conf.FFMPEG.FFProbe,
			conf.FFMPEG.Wsl,
			conf.FFMPEG.Timeout.Duration,
			conf.FFMPEG.Online,
			conf.FFMPEG.Mime,
			nil,
			actionDispatcher)
		logStartup(logger, NameFFProbe)
	}
	if conf.ImageMagick.Enabled {
		_ = NewActionIdentifyV2(
			NameIdentify,
			conf.ImageMagick.Identify,
			conf.ImageMagick.Convert,
			conf.ImageMagick.Wsl,
			conf.ImageMagick.Timeout.Duration,
			conf.ImageMagick.Online, nil, actionDispatcher)
		logStartup(logger, NameIdentify)
	}
	if conf.Tika.Enabled {
		_ = NewActionTika(
			NameTika,
			conf.Tika.AddressMeta,
			conf.Tika.Timeout.Duration,
			conf.Tika.RegexpMimeMeta,
			conf.Tika.RegexpMimeMetaNot,
			"",
			conf.Tika.Online,
			nil, actionDispatcher)
		logStartup(logger, NameTika)
		_ = NewActionTika(
			NameFullText,
			conf.Tika.AddressFulltext,
			conf.Tika.Timeout.Duration,
			conf.Tika.RegexpMimeFulltext,
			conf.Tika.RegexpMimeFulltextNot,
			"X-TIKA:content",
			conf.Tika.Online,
			nil,
			actionDispatcher)
		logStartup(logger, NameFullText)
	}

	return actionDispatcher, nil
}

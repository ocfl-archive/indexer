// Copyright 2021 Juergen Enge, info-age GmbH, Basel. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package indexer

import (
	"encoding/json"

	"emperror.dev/errors"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/golang/snappy"
	"github.com/je4/utils/v2/pkg/zLogger"

	"io"
)

const NSRL_OS = "OpSystemCode-"
const NSRL_PROD = "ProductCode-"
const NSRL_MFG = "MFgCode-"
const NSRL_File = "SHA-1-"

func NewActionNSRL(name string, nsrldb *badger.DB, ad *ActionDispatcher, logger zLogger.ZLogger) Action {
	an := &ActionNSRL{name: name, nsrldb: nsrldb, caps: ACTFILE, logger: logger}
	ad.RegisterAction(an)
	return an
}

type ActionNSRL struct {
	name   string
	caps   ActionCapability
	nsrldb *badger.DB
	logger zLogger.ZLogger
}

func (aNSRL *ActionNSRL) DoV2(filename string) (*ResultV2, error) {
	//TODO implement me
	panic("implement me")
}

func (aNSRL *ActionNSRL) CanHandle(contentType string, filename string) bool {
	return true
}

func (aNSRL *ActionNSRL) Stream(contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	return nil, errors.New("nsrl actions does not support streaming")
}

type ActionNSRLMeta struct {
	File    map[string]string
	FileMfG map[string]string
	OS      map[string]string
	OSMfg   map[string]string
	Prod    map[string]string
	ProdMfg map[string]string
}

func (aNSRL *ActionNSRL) GetWeight() uint {
	return 100
}

func getStringMap(txn *badger.Txn, key string) ([]map[string]string, error) {
	var result []map[string]string
	item, err := txn.Get([]byte(key))
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "cannot get %s", key)
	}
	if err := item.Value(func(val []byte) error {
		jsonStr, err := snappy.Decode(nil, val)
		if err != nil {
			return errors.Wrapf(err, "cannot decompress snappy of %s", key)
		}
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return errors.Wrapf(err, "cannot unmarshal %s for %s", jsonStr, key)
		}
		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "cannot get data of %s", key)
	}
	return result, nil
}

func (aNSRL *ActionNSRL) getNSRL(sha1sum string) (interface{}, []string, []string, error) {
	var result []ActionNSRLMeta
	aNSRL.nsrldb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(NSRL_File + sha1sum))
		if err != nil {
			return errors.Wrapf(err, "cannot get os %s", NSRL_File+sha1sum)
		}
		var fileData []map[string]string
		if err := item.Value(func(val []byte) error {
			jsonStr, err := snappy.Decode(nil, val)
			if err != nil {
				return errors.Wrapf(err, "cannot decompress snappy of %s", NSRL_File+sha1sum)
			}
			if err := json.Unmarshal([]byte(jsonStr), &fileData); err != nil {
				return errors.Wrapf(err, "cannot unmarshal %s for %s", jsonStr, NSRL_File+sha1sum)
			}
			return nil
		}); err != nil {
			return errors.Wrapf(err, "cannot get value of %s", NSRL_File+sha1sum)
		}
		if len(fileData) > 10 {
			fileData = fileData[0:10]
		}
		for _, file := range fileData {
			var am ActionNSRLMeta
			am.File = file
			if am.File["MfgCode"] != "" {
				r, _ := getStringMap(txn, NSRL_MFG+am.Prod["MfgCode"])
				if len(r) > 0 {
					am.FileMfG = r[0]
				}
			}
			r, err := getStringMap(txn, NSRL_PROD+file["ProductCode"])
			if err != nil {
				aNSRL.logger.Error().Msgf("cannot get data of %s: %v", NSRL_PROD+file["ProductCode"], err)
				// return errors.Wrapf(err, "cannot get data of %s", NSRL_PROD+file["ProductCode"])
			}
			if len(r) > 0 {
				am.Prod = r[0]
			}
			if am.Prod["MfgCode"] != "" {
				r, _ = getStringMap(txn, NSRL_MFG+am.Prod["MfgCode"])
				if len(r) > 0 {
					am.ProdMfg = r[0]
				}
			}
			r, err = getStringMap(txn, NSRL_OS+file["OpSystemCode"])
			if err != nil {
				aNSRL.logger.Error().Msgf("cannot get data of %s: %v", NSRL_OS+file["OpSystemCode"], err)
				// return errors.Wrapf(err, "cannot get data of %s", NSRL_PROD+file["ProductCode"])
			}
			if len(r) > 0 {
				am.OS = r[0]
			}
			if am.OS["MfgCode"] != "" {
				r, _ = getStringMap(txn, NSRL_MFG+am.Prod["MfgCode"])
				if len(r) > 0 {
					am.OSMfg = r[0]
				}
			}
			result = append(result, am)
		}
		return nil
	})
	return result, []string{}, nil, nil
}

func (aNSRL *ActionNSRL) GetCaps() ActionCapability {
	return aNSRL.caps
}

func (aNSRL *ActionNSRL) GetName() string {
	return aNSRL.name
}

var (
	_ Action = &ActionNSRL{}
)

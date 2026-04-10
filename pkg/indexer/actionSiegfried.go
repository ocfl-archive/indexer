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
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"emperror.dev/errors"
	"github.com/richardlehane/siegfried"
	"github.com/richardlehane/siegfried/pkg/pronom"
)

type ActionSiegfried struct {
	name    string
	sf      *siegfried.Siegfried
	mimeMap map[string]string
	//	server        *Server
	typeMap       map[string]TypeSubtype
	signatureData []byte
	sfPool        sync.Pool
}

func (as *ActionSiegfried) CanHandle(contentType string, filename string) bool {
	return true
}

func NewActionSiegfried(name string, signatureData []byte, mimeMap map[string]string, typeMap map[string]TypeSubtype, ad *ActionDispatcher, streamSize int) Action {

	/*
		sf, err := siegfried.LoadReader(bytes.NewBuffer(signatureData))
		if err != nil {
			log.Fatalln(err)
		}
	*/
	if streamSize > 0 {
		siegfried.StreamSize(streamSize)
	}
	pool := sync.Pool{
		New: func() interface{} {
			sf, err := siegfried.LoadReader(bytes.NewBuffer(signatureData))
			if err != nil {
				log.Printf("Error instantiating siegfried: %v", err)
				return nil
			}
			return sf
		},
	}
	as := &ActionSiegfried{
		name: name,
		//sf:            sf,
		mimeMap: mimeMap,
		typeMap: typeMap,
		//		server:        server,
		signatureData: signatureData,
		sfPool:        pool,
	}
	ad.RegisterAction(as)
	return as
}

func (as *ActionSiegfried) GetWeight() uint {
	return 10
}

func (as *ActionSiegfried) GetCaps() ActionCapability {
	return ACTFILEHEAD | ACTSTREAM
}

func (as *ActionSiegfried) GetName() string {
	return as.name
}

func (as *ActionSiegfried) Stream(contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	sf := as.sfPool.Get().(*siegfried.Siegfried)
	if sf == nil {
		return nil, errors.New("cannot get siegfried from pool")
	}
	defer func() {
		as.sfPool.Put(sf)
	}()
	ident, err := sf.Identify(reader, filepath.Base(filename), "")
	if err != nil {
		return nil, errors.Wrapf(err, "cannot identify file %s", filename)
	}
	var result = NewResultV2()
	for _, id := range ident {
		if pid, ok := id.(pronom.Identification); ok {
			if pid.MIME != "" {
				result.Mimetypes = append(result.Mimetypes, pid.MIME)
			}
			if pid.ID != "" {
				result.Pronoms = append(result.Pronoms, pid.ID)
				if t, ok := as.typeMap[pid.ID]; ok {
					result.Type = t.Type
					result.Subtype = t.Subtype
				}
				if mime, ok := as.mimeMap[pid.ID]; ok {
					if mime != "" {
						result.Mimetypes = append(result.Mimetypes, mime)
					}
				}
			}

		}
	}
	result.Metadata[as.GetName()] = ident
	return result, nil
}

func (as *ActionSiegfried) DoV2(filename string) (*ResultV2, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open file '%s'", filename)
	}
	defer reader.Close()
	sf := as.sfPool.Get().(*siegfried.Siegfried)
	if sf == nil {
		return nil, errors.New("cannot get siegfried from pool")
	}
	defer func() {
		as.sfPool.Put(sf)
	}()
	ident, err := sf.Identify(reader, filepath.Base(filename), "")
	if err != nil {
		return nil, errors.Wrapf(err, "cannot identify file %s", filename)
	}
	var result = NewResultV2()
	for _, id := range ident {
		if pid, ok := id.(pronom.Identification); ok {
			if pid.MIME != "" {
				result.Mimetypes = append(result.Mimetypes, pid.MIME)
			}
			if pid.ID != "" {
				result.Pronoms = append(result.Pronoms, pid.ID)
				if t, ok := as.typeMap[pid.ID]; ok {
					result.Type = t.Type
					result.Subtype = t.Subtype
				}
				if mime, ok := as.mimeMap[pid.ID]; ok {
					if mime != "" {
						result.Mimetypes = append(result.Mimetypes, mime)
					}
				}
			}
		}
	}
	result.Metadata[as.GetName()] = ident

	return result, nil
}

var (
	_ Action = &ActionSiegfried{}
)

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
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
	"github.com/ocfl-archive/indexer/v3/pkg/util"
	ublogger "gitlab.switch.ch/ub-unibas/go-ublogger/v2"
	"go.ub.unibas.ch/cloud/certloader/v2/pkg/loader"
)

const INDEXER = "indexer v0.2, info-age GmbH Basel"

var configFile = flag.String("cfg", "", "config file location")
var inputFile = flag.String("in", "", "input file location")

func main() {
	var err error
	println(INDEXER)

	flag.Parse()

	var exPath = ""
	// if configfile not found try path of executable as prefix
	if !indexer.FileExists(*configFile) {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath = filepath.Dir(ex)
		if indexer.FileExists(filepath.Join(exPath, *configFile)) {
			*configFile = filepath.Join(exPath, *configFile)
		}
	}
	// configfile should exists at this place
	conf := LoadConfig(*configFile)

	var loggerTLSConfig *tls.Config
	var loggerLoader io.Closer
	if conf.Log.Stash.TLS != nil {
		loggerTLSConfig, loggerLoader, err = loader.CreateClientLoader(conf.Log.Stash.TLS, nil)
		if err != nil {
			log.Fatalf("cannot create client loader: %v", err)
		}
		defer func(loggerLoader io.Closer) {
			err := loggerLoader.Close()
			if err != nil {
				log.Printf("cannot close logger loader: %v", err)
			}
		}(loggerLoader)
	}

	// create logger instance
	_logger, _logstash, _logfile, err := ublogger.CreateUbMultiLoggerTLS(conf.Log.Level, conf.Log.File,
		ublogger.SetDataset(conf.Log.Stash.Dataset),
		ublogger.SetLogStash(conf.Log.Stash.LogstashHost, conf.Log.Stash.LogstashPort, conf.Log.Stash.Namespace, conf.Log.Stash.LogstashTraceLevel),
		ublogger.SetTLS(conf.Log.Stash.TLS != nil),
		ublogger.SetTLSConfig(loggerTLSConfig),
	)
	if err != nil {
		log.Fatalf("cannot create logger: %v", err)
	}
	if _logstash != nil {
		defer _logstash.Close()
	}
	if _logfile != nil {
		defer _logfile.Close()
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("cannot get hostname: %v", err)
	}
	l2 := _logger.With().Timestamp().Str("host", hostname).Logger() //.Output(output)
	var logger zLogger.ZLogger = &l2

	ad, actions, closer, err := util.InitIndexer(conf.Indexer, logger)
	if err != nil {
		log.Fatalf("Error initializing indexer: %v", err)
	}
	defer func(closer io.Closer) {
		err := closer.Close()
		if err != nil {
			logger.Error().Msgf("error closing indexer: %v", err)
		}
	}(closer)

	fp, err := os.Open(*inputFile)
	if err != nil {
		logger.Fatal().Msgf("cannot open input file: %v", err)
	}
	defer func(fp *os.File) {
		err := fp.Close()
		if err != nil {
			logger.Error().Msgf("error closing file: %v", err)
		}
	}(fp)

	result, err := ad.ActionDispatcher().Stream(fp, []string{filepath.Base(*inputFile)}, actions)
	if err != nil {
		logger.Error().Msgf("error streaming file: %v", err)
		return
	}
	fmt.Println(result)
}

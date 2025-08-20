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
	"context"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/v4"
	lm "github.com/je4/utils/v2/pkg/logger"
	datasiegfried "github.com/ocfl-archive/indexer/v3/internal/siegfried"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
	"github.com/ocfl-archive/indexer/v3/pkg/util"
)

const INDEXER = "indexer v0.2, info-age GmbH Basel"

func main() {
	println(INDEXER)

	configFile := flag.String("cfg", "./indexer.toml", "config file location")
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
		} else {
			log.Fatalf("cannot find configuration file: %v", *configFile)
			return
		}
	}
	// configfile should exists at this place
	config := LoadConfig(*configFile)
	if err := util.OptimizeConfig(config.Indexer); err != nil {
		log.Fatalf("Error optimizing config: %v", err)
	}

	// create logger instance
	log, lf := lm.CreateLogger("indexer", config.Logfile, nil, config.Loglevel, config.LogFormat)
	defer lf.Close()

	var accesslog io.Writer
	if config.AccessLog == "" {
		accesslog = os.Stdout
	} else {
		f, err := os.OpenFile(config.AccessLog, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Panicf("cannot open file %s: %v", config.AccessLog, err)
			return
		}
		defer f.Close()
		accesslog = f
	}

	mapping := map[string]string{}
	for _, val := range config.Indexer.FileMap {
		mapping[strings.ToLower(val.Alias)] = val.Folder
	}
	fm := indexer.NewFileMapper(mapping)

	sftp, err := indexer.NewSFTP(config.SFTP.PrivateKey, config.SFTP.Password, config.SFTP.Knownhosts, log)
	if err != nil {
		log.Panicf("cannot initialize sftp client: %v", err)
		return
	}

	mimeRelevance := map[int]indexer.MimeWeightString{}
	for key, val := range config.Indexer.MimeRelevance {
		keyInt, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			log.Panicf("cannot convert mimeRelevance %s to string", key)
			return
		}
		mimeRelevance[int(keyInt)] = indexer.MimeWeightString{
			Regexp: val.Regexp,
			Weight: val.Weight,
		}
	}
	errorTpl, err := template.ParseFiles(config.ErrorTemplate)
	if err != nil {
		log.Panicf("cannot parse error template %s: %v", config.ErrorTemplate, err)
		return
	}

	srv, err := indexer.NewServer(
		config.Indexer.HeaderTimeout.Duration,
		config.Indexer.HeaderSize,
		config.Indexer.DownloadMime,
		config.Indexer.MaxDownloadSize,
		mimeRelevance,
		config.JwtKey,
		config.JwtAlg,
		config.InsecureCert,
		log,
		accesslog,
		errorTpl,
		config.Indexer.TempDir,
		fm,
		sftp,
	)
	if err != nil {
		log.Panicf("cannot initialize server: %v", err)
		return
	}

	ad := indexer.NewActionDispatcher(mimeRelevance)

	var nsrldb *badger.DB
	if config.Indexer.NSRL.Enabled {
		stat2, err := os.Stat(config.Indexer.NSRL.Badger)
		if err != nil {
			fmt.Printf("cannot stat badger folder %s: %v\n", config.Indexer.NSRL.Badger, err)
			return
		}
		if !stat2.IsDir() {
			fmt.Printf("%s is not a directory\n", config.Indexer.NSRL.Badger)
			return
		}

		bconfig := badger.DefaultOptions(config.Indexer.NSRL.Badger)
		bconfig.ReadOnly = true
		nsrldb, err = badger.Open(bconfig)
		if err != nil {
			log.Panicf("cannot open badger database in %s: %v\n", config.Indexer.NSRL.Badger, err)
			return
		}
		//log.Infof("nsrl max batch count: %v", nsrldb.MaxBatchCount())
		defer nsrldb.Close()
		var keyCount uint32
		for _, tbl := range nsrldb.Tables() {
			keyCount += tbl.KeyCount
		}
		log.Infof("NSRL-Table: %v keys", keyCount)
		indexer.NewActionNSRL("nsrl", nsrldb, srv, ad)
		//return
	}

	if config.Indexer.Siegfried.Enabled {
		var signatureData []byte
		if config.Indexer.Siegfried.SignatureFile == "internal:default" {
			signatureData = datasiegfried.DefaultSig
		} else {
			if _, err := os.Stat(config.Indexer.Siegfried.SignatureFile); err != nil {
				log.Panicf("siegfried signature file at %s not found. Please use 'sf -update' to download it: %v", config.Indexer.Siegfried.SignatureFile, err)
			}
			signatureData, err = os.ReadFile(config.Indexer.Siegfried.SignatureFile)
			if err != nil {
				log.Panicf("cannot read signature file at %s: %v", config.Indexer.Siegfried.SignatureFile, err)
			}
		}
		indexer.NewActionSiegfried("siegfried", signatureData, config.Indexer.Siegfried.MimeMap, config.Indexer.Siegfried.TypeMap, srv, ad)
		//srv.AddActions(sf)
	}

	if config.Indexer.FFMPEG.Enabled {
		var ffmpegmime []indexer.FFMPEGMime
		for _, val := range config.Indexer.FFMPEG.Mime {
			ffmpegmime = append(ffmpegmime, indexer.FFMPEGMime{
				Video:  val.Video,
				Audio:  val.Audio,
				Format: val.Format,
				Mime:   val.Mime,
			})
		}
		indexer.NewActionFFProbe("ffprobe", config.Indexer.FFMPEG.FFProbe, config.Indexer.FFMPEG.Wsl, config.Indexer.FFMPEG.Timeout.Duration, config.Indexer.FFMPEG.Online, ffmpegmime, srv, ad)
	}

	if config.Indexer.ImageMagick.Enabled {
		indexer.NewActionIdentify("identify", config.Indexer.ImageMagick.Identify, config.Indexer.ImageMagick.Convert, config.Indexer.ImageMagick.Wsl, config.Indexer.ImageMagick.Timeout.Duration, config.Indexer.ImageMagick.Online, srv, ad)
		indexer.NewActionIdentifyV2("identify2", config.Indexer.ImageMagick.Identify, config.Indexer.ImageMagick.Convert, config.Indexer.ImageMagick.Wsl, config.Indexer.ImageMagick.Timeout.Duration, config.Indexer.ImageMagick.Online, srv, ad)
	}

	if config.Indexer.Tika.Enabled {
		indexer.NewActionTika("tika", config.Indexer.Tika.AddressMeta, config.Indexer.Tika.Timeout.Duration, config.Indexer.Tika.RegexpMimeMeta, config.Indexer.Tika.RegexpMimeMetaNot, "", config.Indexer.Tika.Online, srv, ad)
		//srv.AddActions(tika)
	}

	if config.Indexer.Clamav.Enabled {
		indexer.NewActionClamAV(
			config.Indexer.Clamav.ClamScan,
			config.Indexer.Clamav.Wsl,
			config.Indexer.Clamav.Timeout.Duration,
			srv,
			ad)
	}

	for _, eaconfig := range config.Indexer.External {
		var caps uint
		for _, c := range eaconfig.ActionCapabilities {
			caps |= uint(c)
		}
		indexer.NewActionExternal(eaconfig.Name, eaconfig.Address, indexer.ActionCapability(caps), eaconfig.CallType, eaconfig.Mimetype, srv, ad)
		//srv.AddActions(ea)
	}

	for _, a := range ad.GetActions() {
		srv.AddActions(a)
	}

	go func() {
		if err := srv.ListenAndServe(config.Addr, config.CertPEM, config.KeyPEM); err != nil {
			log.Errorf("server died: %v", err)
		}
	}()

	end := make(chan bool, 1)

	// process waiting for interrupt signal (TERM or KILL)
	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)

		signal.Notify(sigint, syscall.SIGTERM)
		signal.Notify(sigint, syscall.SIGKILL)

		<-sigint

		// We received an interrupt signal, shut down.
		log.Infof("shutdown requested")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.Shutdown(ctx)

		end <- true
	}()

	<-end
	log.Info("server stopped")
}

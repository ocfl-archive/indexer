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
	"github.com/BurntSushi/toml"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
	"log"
	"os"
)

type SFTP struct {
	Knownhosts string
	Password   string
	PrivateKey []string
}

type Config struct {
	ErrorTemplate string
	Logfile       string
	Loglevel      string
	LogFormat     string
	AccessLog     string
	CertPEM       string
	KeyPEM        string
	Addr          string
	JwtKey        string
	InsecureCert  bool
	JwtAlg        []string
	SFTP          SFTP
	Indexer       *indexer.IndexerConfig
}

func LoadConfig(fp string) *Config {
	var conf = &Config{
		LogFormat:    `%{time:2006-01-02T15:04:05.000} %{shortpkg}::%{longfunc} [%{shortfile}] > %{level:.5s} - %{message}`,
		InsecureCert: false,
		Indexer:      indexer.GetDefaultConfig(),
	}

	if _, err := toml.DecodeFile(fp, conf); err != nil {
		log.Fatalln("Error on loading config: ", err)
	}
	pwd := os.Getenv("SFTP_PASSWORD")
	if pwd != "" {
		conf.SFTP.Password = pwd
	}

	return conf
}

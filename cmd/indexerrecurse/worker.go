package main

import (
	"emperror.dev/errors"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/je4/utils/v2/pkg/checksum"
	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
	"github.com/ocfl-archive/indexer/v3/pkg/util"
	"golang.org/x/exp/slices"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var waiter sync.WaitGroup

var serialWriterLock sync.Mutex

func jsonlWriteLine(w io.Writer, fData *fileData) error {
	d, err := json.Marshal(fData)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal data")
	}
	serialWriterLock.Lock()
	defer serialWriterLock.Unlock()
	if _, err := w.Write(append(d, []byte("\n")...)); err != nil {
		return errors.Wrapf(err, "cannot write to output")
	}
	return nil
}

var csvWriterlock sync.Mutex

func csvWriteLine(csvWriter *csv.Writer, fData *fileData) error {
	dupStr := "no"
	if fData.Duplicate {
		dupStr = "yes"
	}
	csvWriterlock.Lock()
	defer csvWriterlock.Unlock()
	return errors.WithStack(csvWriter.Write([]string{fData.Path, fData.Folder, fData.Basename, fmt.Sprintf("%v", fData.Indexer.Size), time.Unix(fData.LastMod, 0).Format(time.DateTime), dupStr, fData.Indexer.Mimetype, fData.Indexer.Pronom, fData.Indexer.Type, fData.Indexer.Subtype, fData.Indexer.Checksum[string(checksum.DigestSHA512)], fmt.Sprintf("%v", fData.Indexer.Width), fmt.Sprintf("%v", fData.Indexer.Height), fmt.Sprintf("%v", fData.Indexer.Duration)}))
}

func writeConsole(fData *fileData, id uint, basePath string, cached bool) {
	var cachedStr string
	if cached {
		cachedStr = " [cached]"
	}
	basePath = strings.TrimSuffix(basePath, "/")
	p := path.Join(basePath, fData.Path)
	fmt.Printf("#%03d:%s %s\n           [%s] - %s\n", id, cachedStr, p, fData.Indexer.Mimetype, fData.Indexer.Checksum[string(checksum.DigestSHA512)])
	if fData.Indexer.Type == "image" && fData.Indexer.Width > 0 {
		fmt.Printf("#           image: %vx%v\n", fData.Indexer.Width, fData.Indexer.Height)
	}
}

var hashes = []string{}
var hashLock sync.Mutex

func isDup(t string) bool {
	hashLock.Lock()
	defer hashLock.Unlock()
	i, found := slices.BinarySearch(hashes, t) // find slot
	if found {
		return true // already in slice
	}
	// Make room for new value and add it
	hashes = append(hashes, *new(string))
	copy(hashes[i+1:], hashes[i:])
	hashes[i] = t
	return false
}

type fileData struct {
	Path      string            `json:"path"`
	Folder    string            `json:"folder"`
	Basename  string            `json:"basename"`
	Size      int64             `json:"size"`
	Duplicate bool              `json:"duplicate"`
	LastMod   int64             `json:"lastmod"`
	Indexer   *indexer.ResultV2 `json:"indexer"`
	LastSeen  int64             `json:"lastseen"`
}

func worker(id uint, fsys fs.FS, idx *util.Indexer, logger zLogger.ZLogger, jobs <-chan string, results chan<- string, jsonlWriter io.Writer, csvWriter *csv.Writer, badgerDB *badger.DB, startTime int64) {
	for path := range jobs {
		fmt.Println("worker", id, "processing job", path)
		finfo, err := fs.Stat(fsys, path)
		if err != nil {
			logger.Error().Err(err).Msgf("cannot stat (%s)%s", fsys, path)
			waiter.Done()
			return
		}
		if finfo.IsDir() {
			logger.Error().Err(err).Msgf("cannot index (%s)%s: is a directory", fsys, path)
			waiter.Done()
			return
		}

		var fData *fileData
		var fromCache bool
		if badgerDB != nil {
			err := badgerDB.View(func(txn *badger.Txn) error {
				key := []byte(path)
				if data, err := txn.Get(key); err != nil {
					if !errors.Is(err, badger.ErrKeyNotFound) {
						return errors.Wrapf(err, "cannot read from badger db")
					}
				} else {
					fData = &fileData{}
					data.Value(func(val []byte) error {
						if err := json.Unmarshal(val, fData); err != nil {
							return errors.Wrapf(err, "cannot unmarshal data")
						}
						fData.LastSeen = startTime
						if err := badgerDB.Update(func(txn *badger.Txn) error {
							key := []byte(path)
							value, err := json.Marshal(fData)
							if err != nil {
								return errors.Wrapf(err, "cannot marshal result")
							}
							if err := txn.Set(key, value); err != nil {
								return errors.Wrapf(err, "cannot write to badger db")
							}
							return nil
						}); err != nil {
							logger.Error().Err(err).Msgf("cannot write to badger db")
						}
						return nil
					})
				}
				return nil
			})
			if err != nil {
				logger.Error().Err(err).Msgf("cannot read from badger db")
			} else {
				fromCache = true
			}
		}

		if fData == nil {
			actions := []string{"siegfried", "xml"} // , "identify", "ffprobe", "tika"
			if *actionsFlag != "" {
				for _, a := range strings.Split(*actionsFlag, ",") {
					a = strings.ToLower(strings.TrimSpace(a))
					if a != "" {
						actions = append(actions, a)
					}
				}
			}
			slices.Sort(actions)
			actions = slices.Compact(actions)
			r, cs, err := idx.Index(fsys, path, "", actions, []checksum.DigestAlgorithm{checksum.DigestSHA512}, io.Discard, logger)
			if err != nil {
				logger.Error().Err(err).Msgf("cannot index (%s)%s", fsys, path)
				waiter.Done()
				return
			}
			if len(r.Checksum) == 0 {
				r.Checksum = make(map[string]string)
				for alg, c := range cs {
					r.Checksum[string(alg)] = c
				}
			}
			dup := r.Size > 0 && isDup(cs[checksum.DigestSHA512])
			fData = &fileData{
				path,
				filepath.Dir(path),
				filepath.Base(path),
				int64(r.Size),
				dup,
				finfo.ModTime().Unix(),
				r,
				startTime,
			}
		}

		basePath := fmt.Sprintf("%v", fsys)
		writeConsole(fData, id, basePath, fromCache)

		if badgerDB != nil {
			if err := badgerDB.Update(func(txn *badger.Txn) error {
				key := []byte(path)
				value, err := json.Marshal(fData)
				if err != nil {
					return errors.Wrapf(err, "cannot marshal result")
				}
				if err := txn.Set(key, value); err != nil {
					return errors.Wrapf(err, "cannot write to badger db")
				}
				return nil
			}); err != nil {
				logger.Error().Err(err).Msgf("cannot write to badger db")
			}
		}

		if jsonlWriter != nil {
			if err := jsonlWriteLine(jsonlWriter, fData); err != nil {
				logger.Error().Err(err).Msgf("cannot write to output")
			}
		}
		if csvWriter != nil {
			csvWriteLine(csvWriter, fData)
		}
		results <- path + " done"
		waiter.Done()
	}
}

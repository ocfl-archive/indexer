//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package util

import (
	"regexp"
)

var checkProgramList = map[string]checkProgramStruct{
	"magickconvert": {
		Name:   []string{"magick convert", "convert"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^Version: ImageMagick "),
	},
	"magickidentify": {
		Name:   []string{"magick identify", "identify"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^Version: ImageMagick "),
	},
	"ffprobe": {
		Name:   []string{"ffprobe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^ffprobe version "),
	},
	"ffmpeg": {
		Name:   []string{"ffmpeg"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^ffmpeg version "),
	},
}

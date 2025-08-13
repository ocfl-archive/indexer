//go:build windows

package util

import (
	"regexp"
)

var checkProgramList = map[string]checkProgramStruct{
	"magickconvert": {
		Name:   []string{"magick.exe convert", "convert.exe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^Version: ImageMagick "),
	},
	"magickidentify": {
		Name:   []string{"magick.exe identify", "identify.exe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^Version: ImageMagick "),
	},
	"ffprobe": {
		Name:   []string{"ffprobe.exe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^ffprobe version "),
	},
	"ffmpeg": {
		Name:   []string{"ffmpeg.exe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^ffmpeg version "),
	},
}

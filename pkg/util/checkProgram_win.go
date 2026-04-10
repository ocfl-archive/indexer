//go:build windows

package util

import (
	"regexp"
)

var checkProgramList = map[string]checkProgramStruct{
	CheckProgramMagickConvert: {
		Name:   []string{"magick.exe convert", "convert.exe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^Version: ImageMagick "),
	},
	CheckProgramMagickIdentify: {
		Name:   []string{"magick.exe identify", "identify.exe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^Version: ImageMagick "),
	},
	CheckProgramFFProbe: {
		Name:   []string{"ffprobe.exe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^ffprobe version "),
	},
	CheckProgramFFMpeg: {
		Name:   []string{"ffmpeg.exe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^ffmpeg version "),
	},
}

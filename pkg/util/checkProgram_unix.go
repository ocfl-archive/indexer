//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package util

import (
	"regexp"
)

var checkProgramList = map[string]checkProgramStruct{
	CheckProgramGhostscript: {
		Name:   []string{"gs"},
		Param:  []string{"-v"},
		Result: regexp.MustCompile(`^GPL Ghostscript`),
	},
	CheckProgramMagickConvert: {
		Name:   []string{"magick convert", "convert"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^Version: ImageMagick "),
	},
	CheckProgramMagickIdentify: {
		Name:   []string{"magick identify", "identify"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^Version: ImageMagick "),
	},
	CheckProgramFFProbe: {
		Name:   []string{"ffprobe"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^ffprobe version "),
	},
	CheckProgramFFMpeg: {
		Name:   []string{"ffmpeg"},
		Param:  []string{"-version"},
		Result: regexp.MustCompile("^ffmpeg version "),
	},
}

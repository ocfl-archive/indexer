package util

import (
	"fmt"
	"testing"
)

func TestCheckProgram_ImageMagickConvert_MagickExe(t *testing.T) {
	// Aufruf testen
	got, ok := checkProgram(CheckProgramMagickConvert, "")
	if !ok {
		fmt.Println("nicht gefunden")
	}
	fmt.Println(got)
}

func TestCheckProgram_FFProbe(t *testing.T) {
	// Aufruf testen
	got, ok := checkProgram(CheckProgramFFProbe, "")
	if !ok {
		fmt.Println("nicht gefunden")
	}
	fmt.Println(got)
}

func TestCheckProgram_Ffmpeg(t *testing.T) {
	// Aufruf testen
	got, ok := checkProgram(CheckProgramFFMpeg, "")
	if !ok {
		fmt.Println("nicht gefunden")
	}
	fmt.Println(got)
}

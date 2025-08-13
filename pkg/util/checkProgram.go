package util

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type checkProgramStruct struct {
	Name   []string
	Param  []string
	Result *regexp.Regexp
}

func doCheck(prg string, params []string, resultRegex *regexp.Regexp) (string, bool) {
	path := os.Getenv("path")
	for _, dir := range append([]string{""}, filepath.SplitList(path)...) {
		fullpath := filepath.Join(dir, prg)
		var outb, errb bytes.Buffer
		cmd := exec.Command(fullpath, params...)
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		if err := cmd.Run(); err != nil {
			continue
		}
		if resultRegex.Match(outb.Bytes()) {
			return dir, true
		}
	}
	return "", false
}

func checkProgram(command string) (string, bool) {
	pw, ok := checkProgramList[command]
	if !ok {
		return "", false
	}
	for _, name := range pw.Name {
		nameParts := strings.Split(name, " ")
		prg := nameParts[0]
		params := []string{}
		if len(nameParts) > 1 {
			params = append(params, nameParts[1:]...)
		}
		params = append(params, pw.Param...)
		if dir, ok := doCheck(prg, params, pw.Result); ok {
			ret := filepath.Join(dir, prg)
			if len(nameParts) > 1 {
				ret += " " + strings.Join(nameParts[1:], " ")
			}
			return ret, true
		}
	}
	return "", false
}

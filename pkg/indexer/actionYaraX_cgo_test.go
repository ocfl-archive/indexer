//go:build cgo

package indexer

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	yara_x "github.com/VirusTotal/yara-x/go"
	"github.com/stretchr/testify/require"
)

const sampleYaraRule = `
rule TestRule {
    meta:
        mime = "text/plain"
        pronom = "fmt/999"
        type = "document"
        subtype = "text"
    strings:
        $hello = "HELLO_YARAX"
    condition:
        $hello
}
`

func TestActionYaraXCgo_SourceCompilationAndScan(t *testing.T) {
	ad := NewActionDispatcher(nil)
	ay, err := NewActionYaraX(
		NameYaraX,
		[]byte(sampleYaraRule),
		false,
		ad,
		false,
		5*time.Second,
		100,
	)
	require.NoError(t, err)
	require.NotNil(t, ay)
	defer ay.Close()

	require.Equal(t, NameYaraX, ay.GetName())
	require.True(t, ay.CanHandle("text/plain", "test.txt"))
	require.Equal(t, uint(15), ay.GetWeight())
	require.Equal(t, ACTFILEHEAD|ACTSTREAM, ay.GetCaps())

	// Test Stream
	content := "prefix HELLO_YARAX suffix"
	res, err := ay.Stream(context.Background(), "text/plain", strings.NewReader(content), "test.txt")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Contains(t, res.Mimetypes, "text/plain")
	require.Contains(t, res.Pronoms, "fmt/999")
	require.Equal(t, "document", res.Type)
	require.Equal(t, "text", res.Subtype)

	matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
	require.True(t, ok)
	require.Len(t, matches, 1)
	require.Equal(t, "TestRule", matches[0].Rule)
	require.Len(t, matches[0].Patterns, 1)
	require.Equal(t, "$hello", matches[0].Patterns[0].Identifier)
	require.Len(t, matches[0].Patterns[0].Matches, 1)
	require.Equal(t, uint64(7), matches[0].Patterns[0].Matches[0].Offset)
	require.Equal(t, uint64(11), matches[0].Patterns[0].Matches[0].Length)

	// Test DoV2 (File)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sample.txt")
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	resFile, err := ay.DoV2(context.Background(), filePath)
	require.NoError(t, err)
	require.NotNil(t, resFile)
	require.Contains(t, resFile.Mimetypes, "text/plain")
	require.Contains(t, resFile.Pronoms, "fmt/999")
}

func TestActionYaraXCgo_CompiledRules(t *testing.T) {
	rules, err := yara_x.Compile(sampleYaraRule)
	require.NoError(t, err)
	defer rules.Destroy()

	var buf bytes.Buffer
	_, err = rules.WriteTo(&buf)
	require.NoError(t, err)

	ad := NewActionDispatcher(nil)
	ay, err := NewActionYaraX(
		NameYaraX,
		buf.Bytes(),
		true,
		ad,
		true,
		0,
		0,
	)
	require.NoError(t, err)
	require.NotNil(t, ay)
	defer ay.Close()

	res, err := ay.Stream(context.Background(), "text/plain", strings.NewReader("HELLO_YARAX 123"), "test.txt")
	require.NoError(t, err)
	require.Contains(t, res.Mimetypes, "text/plain")
}

func TestActionYaraXCgo_MultipleRulesFilesAndGlobs(t *testing.T) {
	tmpDir := t.TempDir()

	rule1 := `
rule RuleOne {
    meta:
        mime = "application/x-rule-one"
        pronom = "fmt/1001"
    strings:
        $str1 = "PATTERN_ONE"
    condition:
        $str1
}
`
	rule2 := `
rule RuleTwo {
    meta:
        mime = "application/x-rule-two"
        pronom = "fmt/1002"
    strings:
        $str2 = "PATTERN_TWO"
    condition:
        $str2
}
`
	rule1Path := filepath.Join(tmpDir, "rule1.yar")
	rule2Path := filepath.Join(tmpDir, "rule2.yar")
	require.NoError(t, os.WriteFile(rule1Path, []byte(rule1), 0644))
	require.NoError(t, os.WriteFile(rule2Path, []byte(rule2), 0644))

	// Test glob pattern
	globPattern := filepath.Join(tmpDir, "*.yar")
	ad := NewActionDispatcher(nil)
	ay, err := NewActionYaraXFromFiles(NameYaraX, []string{globPattern}, ad, false, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, ay)
	defer ay.Close()

	// Stream matching both rules
	content := "prefix PATTERN_ONE and PATTERN_TWO suffix"
	res, err := ay.Stream(context.Background(), "application/octet-stream", strings.NewReader(content), "sample.bin")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Contains(t, res.Mimetypes, "application/x-rule-one")
	require.Contains(t, res.Mimetypes, "application/x-rule-two")
	require.Contains(t, res.Pronoms, "fmt/1001")
	require.Contains(t, res.Pronoms, "fmt/1002")

	matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
	require.True(t, ok)
	require.Len(t, matches, 2)
}

func TestActionYaraXCgo_MixedSourceAndCompiledWithAutoDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a source .yar file
	sourceRule := `
rule SourceRule {
    meta:
        mime = "text/source-rule"
        pronom = "fmt/source"
    strings:
        $s = "SOURCE_MATCH"
    condition:
        $s
}
`
	sourcePath := filepath.Join(tmpDir, "source.yar")
	require.NoError(t, os.WriteFile(sourcePath, []byte(sourceRule), 0644))

	// 2. Compile a rule and save as .yrc file
	compiledRule := `
rule CompiledRule {
    meta:
        mime = "text/compiled-rule"
        pronom = "fmt/compiled"
    strings:
        $c = "COMPILED_MATCH"
    condition:
        $c
}
`
	cr, err := yara_x.Compile(compiledRule)
	require.NoError(t, err)
	defer cr.Destroy()

	compiledPath := filepath.Join(tmpDir, "compiled.yrc")
	f, err := os.Create(compiledPath)
	require.NoError(t, err)
	_, err = cr.WriteTo(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// 3. Initialize ActionYaraXFromFiles with both source (.yar) and compiled (.yrc)
	ad := NewActionDispatcher(nil)
	ay, err := NewActionYaraXFromFiles(NameYaraX, []string{sourcePath, compiledPath}, ad, false, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, ay)
	defer ay.Close()

	testFile := filepath.Join(tmpDir, "testfile.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("SOURCE_MATCH and COMPILED_MATCH"), 0644))

	res, err := ay.DoV2(context.Background(), testFile)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Contains(t, res.Mimetypes, "text/source-rule")
	require.Contains(t, res.Mimetypes, "text/compiled-rule")
	require.Contains(t, res.Pronoms, "fmt/source")
	require.Contains(t, res.Pronoms, "fmt/compiled")
}

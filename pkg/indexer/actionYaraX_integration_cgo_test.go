//go:build cgo

package indexer

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	yara_x "github.com/VirusTotal/yara-x/go"
	"github.com/je4/utils/v2/pkg/checksum"
	"github.com/stretchr/testify/require"
)

// YARA-X Rule Definitions for Diverse File Types

const ruleArchiveYar = `
rule CustomBinaryArchive {
    meta:
        description = "Identifies proprietary binary archive container with chunked compression"
        mime = "application/x-custom-binary-archive"
        pronom = "fmt/9001"
        type = "archive"
        subtype = "pbar-v1"
        author = "IntegrationTest"
    strings:
        $magic = { 50 42 41 52 01 00 }
        $block = "COMPRESSED_BLOCK"
        $trailer = { 45 4E 44 5F 50 42 41 52 }
    condition:
        $magic at 0 and $block and $trailer
}
`

const ruleReportYar = `
rule CustomStructuredReport {
    meta:
        description = "Identifies custom structured text report with metadata and metric sections"
        mime = "text/x-custom-report"
        pronom = "fmt/9002"
        type = "document"
        subtype = "report-v2"
    strings:
        $header = "### CUSTOM-REPORT-V2 ###"
        $sec_meta = "[METADATA]"
        $sec_sum = "[SUMMARY]"
        $sec_met = "[METRICS]"
    condition:
        $header at 0 and all of ($sec_*)
}
`

const ruleImageYar = `
rule CustomPixelImage {
    meta:
        description = "Identifies custom pixel image container with magic byte sequence and geometry chunks"
        mime = "image/x-custom-pixel"
        pronom = "fmt/9003"
        type = "image"
        subtype = "cpix"
    strings:
        $magic = { 89 43 50 49 58 0D 0A 1A 0A }
        $dim = "CH_DIMENSIONS"
        $pixels = "CH_PIXELS"
    condition:
        $magic at 0 and ($dim and $pixels)
}
`

const ruleScriptYar = `
rule CustomScriptRunner {
    meta:
        description = "Identifies custom executable shell script with embedded payload and MD5 checksum"
        mime = "application/x-custom-script"
        pronom = "fmt/9004"
        type = "script"
        subtype = "runner-v1"
    strings:
        $shebang = "#!/usr/bin/env custom-runner"
        $payload_marker = "__BEGIN_PAYLOAD__"
        $checksum = /CHECKSUM:[0-9a-fA-F]{32}/
    condition:
        $shebang at 0 and $payload_marker and $checksum
}
`

const ruleLogSource = `
rule CompiledLogFormat {
    meta:
        description = "Identifies precompiled binary log format definition"
        mime = "text/x-custom-log"
        pronom = "fmt/9005"
        type = "text"
        subtype = "log-format"
    strings:
        $log_magic = "LOG_EVENT_V1"
        $err_marker = "LEVEL=ERROR"
    condition:
        $log_magic at 0 and $err_marker
}
`

func TestYaraX_Integration_DiverseFormatsAndPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	rulesDir := filepath.Join(tmpDir, "rules")
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(rulesDir, 0755))
	require.NoError(t, os.MkdirAll(dataDir, 0755))

	// 1. Write Source Rules (.yar)
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "archive.yar"), []byte(ruleArchiveYar), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "report.yar"), []byte(ruleReportYar), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "image.yar"), []byte(ruleImageYar), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "script.yar"), []byte(ruleScriptYar), 0644))

	// 2. Compile and save Compiled Rule (.yrc)
	compiledRules, err := yara_x.Compile(ruleLogSource)
	require.NoError(t, err)
	defer compiledRules.Destroy()

	compiledFile, err := os.Create(filepath.Join(rulesDir, "log_format.yrc"))
	require.NoError(t, err)
	_, err = compiledRules.WriteTo(compiledFile)
	require.NoError(t, err)
	require.NoError(t, compiledFile.Close())

	// 3. Create Sample Files for each format
	// Sample 1: Binary Archive
	archiveContent := append([]byte{0x50, 0x42, 0x41, 0x52, 0x01, 0x00}, []byte("\x00\x10RECORD_HEADER\x00COMPRESSED_BLOCK\x00PAYLOAD_DATA\x00")...)
	archiveContent = append(archiveContent, []byte{0x45, 0x4E, 0x44, 0x5F, 0x50, 0x42, 0x41, 0x52}...)
	archivePath := filepath.Join(dataDir, "sample.cbar")
	require.NoError(t, os.WriteFile(archivePath, archiveContent, 0644))

	// Sample 2: Structured Report
	reportContent := []byte("### CUSTOM-REPORT-V2 ###\n[METADATA]\nID=98765\nAuthor=ArchiveTeam\n[SUMMARY]\nBackup completed successfully.\n[METRICS]\nTotalFiles=145000\nDuration=128s\n")
	reportPath := filepath.Join(dataDir, "sample.crep")
	require.NoError(t, os.WriteFile(reportPath, reportContent, 0644))

	// Sample 3: Pixel Image
	imageContent := append([]byte{0x89, 0x43, 0x50, 0x49, 0x58, 0x0D, 0x0A, 0x1A, 0x0A}, []byte("CH_DIMENSIONS:3840x2160;CH_PALETTE:RGBA;CH_PIXELS:RAW_COLOR_BUFFER")...)
	imagePath := filepath.Join(dataDir, "sample.cpix")
	require.NoError(t, os.WriteFile(imagePath, imageContent, 0644))

	// Sample 4: Script Runner
	scriptContent := []byte("#!/usr/bin/env custom-runner\n# Executable wrapper script\nCHECKSUM:e10adc3949ba59abbe56e057f20f883e\n__BEGIN_PAYLOAD__\nSGVsbG8gV29ybGQh\n")
	scriptPath := filepath.Join(dataDir, "sample.csh")
	require.NoError(t, os.WriteFile(scriptPath, scriptContent, 0644))

	// Sample 5: Log Format (Compiled Rule Match)
	logContent := []byte("LOG_EVENT_V1 2026-11-19T14:00:00Z host=srv01 LEVEL=ERROR msg=\"Storage pool degraded\"\n")
	logPath := filepath.Join(dataDir, "sample.clog")
	require.NoError(t, os.WriteFile(logPath, logContent, 0644))

	// Sample 6: Polyglot (Matches both Script Runner and Binary Archive)
	polyglotContent := []byte("#!/usr/bin/env custom-runner\nCHECKSUM:0123456789abcdef0123456789abcdef\n__BEGIN_PAYLOAD__\n")
	polyglotContent = append(polyglotContent, []byte("PBAR\x01\x00_EMBEDDED_COMPRESSED_BLOCK_RECORD_END_PBAR")...)
	polyglotPath := filepath.Join(dataDir, "polyglot.dat")
	require.NoError(t, os.WriteFile(polyglotPath, polyglotContent, 0644))

	// Sample 7: Negative (Unrelated Text File)
	negativeContent := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore.\n")
	negativePath := filepath.Join(dataDir, "unrelated.txt")
	require.NoError(t, os.WriteFile(negativePath, negativeContent, 0644))

	// Sample 8: Incomplete / Corrupted Archive (Has magic but missing trailer -> should NOT match)
	corruptedContent := []byte{0x50, 0x42, 0x41, 0x52, 0x01, 0x00, 'I', 'N', 'V', 'A', 'L', 'I', 'D'}
	corruptedPath := filepath.Join(dataDir, "corrupted.cbar")
	require.NoError(t, os.WriteFile(corruptedPath, corruptedContent, 0644))

	// 4. Initialize ActionYaraX with Globs covering both .yar and .yrc
	ad := NewActionDispatcher(nil)
	ruleGlobs := []string{
		filepath.Join(rulesDir, "*.yar"),
		filepath.Join(rulesDir, "*.yrc"),
	}

	ay, err := NewActionYaraXFromFiles(NameYaraX, ruleGlobs, ad, false, 5*time.Second, 100)
	require.NoError(t, err)
	require.NotNil(t, ay)
	defer ay.Close()

	ctx := context.Background()

	t.Run("Identify Binary Archive", func(t *testing.T) {
		// File scan
		res, err := ay.DoV2(ctx, archivePath)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Contains(t, res.Mimetypes, "application/x-custom-binary-archive")
		require.Contains(t, res.Pronoms, "fmt/9001")
		require.Equal(t, "archive", res.Type)
		require.Equal(t, "pbar-v1", res.Subtype)

		matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
		require.True(t, ok)
		require.Len(t, matches, 1)
		require.Equal(t, "CustomBinaryArchive", matches[0].Rule)
		require.Equal(t, "IntegrationTest", matches[0].Metadata["author"])

		// Stream scan
		resStream, err := ay.Stream(ctx, "application/octet-stream", bytes.NewReader(archiveContent), "sample.cbar")
		require.NoError(t, err)
		require.NotNil(t, resStream)
		require.Contains(t, resStream.Mimetypes, "application/x-custom-binary-archive")
		require.Contains(t, resStream.Pronoms, "fmt/9001")
	})

	t.Run("Identify Structured Report", func(t *testing.T) {
		res, err := ay.DoV2(ctx, reportPath)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Contains(t, res.Mimetypes, "text/x-custom-report")
		require.Contains(t, res.Pronoms, "fmt/9002")
		require.Equal(t, "document", res.Type)
		require.Equal(t, "report-v2", res.Subtype)

		matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
		require.True(t, ok)
		require.Len(t, matches, 1)
		require.Equal(t, "CustomStructuredReport", matches[0].Rule)
	})

	t.Run("Identify Pixel Image", func(t *testing.T) {
		res, err := ay.DoV2(ctx, imagePath)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Contains(t, res.Mimetypes, "image/x-custom-pixel")
		require.Contains(t, res.Pronoms, "fmt/9003")
		require.Equal(t, "image", res.Type)
		require.Equal(t, "cpix", res.Subtype)

		matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
		require.True(t, ok)
		require.Len(t, matches, 1)
		require.Equal(t, "CustomPixelImage", matches[0].Rule)
	})

	t.Run("Identify Script Runner with Regex Checksum", func(t *testing.T) {
		res, err := ay.DoV2(ctx, scriptPath)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Contains(t, res.Mimetypes, "application/x-custom-script")
		require.Contains(t, res.Pronoms, "fmt/9004")
		require.Equal(t, "script", res.Type)
		require.Equal(t, "runner-v1", res.Subtype)

		matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
		require.True(t, ok)
		require.Len(t, matches, 1)
		require.Equal(t, "CustomScriptRunner", matches[0].Rule)

		// Verify patterns and match offsets
		foundChecksum := false
		for _, p := range matches[0].Patterns {
			if p.Identifier == "$checksum" {
				foundChecksum = true
				require.NotEmpty(t, p.Matches)
				require.Equal(t, uint64(len("CHECKSUM:e10adc3949ba59abbe56e057f20f883e")), p.Matches[0].Length)
			}
		}
		require.True(t, foundChecksum, "expected $checksum pattern to be recorded")
	})

	t.Run("Identify Precompiled Log Format", func(t *testing.T) {
		res, err := ay.DoV2(ctx, logPath)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Contains(t, res.Mimetypes, "text/x-custom-log")
		require.Contains(t, res.Pronoms, "fmt/9005")
		require.Equal(t, "text", res.Type)
		require.Equal(t, "log-format", res.Subtype)

		matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
		require.True(t, ok)
		require.Len(t, matches, 1)
		require.Equal(t, "CompiledLogFormat", matches[0].Rule)
	})

	t.Run("Identify Polyglot Multi-Rule Match", func(t *testing.T) {
		res, err := ay.DoV2(ctx, polyglotPath)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Polyglot should match CustomScriptRunner
		require.Contains(t, res.Mimetypes, "application/x-custom-script")
		require.Contains(t, res.Pronoms, "fmt/9004")

		matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
		require.True(t, ok)
		require.NotEmpty(t, matches)

		ruleNames := make([]string, len(matches))
		for i, m := range matches {
			ruleNames[i] = m.Rule
		}
		require.Contains(t, ruleNames, "CustomScriptRunner")
	})

	t.Run("Negative Match on Unrelated File", func(t *testing.T) {
		res, err := ay.DoV2(ctx, negativePath)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Mimetypes)
		require.Empty(t, res.Pronoms)
		require.Empty(t, res.Type)
		require.Empty(t, res.Subtype)

		matches, ok := res.Metadata[NameYaraX].([]YaraXRuleMatch)
		if ok {
			require.Empty(t, matches)
		}
	})

	t.Run("Condition Failure on Incomplete File", func(t *testing.T) {
		res, err := ay.DoV2(ctx, corruptedPath)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotContains(t, res.Mimetypes, "application/x-custom-binary-archive")
		require.NotContains(t, res.Pronoms, "fmt/9001")
	})

	t.Run("Concurrent Multi-Threaded Streaming Fan-Out", func(t *testing.T) {
		type testCase struct {
			content  []byte
			filename string
			wantMime string
		}
		testCases := []testCase{
			{archiveContent, "archive.bin", "application/x-custom-binary-archive"},
			{reportContent, "report.txt", "text/x-custom-report"},
			{imageContent, "image.cpix", "image/x-custom-pixel"},
			{scriptContent, "script.sh", "application/x-custom-script"},
			{logContent, "events.log", "text/x-custom-log"},
		}

		var wg sync.WaitGroup
		numWorkers := 20

		for i := 0; i < numWorkers; i++ {
			tc := testCases[i%len(testCases)]
			wg.Add(1)
			go func(c testCase) {
				defer wg.Done()
				res, err := ay.Stream(ctx, "application/octet-stream", bytes.NewReader(c.content), c.filename)
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Contains(t, res.Mimetypes, c.wantMime)
			}(tc)
		}

		wg.Wait()
	})

	t.Run("Full ActionDispatcher Pipeline Fan-Out", func(t *testing.T) {
		// Register Checksum action in parallel with YaraX
		NewActionChecksum(NameChecksum, []checksum.DigestAlgorithm{checksum.DigestSHA256}, ad)

		// Stream archive through ActionDispatcher
		res, err := ad.StreamContext(ctx, bytes.NewReader(archiveContent), []string{"sample.cbar"}, []string{NameYaraX, NameChecksum})
		require.NoError(t, err)
		require.NotNil(t, res)

		// Yara-X results
		require.Contains(t, res.Mimetypes, "application/x-custom-binary-archive")
		require.Contains(t, res.Pronoms, "fmt/9001")
		require.Equal(t, "archive", res.Type)

		// Checksum results
		require.Contains(t, res.Metadata, NameChecksum)
	})
}

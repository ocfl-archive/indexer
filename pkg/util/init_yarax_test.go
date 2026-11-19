package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/je4/utils/v2/pkg/zLogger"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestInitIndexer_YaraX_Disabled(t *testing.T) {
	l := zerolog.New(os.Stderr)
	var logger zLogger.ZLogger = &l
	conf := &indexer.IndexerConfig{
		YaraX: indexer.ConfigYaraX{
			Enabled: false,
		},
	}

	idx, actions, closer, err := InitIndexer(conf, logger)
	require.NoError(t, err)
	require.NotNil(t, idx)
	if closer != nil {
		defer closer.Close()
	}
	require.NotContains(t, actions, indexer.NameYaraX)
}

func TestInitIndexer_YaraX_Config(t *testing.T) {
	tmpDir := t.TempDir()
	ruleFile := filepath.Join(tmpDir, "test.yar")
	ruleContent := `
rule DummyRule {
    meta:
        mime = "application/x-dummy"
    condition:
        true
}
`
	require.NoError(t, os.WriteFile(ruleFile, []byte(ruleContent), 0644))

	l := zerolog.New(os.Stderr)
	var logger zLogger.ZLogger = &l
	conf := &indexer.IndexerConfig{
		YaraX: indexer.ConfigYaraX{
			Enabled:    true,
			RulesFiles: []string{ruleFile},
		},
	}

	idx, actions, closer, err := InitIndexer(conf, logger)
	if closer != nil {
		defer closer.Close()
	}

	// On non-CGO platforms/builds, YaraX will fail with ErrYaraXCgoRequired
	// On CGO platforms, it will succeed and register NameYaraX in actions
	if err != nil {
		require.ErrorIs(t, err, indexer.ErrYaraXCgoRequired)
	} else {
		require.NotNil(t, idx)
		require.Contains(t, actions, indexer.NameYaraX)
	}
}

//go:build !cgo

package indexer

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActionYaraXNoCgo(t *testing.T) {
	ay, err := NewActionYaraX("test_yarax", []byte("rule dummy { condition: true }"), false, nil, false, 0, 0)
	require.ErrorIs(t, err, ErrYaraXCgoRequired)
	require.Nil(t, ay)

	ayFiles, err := NewActionYaraXFromFiles("test_yarax", []string{"rule.yar"}, nil, false, 0, 0)
	require.ErrorIs(t, err, ErrYaraXCgoRequired)
	require.Nil(t, ayFiles)

	ayRule, err := NewActionYaraXWithRule("test_yarax", nil, nil, false, 0, 0)
	require.ErrorIs(t, err, ErrYaraXCgoRequired)
	require.Nil(t, ayRule)

	ayRules, err := NewActionYaraXWithRules("test_yarax", nil, nil, false, 0, 0)
	require.ErrorIs(t, err, ErrYaraXCgoRequired)
	require.Nil(t, ayRules)

	stub := &ActionYaraX{name: "stub"}
	require.Equal(t, "stub", stub.GetName())
	require.False(t, stub.CanHandle("text/plain", "test.txt"))
	require.Equal(t, uint(15), stub.GetWeight())
	require.Equal(t, ACTFILEHEAD|ACTSTREAM, stub.GetCaps())
	require.NoError(t, stub.Close())

	res, err := stub.DoV2(context.Background(), "test.txt")
	require.ErrorIs(t, err, ErrYaraXCgoRequired)
	require.Nil(t, res)

	resStream, err := stub.Stream(context.Background(), "text/plain", strings.NewReader("data"), "test.txt")
	require.ErrorIs(t, err, ErrYaraXCgoRequired)
	require.Nil(t, resStream)
}

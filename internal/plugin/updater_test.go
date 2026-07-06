// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdaterReplacesBareVersion(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "name: demo\nversion: 1.2.3\ndescription: sample\n")
	result, err := NewUpdater().Update(file, "v1.3.0", Options{})
	require.NoError(t, err)
	require.Equal(t, "1.3.0", result.Version)
	require.Equal(t, "version: 1.3.0", result.VersionLine)
	require.Empty(t, result.Warnings)
	require.Equal(t, "name: demo\nversion: 1.3.0\ndescription: sample\n", readFile(t, file))
}

func TestUpdaterPreservesCommentsAndNestedVersions(t *testing.T) {
	t.Parallel()

	content := "name: demo\nversion: \"1.2.3+7\"  # keep me\ndependencies:\n  version: 9.9.9\n# tail\n"
	file := writePubspec(t, content)
	result, err := NewUpdater().Update(file, "1.4.0", Options{BuildStrategy: BuildStrategyIncrement})
	require.NoError(t, err)
	require.Equal(t, "1.4.0+8", result.Version)
	require.Equal(t, "name: demo\nversion: \"1.4.0+8\"  # keep me\ndependencies:\n  version: 9.9.9\n# tail\n", readFile(t, file))
}

func TestUpdaterIncrementStartsAtOneWhenNoBuildSuffixExists(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.2.3\n")
	result, err := NewUpdater().Prepare(file, "1.3.0", Options{BuildStrategy: BuildStrategyIncrement})
	require.NoError(t, err)
	require.Equal(t, "1.3.0+1", result.Version)
	require.Equal(t, "version: 1.3.0+1", result.VersionLine)
	require.Equal(t, "version: 1.2.3\n", readFile(t, file))
}

func TestUpdaterReleaseCountUsesSemrelReleaseCount(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.2.3+4\n")
	result, err := NewUpdater().Prepare(file, "1.5.0", Options{BuildStrategy: BuildStrategyReleaseCount, ReleaseCount: "12"})
	require.NoError(t, err)
	require.Equal(t, "1.5.0+12", result.Version)
	require.Empty(t, result.Warnings)
}

func TestUpdaterReleaseCountUsesSemrelBuildNumberFallback(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.2.3+4\n")
	result, err := NewUpdater().Prepare(file, "1.5.0", Options{BuildStrategy: BuildStrategyReleaseCount, BuildNumber: "99"})
	require.NoError(t, err)
	require.Equal(t, "1.5.0+99", result.Version)
}

func TestUpdaterReleaseCountFallsBackToExistingBuildPlusOne(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.2.3+4\n")
	result, err := NewUpdater().Prepare(file, "1.5.0", Options{BuildStrategy: BuildStrategyReleaseCount})
	require.NoError(t, err)
	require.Equal(t, "1.5.0+5", result.Version)
	require.Len(t, result.Warnings, 1)
	require.Contains(t, result.Warnings[0], "falling back")
}

func TestUpdaterExplicitBuildNumberOverridesStrategy(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.2.3+4\n")
	result, err := NewUpdater().Prepare(file, "1.5.0", Options{BuildStrategy: BuildStrategyNone, ExplicitBuildNumber: "42", ReleaseCount: "9", BuildNumber: "10"})
	require.NoError(t, err)
	require.Equal(t, "1.5.0+42", result.Version)
}

func TestUpdaterMalformedBuildNumberWarnsAndStartsFresh(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.2.3+abc\n")
	result, err := NewUpdater().Prepare(file, "1.5.0", Options{BuildStrategy: BuildStrategyIncrement})
	require.NoError(t, err)
	require.Equal(t, "1.5.0+1", result.Version)
	require.Len(t, result.Warnings, 1)
	require.Contains(t, result.Warnings[0], "malformed")
}

func TestUpdaterMissingFile(t *testing.T) {
	t.Parallel()

	_, err := NewUpdater().Prepare(filepath.Join(t.TempDir(), "pubspec.yaml"), "1.0.0", Options{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "read")
}

func TestUpdaterMissingVersionField(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "name: demo\ndependencies:\n  flutter:\n    sdk: flutter\n")
	_, err := NewUpdater().Prepare(file, "1.0.0", Options{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "top-level version field not found")
}

func writePubspec(t *testing.T, content string) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "pubspec.yaml")
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
	return file
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

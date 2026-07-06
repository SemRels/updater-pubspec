// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	plugin "github.com/SemRels/updater-pubspec/internal/plugin"
	"github.com/stretchr/testify/require"
)

func TestRunUpdatesPubspec(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "name: demo\nversion: 1.0.0\n")
	env := map[string]string{"SEMREL_VERSION": "v1.1.0", "SEMREL_PLUGIN_PUBSPEC_PATH": file}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterStub{})
	require.Equal(t, 0, code, stderr.String())
	require.Contains(t, stdout.String(), "updated "+file+" to 1.1.0")
	require.Equal(t, "name: demo\nversion: 1.1.0\n", readPubspec(t, file))
	require.Equal(t, "plugin_schema_version=1\n", stderr.String())
}

func TestRunDryRunPrintsComputedVersionLine(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.0.0+7\n")
	env := map[string]string{
		"SEMREL_VERSION":                       "1.1.0",
		"SEMREL_DRY_RUN":                       "true",
		"SEMREL_PLUGIN_PUBSPEC_PATH":           file,
		"SEMREL_PLUGIN_PUBSPEC_BUILD_STRATEGY": "increment",
	}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterStub{})
	require.Equal(t, 0, code, stderr.String())
	require.Contains(t, stdout.String(), "[dry-run] version: 1.1.0+8")
	require.Equal(t, "version: 1.0.0+7\n", readPubspec(t, file))
}

func TestRunReleaseCountFallbackWarns(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.0.0+7\n")
	env := map[string]string{
		"SEMREL_VERSION":                       "1.1.0",
		"SEMREL_PLUGIN_PUBSPEC_PATH":           file,
		"SEMREL_PLUGIN_PUBSPEC_BUILD_STRATEGY": "release-count",
	}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterStub{})
	require.Equal(t, 0, code, stderr.String())
	require.Contains(t, stderr.String(), "warning: release-count strategy requested")
	require.Contains(t, stdout.String(), "updated")
	require.Equal(t, "version: 1.1.0+8\n", readPubspec(t, file))
}

func TestRunUsesSemrelNextVersion(t *testing.T) {
	t.Parallel()

	file := writePubspec(t, "version: 1.0.0\n")
	env := map[string]string{"SEMREL_NEXT_VERSION": "v1.2.0", "SEMREL_PLUGIN_PUBSPEC_PATH": file}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterStub{})
	require.Equal(t, 0, code, stderr.String())
	require.Equal(t, "version: 1.2.0\n", readPubspec(t, file))
}

func TestRunRequiresVersion(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, func(string) string { return "" }, updaterStub{})
	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "SEMREL_VERSION is required")
}

func TestRunReportsErrors(t *testing.T) {
	t.Parallel()

	env := map[string]string{"SEMREL_VERSION": "1.0.0", "SEMREL_PLUGIN_PUBSPEC_PATH": filepath.Join(t.TempDir(), "missing.yaml")}
	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterStub{})
	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "updater-pubspec: read")
}

type updaterStub struct{}

func (updaterStub) Prepare(path, version string, opts plugin.Options) (plugin.Result, error) {
	return plugin.NewUpdater().Prepare(path, version, opts)
}

func (updaterStub) Update(path, version string, opts plugin.Options) (plugin.Result, error) {
	return plugin.NewUpdater().Update(path, version, opts)
}

func writePubspec(t *testing.T, content string) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "pubspec.yaml")
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
	return file
}

func readPubspec(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

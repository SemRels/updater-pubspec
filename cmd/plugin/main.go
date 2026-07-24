// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	plugin "github.com/SemRels/updater-pubspec/internal/plugin"
)

const pluginSchemaVersion = 1

type updater interface {
	Prepare(path, version string, opts plugin.Options) (plugin.Result, error)
	Update(path, version string, opts plugin.Options) (plugin.Result, error)
}

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Getenv, plugin.NewUpdater()))
}

func run(stdout, stderr io.Writer, getenv func(string) string, updater updater) int {
	_, _ = fmt.Fprintf(stderr, "plugin_schema_version=%d\n", pluginSchemaVersion)

	version := getenv("SEMREL_VERSION")
	if version == "" {
		version = getenv("SEMREL_NEXT_VERSION")
	}
	if version == "" {
		_, _ = fmt.Fprintln(stderr, "updater-pubspec: SEMREL_VERSION is required")
		return 1
	}

	file := getenv("SEMREL_PLUGIN_PUBSPEC_PATH")
	if file == "" {
		file = "./pubspec.yaml"
	}

	opts := plugin.Options{
		BuildStrategy:       getenv("SEMREL_PLUGIN_PUBSPEC_BUILD_STRATEGY"),
		ExplicitBuildNumber: getenv("SEMREL_PLUGIN_PUBSPEC_BUILD_NUMBER"),
		ReleaseCount:        getenv("SEMREL_RELEASE_COUNT"),
		BuildNumber:         getenv("SEMREL_BUILD_NUMBER"),
	}

	if strings.EqualFold(getenv("SEMREL_DRY_RUN"), "true") {
		result, err := updater.Prepare(file, version, opts)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "updater-pubspec: %v\n", err)
			return 1
		}
		emitWarnings(stderr, result.Warnings)
		_, _ = fmt.Fprintf(stdout, "updater-pubspec: [dry-run] %s\n", result.VersionLine)
		return 0
	}

	result, err := updater.Update(file, version, opts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "updater-pubspec: %v\n", err)
		return 1
	}
	emitWarnings(stderr, result.Warnings)
	_, _ = fmt.Fprintf(stdout, "updater-pubspec: updated %s to %s\n", file, result.Version)
	return 0
}

func emitWarnings(stderr io.Writer, warnings []string) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(stderr, "updater-pubspec: warning: %s\n", warning)
	}
}

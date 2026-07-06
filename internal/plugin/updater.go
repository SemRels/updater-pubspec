// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

// Package plugin updates Dart and Flutter pubspec.yaml files in-place.
package plugin

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	BuildStrategyNone         = "none"
	BuildStrategyIncrement    = "increment"
	BuildStrategyReleaseCount = "release-count"
)

// Options controls build suffix handling.
type Options struct {
	BuildStrategy       string
	ExplicitBuildNumber string
	ReleaseCount        string
	BuildNumber         string
}

// Result describes the computed update.
type Result struct {
	Version     string
	VersionLine string
	Content     string
	Warnings    []string
}

// Updater updates pubspec.yaml version fields.
type Updater struct{}

// NewUpdater creates an updater.
func NewUpdater() *Updater {
	return &Updater{}
}

// Prepare computes the updated file content without writing it.
func (u *Updater) Prepare(path, version string, opts Options) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", path, err)
	}
	return prepareContent(string(data), path, normalizeVersion(version), opts)
}

// Update rewrites the version field in pubspec.yaml.
func (u *Updater) Update(path, version string, opts Options) (Result, error) {
	result, err := u.Prepare(path, version, opts)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(path, []byte(result.Content), 0o644); err != nil {
		return Result{}, fmt.Errorf("write %s: %w", path, err)
	}
	return result, nil
}

func prepareContent(content, path, version string, opts Options) (Result, error) {
	strategy := normalizeStrategy(opts.BuildStrategy)
	scanner := bufio.NewScanner(strings.NewReader(content))
	lines := make([]string, 0)
	found := false
	var warnings []string
	var resolvedVersion string
	var versionLine string

	for scanner.Scan() {
		line := scanner.Text()
		if !found && isTopLevelVersionLine(line) {
			updatedLine, finalVersion, lineWarnings, err := rewriteVersionLine(line, version, strategy, opts)
			if err != nil {
				return Result{}, fmt.Errorf("resolve version for %s: %w", path, err)
			}
			line = updatedLine
			resolvedVersion = finalVersion
			versionLine = updatedLine
			warnings = append(warnings, lineWarnings...)
			found = true
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("scan %s: %w", path, err)
	}
	if !found {
		return Result{}, fmt.Errorf("top-level version field not found in %s", path)
	}

	updated := strings.Join(lines, "\n")
	if strings.HasSuffix(content, "\n") {
		updated += "\n"
	}

	return Result{
		Version:     resolvedVersion,
		VersionLine: versionLine,
		Content:     updated,
		Warnings:    warnings,
	}, nil
}

func rewriteVersionLine(line, version, strategy string, opts Options) (string, string, []string, error) {
	value, quote, leadingSpace, spacerBeforeComment, comment := splitVersionLine(line)
	_, existingBuild, buildPresent, malformedBuild := parseExistingVersion(value)

	finalVersion, warnings, err := resolveVersion(normalizeVersion(version), strategy, existingBuild, buildPresent, malformedBuild, opts)
	if err != nil {
		return "", "", nil, err
	}
	if malformedBuild {
		warnings = append(warnings, fmt.Sprintf("existing build number %q is malformed; treating it as 0", value))
	}

	quoted := finalVersion
	if quote != "" {
		quoted = quote + finalVersion + quote
	}
	updatedLine := "version:" + leadingSpace + quoted + spacerBeforeComment + comment
	return updatedLine, finalVersion, warnings, nil
}

func resolveVersion(version, strategy string, existingBuild int, buildPresent, malformedBuild bool, opts Options) (string, []string, error) {
	if override := strings.TrimSpace(opts.ExplicitBuildNumber); override != "" {
		build, err := parseNonNegativeInt("SEMREL_PLUGIN_PUBSPEC_BUILD_NUMBER", override)
		if err != nil {
			return "", nil, err
		}
		return versionWithBuild(version, build), nil, nil
	}

	switch strategy {
	case BuildStrategyNone:
		return version, nil, nil
	case BuildStrategyIncrement:
		if malformedBuild || !buildPresent {
			return versionWithBuild(version, 1), nil, nil
		}
		return versionWithBuild(version, existingBuild+1), nil, nil
	case BuildStrategyReleaseCount:
		for _, source := range []struct {
			name  string
			value string
		}{
			{name: "SEMREL_RELEASE_COUNT", value: opts.ReleaseCount},
			{name: "SEMREL_BUILD_NUMBER", value: opts.BuildNumber},
		} {
			if strings.TrimSpace(source.value) == "" {
				continue
			}
			build, err := parseNonNegativeInt(source.name, source.value)
			if err != nil {
				return "", nil, err
			}
			return versionWithBuild(version, build), nil, nil
		}
		fallback := 1
		if buildPresent && !malformedBuild {
			fallback = existingBuild + 1
		}
		return versionWithBuild(version, fallback), []string{"release-count strategy requested, but neither SEMREL_RELEASE_COUNT nor SEMREL_BUILD_NUMBER is set; falling back to existing build number + 1"}, nil
	default:
		return "", nil, fmt.Errorf("unsupported build strategy %q", strategy)
	}
}

func isTopLevelVersionLine(line string) bool {
	return strings.HasPrefix(line, "version:")
}

func splitVersionLine(line string) (value, quote, leadingSpace, spacerBeforeComment, comment string) {
	after := strings.TrimPrefix(line, "version:")
	trimmedLeft := strings.TrimLeft(after, " \t")
	leadingSpace = after[:len(after)-len(trimmedLeft)]

	valuePortion := trimmedLeft
	if idx := strings.Index(valuePortion, "#"); idx >= 0 {
		beforeComment := valuePortion[:idx]
		comment = valuePortion[idx:]
		rawValue := strings.TrimRight(beforeComment, " \t")
		spacerBeforeComment = beforeComment[len(rawValue):]
		valuePortion = rawValue
	}

	value = strings.TrimSpace(valuePortion)
	if len(value) >= 2 {
		if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
			quote = string(value[0])
			value = value[1 : len(value)-1]
		}
	}

	return value, quote, leadingSpace, spacerBeforeComment, comment
}

func parseExistingVersion(value string) (string, int, bool, bool) {
	clean := normalizeVersion(strings.TrimSpace(value))
	if clean == "" {
		return "", 0, false, false
	}
	parts := strings.SplitN(clean, "+", 2)
	if len(parts) == 1 {
		return parts[0], 0, false, false
	}
	build, err := strconv.Atoi(parts[1])
	if err != nil || build < 0 {
		return parts[0], 0, false, true
	}
	return parts[0], build, true, false
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func normalizeStrategy(strategy string) string {
	if strings.TrimSpace(strategy) == "" {
		return BuildStrategyNone
	}
	return strings.ToLower(strings.TrimSpace(strategy))
}

func versionWithBuild(version string, build int) string {
	return fmt.Sprintf("%s+%d", normalizeVersion(version), build)
}

func parseNonNegativeInt(name, value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return parsed, nil
}

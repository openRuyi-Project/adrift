// SPDX-FileCopyrightText: (C) 2026 Institute of Software, Chinese Academy of Sciences (ISCAS)
// SPDX-FileCopyrightText: (C) 2026 openRuyi Project Contributors
// SPDX-FileContributor: misaka00251 <liuxin@iscas.ac.cn>
//
// SPDX-License-Identifier: MulanPSL-2.0

package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// BuildDepInfo represents OBS build dependency information.
type BuildDepInfo struct {
	XMLName  xml.Name       `xml:"builddepinfo"`
	Packages []BuildPackage `xml:"package"`
}

// BuildPackage represents a package in OBS build dependency information.
type BuildPackage struct {
	Name    string   `xml:"name,attr"`
	Source  string   `xml:"source"`
	PkgDeps []string `xml:"pkgdep"`
	SubPkgs []string `xml:"subpkg"`
}

// Candidate represents a source package with no reverse dependencies.
type Candidate struct {
	Source   string
	OBSNames []string
}

// DependencyReport contains the results of BuildRequires analysis.
type DependencyReport struct {
	PackageRecords       int
	SourcePackages       int
	Candidates           []Candidate
	UnmappedDependencies []string
}

// Add a name to a slice if it does not already exist.
func addUnique(names []string, name string) []string {
	for _, oldName := range names {
		if oldName == name {
			return names
		}
	}

	return append(names, name)
}

// Map a package name to its source package.
// Return an error if the same name belongs to different sources.
func addAlias(packageToSource map[string]string, name, source string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return nil
	}

	if oldSource, ok := packageToSource[name]; ok && oldSource != source {
		return fmt.Errorf(
			"package name %q maps to both %q and %q",
			name, oldSource, source,
		)
	}

	packageToSource[name] = source
	return nil
}

// Analyze BuildRequires dependencies.
func AnalyzeDependencies(info BuildDepInfo) (DependencyReport, error) {

	// Map source packages to their OBS package names.
	sourceToNames := make(map[string][]string)

	// Map package names and subpackage names to source packages.
	packageToSource := make(map[string]string)

	// First pass: build package name mappings.
	for i, pkg := range info.Packages {
		name := strings.TrimSpace(pkg.Name)
		source := strings.TrimSpace(pkg.Source)

		if name == "" || source == "" {
			return DependencyReport{}, fmt.Errorf(
				"package record %d has an empty name or source",
				i+1,
			)
		}

		sourceToNames[source] = addUnique(
			sourceToNames[source], name,
		)

		if err := addAlias(packageToSource, source, source); err != nil {
			return DependencyReport{}, err
		}

		if err := addAlias(packageToSource, name, source); err != nil {
			return DependencyReport{}, err
		}

		for _, subpkg := range pkg.SubPkgs {
			if err := addAlias(packageToSource, subpkg, source); err != nil {
				return DependencyReport{}, err
			}
		}
	}

	// referencedBy[target][dependent] records who depends on target.
	referencedBy := make(map[string]map[string]bool)

	// Dependencies that cannot be mapped to this XML.
	unmapped := make(map[string]bool)

	// Second pass: build reverse dependencies.
	for _, pkg := range info.Packages {
		from := strings.TrimSpace(pkg.Source)

		for _, dep := range pkg.PkgDeps {
			dep = strings.TrimSpace(dep)

			if dep == "" {
				continue
			}

			target, ok := packageToSource[dep]

			if !ok {
				unmapped[dep] = true
				continue
			}

			// Ignore dependencies within the same source package.
			if target == from {
				continue
			}

			if referencedBy[target] == nil {
				referencedBy[target] = make(map[string]bool)
			}

			referencedBy[target][from] = true
		}
	}

	// Find packages that have no reverse dependencies.
	report := DependencyReport{
		PackageRecords: len(info.Packages),
		SourcePackages: len(sourceToNames),
	}

	for source, names := range sourceToNames {
		if len(referencedBy[source]) > 0 {
			continue
		}

		sort.Strings(names)

		report.Candidates = append(report.Candidates, Candidate{
			Source:   source,
			OBSNames: names,
		})
	}

	// Sort candidates for predictable output.
	sort.Slice(report.Candidates, func(i, j int) bool {
		return report.Candidates[i].Source <
			report.Candidates[j].Source
	})

	// Collect unmapped dependencies.
	for dep := range unmapped {
		report.UnmappedDependencies = append(
			report.UnmappedDependencies, dep,
		)
	}

	sort.Strings(report.UnmappedDependencies)

	return report, nil
}

// Import XML and print analysis results.
func importXML(path string, out io.Writer) error {
	var reader io.Reader

	if path == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Cannot open XML: %w", err)
		}
		defer file.Close()

		reader = file
	}

	var info BuildDepInfo

	if err := xml.NewDecoder(reader).Decode(&info); err != nil {
		return fmt.Errorf("cannot parse XML: %w", err)
	}

	if info.XMLName.Local != "builddepinfo" {
		return fmt.Errorf("unexpected XML root: %s", info.XMLName.Local)
	}

	if len(info.Packages) == 0 {
		return fmt.Errorf("no packages found in XML")
	}

	report, err := AnalyzeDependencies(info)
	if err != nil {
		return err
	}

	// Print summary.
	fmt.Fprintln(out, "BuildRequires analysis")
	fmt.Fprintln(out, "======================")
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Package records: %d\n", report.PackageRecords)
	fmt.Fprintf(out, "Source packages: %d\n", report.SourcePackages)
	fmt.Fprintf(out, "Candidates: %d\n", len(report.Candidates))
	fmt.Fprintf(out, "Dependencies not found in this XML: %d\n",
		len(report.UnmappedDependencies))

	// Print candidates.
	fmt.Fprintln(out, "\nPotentially unreferenced packages:")

	if len(report.Candidates) == 0 {
		fmt.Fprintln(out, "  (none)")
	}

	for _, candidate := range report.Candidates {
		fmt.Fprintf(out, "  %-45s source: %s\n",
			strings.Join(candidate.OBSNames, ", "),
			candidate.Source,
		)
	}

	// Print unmapped dependencies, if any.
	if len(report.UnmappedDependencies) > 0 {
		fmt.Fprintln(out, "\nDependencies not found in this XML:")

		for _, dep := range report.UnmappedDependencies {
			fmt.Fprintf(out, "  %s\n", dep)
		}
	}

	return nil
}

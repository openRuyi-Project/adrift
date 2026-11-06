// SPDX-FileCopyrightText: (C) 2026 Institute of Software, Chinese Academy of Sciences (ISCAS)
// SPDX-FileCopyrightText: (C) 2026 openRuyi Project Contributors
// SPDX-FileContributor: misaka00251 <liuxin@iscas.ac.cn>
//
// SPDX-License-Identifier: MulanPSL-2.0

package main

import (
	"fmt"
	"io"
	"os"
)

var version = "dev"

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: adrift <command> [arguments]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  help                 Show this help")
	fmt.Fprintln(w, "  version              Show version")
	fmt.Fprintln(w, "  import <file.xml>    Import and analyze OBS XML")
	fmt.Fprintln(w, "  import -             Read XML from stdin")
}

func main() {
	fmt.Println("adrift - an orphan package detecter for openRuyi")
	fmt.Println()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: missing command.")
		printUsage(os.Stderr)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "help":
		printUsage(os.Stdout)
	case "version":
		fmt.Printf("%s\n", version)
	case "import":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "Error: import requires an XML file.")
			os.Exit(2)
		}

		if err := importXML(os.Args[2], os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "default":
		fmt.Fprintf(os.Stderr, "Error: unknown command %q.\n", os.Args[1])
		fmt.Println("Try adrift help.\n")
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {

	wd, err := os.Getwd()
	abortonerr(err, "getting working dir")

	dir := ""
	flag.StringVar(&dir, "dir", wd, "dir that will be recursively walked for deps")
	flag.Parse()

	packages := parseAllDependencies(dir)
	for pkg := range packages {
		// TODO: could use concurrency here (fan out -> fan in)
		getPackage(pkg)
	}

	for pkg := range packages {
		// TODO: could use concurrency here (fan out -> fan in)
		vendorPackage(pkg)
	}
}

func parsePkgDependencies(dir string) []string {
	fileset := token.NewFileSet()
	pkgsAST, err := parser.ParseDir(fileset, dir, nil, parser.ImportsOnly)
	abortonerr(err, fmt.Sprintf("parsing dir[%s] for Go file", dir))

	pkgs := []string{}

	for _, pkgAST := range pkgsAST {
		for _, file := range pkgAST.Files {
			for _, pkg := range file.Imports {
				pkgs = append(pkgs, strings.Trim(pkg.Path.Value, "\""))
			}
		}
	}

	return pkgs
}

func parseAllDependencies(rootdir string) map[string]struct{} {
	deps := map[string]struct{}{}

	filepath.Walk(rootdir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		if strings.Contains(path, "/vendor/") {
			return nil
		}

		for _, pkg := range parsePkgDependencies(path) {
			deps[pkg] = struct{}{}
		}
		return nil
	})

	return deps
}

func getPackage(pkg string) {
	cmd := exec.Command("go", "get", pkg)
	fmt.Printf("go get %s\n", pkg)
	output, err := cmd.CombinedOutput()
	details := fmt.Sprintf("running go get %s. output: %s", pkg, string(output))
	abortonerr(err, details)
}

func vendorPackage(pkg string) {
}

func abortonerr(err error, details string) {
	if err != nil {
		fmt.Printf("unexpected error[%s] %s\n", err, details)
		os.Exit(1)
	}
}

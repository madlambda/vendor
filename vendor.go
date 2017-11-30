package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
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

	for name, pkgAST := range pkgsAST {
		fmt.Printf("name[%s]\nimports:\n", name)
		for name, file := range pkgAST.Files {
			fmt.Printf("filename[%s] imports:\n", name)
			for _, pkg := range file.Imports {
				fmt.Println(pkg.Path.Value)
			}
			fmt.Println("---------")
		}
		fmt.Println("--------")
	}

	return []string{}
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

		parsePkgDependencies(path)
		return nil
	})

	return deps
}

func getPackage(pkg string) {
}

func vendorPackage(pkg string) {
}

func abortonerr(err error, details string) {
	if err != nil {
		fmt.Printf("unexpected error[%s] %s\n", err, details)
		os.Exit(1)
	}
}

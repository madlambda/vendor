package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {

	wd, err := os.Getwd()
	abortonerr(err, "getting working dir")

	dir := ""
	flag.StringVar(&dir, "dir", wd, "dir that will be recursively walked for deps")
	flag.Parse()

	packages := parseDependencies(dir)
	for pkg := range packages {
		// TODO: could use concurrency here (fan out -> fan in)
		getPackage(pkg)
	}

	for pkg := range packages {
		// TODO: could use concurrency here (fan out -> fan in)
		vendorPackage(pkg)
	}
}

func parseDependencies(dir string) map[string]struct{} {
	deps := map[string]struct{}{}
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

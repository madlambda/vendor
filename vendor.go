package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func main() {

	wd, err := os.Getwd()
	abortonerr(err, "getting working dir")

	projectdir := ""
	flag.StringVar(&projectdir, "dir", wd, "dir that will be recursively walked for deps")
	flag.Parse()

	gohome := getGoHome()
	if !strings.HasPrefix(projectdir, gohome) {
		fmt.Println("dir must be inside your GOPATH")
		os.Exit(1)
	}

	packages := parseAllDependencies(gohome, projectdir)
	for pkg := range packages {
		// TODO: could use concurrency here (fan out -> fan in)
		getPackage(pkg)
	}

	for pkg := range packages {
		// TODO: could use concurrency here (fan out -> fan in)
		vendorPackage(gohome, projectdir, pkg)
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

func parseProjectDomain(gohome string, rootdir string) string {
	projectroot := strings.TrimPrefix(rootdir, path.Join(gohome, "src"))
	return projectroot[1:]
}

func parseAllDependencies(gohome string, rootdir string) map[string]struct{} {
	deps := map[string]struct{}{}
	projectRoot := parseProjectDomain(gohome, rootdir)

	filepath.Walk(rootdir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		if strings.Contains(path, "/vendor/") {
			return nil
		}

		for _, pkg := range parsePkgDependencies(path) {
			if strings.HasPrefix(pkg, projectRoot) {
				continue
			}
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

func getGoHome() string {
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		return gopath
	}
	home := os.Getenv("HOME")
	if home == "" {
		fmt.Println("no GOPATH env var found and no HOME to infer GOPATH from")
		os.Exit(1)
	}
	return path.Join(home, "go")
}

func vendorPackage(gohome string, rootdir string, pkg string) {
	srcpkgpath := path.Join(gohome, "src", pkg)

	entries, err := ioutil.ReadDir(srcpkgpath)
	if err != nil {
		// WHY: supposing that invalid paths are probably builtin packages
		// This makes sense because go get fails with names that do not
		// match any builtin or that can't be downloaded
		return
	}

	targetpkgpath := path.Join(rootdir, "vendor", pkg)
	err = os.MkdirAll(targetpkgpath, 0664)
	abortonerr(err, fmt.Sprintf("creating vendor dir[%s]", targetpkgpath))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		srcpath := path.Join(srcpkgpath, entry.Name())
		dstpath := path.Join(targetpkgpath, entry.Name())
		copyFile(srcpath, dstpath)
	}
}

func closeFile(f io.Closer, name string) {
	err := f.Close()
	abortonerr(err, fmt.Sprintf("closing %s", name))
}

func copyFile(src string, dst string) {
	in, err := os.Open(src)
	abortonerr(err, fmt.Sprintf("opening %s", src))
	defer closeFile(in, src)

	out, err := os.Create(dst)
	abortonerr(err, fmt.Sprintf("opening %s", out))
	defer closeFile(out, dst)

	_, err = io.Copy(out, in)
	abortonerr(err, fmt.Sprintf("copying %s to %s", src, dst))
}

func abortonerr(err error, details string) {
	if err != nil {
		fmt.Printf("unexpected error[%s] %s\n", err, details)
		os.Exit(1)
	}
}

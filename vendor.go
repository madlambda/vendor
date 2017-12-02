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
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

var cleanfuncs []func()

func main() {
	wd, err := os.Getwd()
	abortonerr(err, "getting working dir")

	dir := ""
	flag.StringVar(&dir, "dir", wd, "dir that will be recursively walked for deps")
	flag.Parse()

	gopath := getGoPath()
	projectdir, err := filepath.Abs(dir)
	abortonerr(err, fmt.Sprintf("getting absolute path of[%s]", dir))

	if !strings.HasPrefix(projectdir, gopath) {
		fmt.Println("dir must be inside your GOPATH")
		os.Exit(1)
	}

	packages := parseAllDeps(gopath, projectdir)
	depsGoHome, err := ioutil.TempDir("", "vendor")
	abortonerr(err, "creating temp dir")
	addCleanup(func() { os.RemoveAll(depsGoHome) })

	os.Setenv("GOPATH", depsGoHome)
	for pkg := range packages {
		// TODO: could use concurrency here (fan out -> fan in)
		getPackage(pkg)
	}
	os.Setenv("GOPATH", gopath)

	vendorPackages(depsGoHome, projectdir)
	cleanup()
}

func parsePkgDeps(dir string) []string {
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

func parseImportPath(gopath string, rootdir string) string {
	projectroot := strings.TrimPrefix(rootdir, path.Join(gopath, "src"))
	return projectroot[1:]
}

func parseAllDeps(gopath string, rootdir string) map[string]struct{} {
	deps := map[string]struct{}{}
	projectImportPath := parseImportPath(gopath, rootdir)

	filepath.Walk(rootdir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		if strings.Contains(path, "/vendor/") {
			return nil
		}

		for _, pkg := range parsePkgDeps(path) {
			if strings.HasPrefix(pkg, projectImportPath) {
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

func getGoPath() string {
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		return gopath
	}
	u, err := user.Current()
	abortonerr(err, "getting current user")
	return path.Join(u.HomeDir, "go")
}

func createDir(dir string) {
	err := os.MkdirAll(dir, 0774)
	abortonerr(err, fmt.Sprintf("creating dir[%s]", dir))
}

func vendorPackages(depsGoHome string, projectdir string) {
	depsrootdir := path.Join(depsGoHome, "src")
	projectVendorPath := path.Join(projectdir, "vendor")

	filepath.Walk(depsrootdir, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		// Not sure this works, but seems feasible to ignore
		// vendoring from libs, they should not do that
		if strings.Contains(path, "/vendor/") {
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		vendoredPath := filepath.Join(
			projectVendorPath,
			strings.TrimPrefix(path, depsrootdir),
		)

		vendoredDir := filepath.Dir(vendoredPath)
		createDir(vendoredDir)
		copyFile(path, vendoredPath)
		return nil
	})
}

func closeFile(f io.Closer, name string) {
	err := f.Close()
	abortonerr(err, fmt.Sprintf("closing %s", name))
}

func copyFile(src string, dst string) {
	in, err := os.Open(src)
	abortonerr(err, fmt.Sprintf("opening %s", src))
	addCleanup(func() { closeFile(in, src) })

	out, err := os.Create(dst)
	abortonerr(err, fmt.Sprintf("opening %s", out))
	addCleanup(func() { closeFile(out, dst) })

	_, err = io.Copy(out, in)
	abortonerr(err, fmt.Sprintf("copying %s to %s", src, dst))
}

func cleanup() {
	for _, cleanfunc := range cleanfuncs {
		cleanfunc()
	}
	cleanfuncs = nil
}

func addCleanup(c func()) {
	cleanfuncs = append(cleanfuncs, c)
}

func abortonerr(err error, details string) {
	if err != nil {
		fmt.Sprintf("unexpected error[%s] %s\n", err, details)
		cleanup()
		os.Exit(1)
	}
}

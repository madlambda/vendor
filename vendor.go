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
	projectroot := strings.TrimPrefix(rootdir, filepath.Join(gopath, "src"))
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
	return filepath.Join(u.HomeDir, "go")
}

func createDir(dir string) {
	err := os.MkdirAll(dir, 0774)
	abortonerr(err, fmt.Sprintf("creating dir[%s]", dir))
}

func vendorPackages(depsGoHome string, projectdir string) {
	depsrootdir := filepath.Join(depsGoHome, "src")
	projectVendorPath := filepath.Join(projectdir, "vendor")

	err := os.RemoveAll(projectVendorPath)
	abortonerr(err, "removing project vendor path")

	filepath.Walk(depsrootdir, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return nil
		}

		if ignore, err := ignorePath(path, info.IsDir()); ignore {
			return err
		}

		if info.IsDir() {
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

func ignorePath(path string, isdir bool) (bool, error) {
	base := filepath.Base(path)
	if isdir {
		if base == "vendor" || base == "testdata" {
			return true, filepath.SkipDir
		}
	}
	if base[0] == '.' || base[0] == '_' {
		if isdir {
			return true, filepath.SkipDir
		}
		return true, nil
	}
	if strings.HasSuffix(path, "_test.go") {
		return true, nil
	}
	ext := filepath.Ext(path)
	return ext != ".go" && ext != ".s" && ext != ".c" && ext != ".h", nil
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
	abortonerr(err, fmt.Sprintf("opening %v", out))

	defer closeFile(out, dst)

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
		fmt.Printf("unexpected error[%s] %s\n", err, details)
		cleanup()
		os.Exit(1)
	}
}

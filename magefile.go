//go:build mage

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = Build
var ldflags = "-s -w -X main.AUTHOR=${AUTHOR} -X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT} -X main.BRANCH=${BRANCH} -X main.TAG=${TAG} -X main.BUILDTIME=${BUILDTIME}"
var goexe = "go"
var Binary = "glutton"
var VetReport = "vet.report"
var TestReport = "tests.xml"
var Goarch = "amd64"
var Goos = "linux"

func init() {
	if exe := os.Getenv("GOEXE"); exe != "" {
		goexe = exe
	}

	// We want to use Go 1.11 modules even if the source lives inside GOPATH.
	// The default is "auto".
	os.Setenv("GO111MODULE", "on")
}

func flagEnv() map[string]string {
	hash, _ := sh.Output("git", "rev-parse", "--short", "HEAD")
	branch, _ := sh.Output("git", "rev-parse", "--abbrev-ref", "HEAD")
	author, _ := sh.Output("git", "log", "-1", "--pretty=format:'%an'")
	version, _ := sh.Output("git", "describe", "--tags", "--abbrev=0")
	return map[string]string{
		"COMMIT":      normalizeString(hash),
		"BRANCH":      normalizeString(branch),
		"AUTHOR":      normalizeString(author),
		"VERSION":     normalizeString(version),
		"BUILDTIME":   time.Now().Format("2006-01-02T15:04:05Z0700"),
		"GOARCH":      Goarch,
		"GOOS":        Goos,
		"CGO_ENABLED": "0",
	}
}

// A build step that requires additional params, or platform specific steps for example
func Build() error {
	if needed, err := isBuildNeeded(); !needed && err == nil {
		log.Printf("Build not required.")
		return nil
	}
	mg.Deps(InstallDeps)
	mg.Deps(TestRace)
	mg.Deps(Lint)
	mg.Deps(Fmt)
	mg.Deps(Vet)
	return sh.RunWith(flagEnv(), goexe, "build", "-ldflags", ldflags, "-o", appName(), "github.com/defectus/glutton/cmd/glutton")
}

// A custom install step if you need your bin someplace other than go/bin
func Install() error {
	mg.Deps(Build)
	return os.Rename(appName(), "/usr/bin/"+appName())
}

// Manage your deps, or running package managers.
func InstallDeps() error {
	cmd := exec.Command("go", "get", "github.com/tebeka/go2xunit")
	return cmd.Run()
}

// Run tests
func Test() error {
	return sh.Run(goexe, "test", "-coverprofile=coverage.txt", "-covermode=atomic", "-v", "-tags", buildTags(), "./...")
}

// Run tests with race detector
func TestRace() error {
	return sh.Run(goexe, "test", "-race", "-coverprofile=coverage.txt", "-covermode=atomic", "-v", "-tags", buildTags(), "./...")
}

// Run tests in 32-bit mode
// Note that we don't run with the extended tag. Currently not supported in 32 bit.
func Test386() error {
	return sh.RunWith(map[string]string{"GOARCH": "386"}, goexe, "test", "./...")
}

// Run go vet linter
func Vet() error {
	if err := sh.Run(goexe, "vet", "./..."); err != nil {
		return fmt.Errorf("error running go vet: %v", err)
	}
	return nil
}

// Run golint linter
func Lint() error {
	pkgs, err := packages()
	if err != nil {
		return err
	}
	failed := false
	for _, pkg := range pkgs {
		// We don't actually want to fail this target if we find golint errors,
		// so we don't pass -set_exit_status, but we still print out any failures.
		if _, err := sh.Exec(nil, os.Stderr, nil, "golint", pkg); err != nil {
			fmt.Printf("ERROR: running go lint on %q: %v\n", pkg, err)
			failed = true
		}
	}
	if failed {
		return errors.New("errors running golint")
	}
	return nil
}

// Run gofmt linter
func Fmt() error {
	pkgs, err := packages()
	if err != nil {
		return err
	}
	failed := false
	first := true
	for _, pkg := range pkgs {
		files, err := filepath.Glob(filepath.Join(pkg, "*.go"))
		if err != nil {
			return nil
		}
		for _, f := range files {
			// gofmt doesn't exit with non-zero when it finds unformatted code
			// so we have to explicitly look for output, and if we find any, we
			// should fail this target.
			s, err := sh.Output("gofmt", "-l", f)
			if err != nil {
				fmt.Printf("ERROR: running gofmt on %q: %v\n", f, err)
				failed = true
			}
			if s != "" {
				if first {
					fmt.Println("The following files are not gofmt'ed:")
					first = false
				}
				failed = true
				fmt.Println(s)
			}
		}
	}
	if failed {
		return errors.New("improperly formatted go files")
	}
	return nil
}

// Builds and pushes docker image to dockerhub repository (you must have access to it).
func Docker() error {
	err := sh.RunWith(flagEnv(), "docker", "build", "-f", "docker/Dockerfile", "-t", "defectus/glutton:latest", "-t", "defectus/glutton:${VERSION}", ".")
	if err != nil {
		return err
	}
	err = sh.RunWith(flagEnv(), "docker", "push", "defectus/glutton:${VERSION}")
	if err != nil {
		return err
	}
	return sh.RunWith(flagEnv(), "docker", "push", "defectus/glutton:latest")
}

// Clean up after yourself
func Clean() {
	os.RemoveAll(appName())
	os.RemoveAll("tests.xml")
	os.RemoveAll("vet.report")
	os.RemoveAll("coverage.txt")
}

func buildTags() string {
	// To build the extended Glutton SCSS/SASS enabled version, build with
	// GLUTTON_BUILD_TAGS=extended mage install etc.
	if envtags := os.Getenv("GLUTTON_BUILD_TAGS"); envtags != "" {
		return envtags
	}
	return "none"
}

var (
	pkgPrefixLen = len("github.com/defectus/glutton")
	pkgs         []string
	pkgsInit     sync.Once
)

func packages() ([]string, error) {
	var err error
	pkgsInit.Do(func() {
		var s string
		s, err = sh.Output(goexe, "list", "./...")
		if err != nil {
			return
		}
		pkgs = strings.Split(s, "\n")
		for i := range pkgs {
			pkgs[i] = "." + pkgs[i][pkgPrefixLen:]
		}
	})
	return pkgs, err
}

func isBuildNeeded() (bool, error) {
	return target.Dir(appName(), "pkg", "cmd")
}

func appName() string {
	return Binary + "-" + Goos + "-" + Goarch
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func normalizeString(input string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, input)
	result = strings.Replace(result, " ", "_", -1)
	return result
}

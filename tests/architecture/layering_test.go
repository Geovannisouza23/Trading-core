// Package architecture_test enforces the dependency-direction rules from
// spec sections 3, 4 and 6 using `go list -json`, so a future change that
// violates layering fails CI instead of only a code review.
package architecture_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type goPackage struct {
	ImportPath string
	Dir        string
	Imports    []string
	GoFiles    []string
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate go.mod above " + dir)
		}
		dir = parent
	}
}

func loadPackages(t *testing.T) []goPackage {
	t.Helper()
	root := moduleRoot(t)
	cmd := exec.Command("go", "list", "-buildvcs=false", "-json", "./...")
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list ./... failed: %v\n%s", err, out.String())
	}

	dec := json.NewDecoder(&out)
	var pkgs []goPackage
	for dec.More() {
		var p goPackage
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decoding go list output: %v", err)
		}
		pkgs = append(pkgs, p)
	}
	if len(pkgs) == 0 {
		t.Fatal("go list returned no packages")
	}
	return pkgs
}

func hasPrefixAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func importsAny(imports []string, substrings ...string) []string {
	var hits []string
	for _, imp := range imports {
		if hasPrefixAny(imp, substrings...) {
			hits = append(hits, imp)
		}
	}
	return hits
}

// --- Domain purity --------------------------------------------------------

func TestDomainNeverImportsOuterLayersOrFrameworks(t *testing.T) {
	forbidden := []string{
		"trading-core/internal/application",
		"trading-core/internal/infrastructure",
		"trading-core/internal/interfaces",
		"trading-core/internal/app",
		"go.uber.org/fx",
		"github.com/go-chi/chi",
		"github.com/jackc/pgx",
		"go.opentelemetry.io",
		"google.golang.org/grpc",
	}
	for _, pkg := range loadPackages(t) {
		if !strings.Contains(pkg.ImportPath, "/internal/domain/") && !strings.HasSuffix(pkg.ImportPath, "/internal/domain") {
			continue
		}
		if hits := importsAny(pkg.Imports, forbidden...); len(hits) > 0 {
			t.Errorf("domain package %s imports forbidden dependencies: %v", pkg.ImportPath, hits)
		}
	}
}

// --- Application purity ----------------------------------------------------

func TestApplicationNeverImportsConcreteInfrastructureOrInterfaces(t *testing.T) {
	forbidden := []string{
		"trading-core/internal/infrastructure",
		"trading-core/internal/interfaces",
		"go.uber.org/fx",
	}
	for _, pkg := range loadPackages(t) {
		if !strings.Contains(pkg.ImportPath, "/internal/application/") {
			continue
		}
		if hits := importsAny(pkg.Imports, forbidden...); len(hits) > 0 {
			t.Errorf("application package %s imports forbidden dependencies: %v", pkg.ImportPath, hits)
		}
	}
}

// --- Interfaces layer boundaries --------------------------------------------

func TestHTTPHandlersNeverAccessDatabaseOrBrokerDirectly(t *testing.T) {
	forbidden := []string{
		"trading-core/internal/infrastructure/database",
		"trading-core/internal/infrastructure/external",
		"github.com/jackc/pgx",
	}
	for _, pkg := range loadPackages(t) {
		if !strings.Contains(pkg.ImportPath, "/internal/interfaces/http/handler") {
			continue
		}
		if hits := importsAny(pkg.Imports, forbidden...); len(hits) > 0 {
			t.Errorf("HTTP handler package %s imports forbidden dependencies: %v", pkg.ImportPath, hits)
		}
	}
}

func TestConsumersNeverAccessDatabaseDirectly(t *testing.T) {
	forbidden := []string{
		"trading-core/internal/infrastructure/database",
		"github.com/jackc/pgx",
	}
	for _, pkg := range loadPackages(t) {
		if !strings.Contains(pkg.ImportPath, "/internal/interfaces/consumer/") {
			continue
		}
		if hits := importsAny(pkg.Imports, forbidden...); len(hits) > 0 {
			t.Errorf("consumer package %s imports forbidden dependencies: %v", pkg.ImportPath, hits)
		}
	}
}

func TestInfrastructureNeverImportsInterfaces(t *testing.T) {
	for _, pkg := range loadPackages(t) {
		if !strings.Contains(pkg.ImportPath, "/internal/infrastructure/") {
			continue
		}
		if hits := importsAny(pkg.Imports, "trading-core/internal/interfaces"); len(hits) > 0 {
			t.Errorf("infrastructure package %s imports the interfaces layer: %v", pkg.ImportPath, hits)
		}
	}
}

// --- Composition-only concerns ----------------------------------------------

// fxAllowedExactPackages are the only packages outside cmd/* and
// internal/app/* allowed to import Uber Fx: composition-only module files
// (spec section 25).
var fxAllowedExactPackages = map[string]bool{
	"trading-core/internal/infrastructure/database": true, // module.go
}

func TestUberFxOnlyInCmdAppOrCompositionModules(t *testing.T) {
	for _, pkg := range loadPackages(t) {
		if strings.HasPrefix(pkg.ImportPath, "trading-core/cmd/") {
			continue
		}
		if strings.Contains(pkg.ImportPath, "/internal/app") {
			continue
		}
		if fxAllowedExactPackages[pkg.ImportPath] {
			continue
		}
		if len(importsAny(pkg.Imports, "go.uber.org/fx")) > 0 {
			t.Errorf("package %s imports go.uber.org/fx outside cmd/, internal/app, or an allow-listed composition module", pkg.ImportPath)
		}
	}
}

var osEnvPattern = regexp.MustCompile(`\bos\.(Getenv|LookupEnv|Environ)\s*\(`)

func TestOsGetenvOnlyInsideConfigPackage(t *testing.T) {
	for _, pkg := range loadPackages(t) {
		if strings.Contains(pkg.ImportPath, "/internal/config") {
			continue
		}
		for _, file := range pkg.GoFiles {
			path := filepath.Join(pkg.Dir, file)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}
			if osEnvPattern.Match(content) {
				t.Errorf("%s reads the environment directly outside internal/config", path)
			}
		}
	}
}

// --- Forbidden / duplicated folder names ------------------------------------

func TestNoForbiddenOrDuplicateDirectoryNames(t *testing.T) {
	root := moduleRoot(t)
	internalDir := filepath.Join(root, "internal")

	var databaseDirs []string
	forbiddenNames := map[string]bool{"persistence": true, "bootstrap": true}

	err := filepath.WalkDir(internalDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if forbiddenNames[name] {
			t.Errorf("forbidden directory name %q found at %s", name, path)
		}
		if name == "database" {
			databaseDirs = append(databaseDirs, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", internalDir, err)
	}

	want := filepath.Join(internalDir, "infrastructure", "database")
	for _, dir := range databaseDirs {
		if dir != want {
			t.Errorf("found a second \"database\" directory at %s; only %s is allowed", dir, want)
		}
	}
	if len(databaseDirs) == 0 {
		t.Fatalf("expected to find %s, found none", want)
	}
}

func TestMigrationsOnlyInCanonicalLocation(t *testing.T) {
	root := moduleRoot(t)
	want := filepath.Join(root, "internal", "infrastructure", "database", "migrations")

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".up.sql") || strings.HasSuffix(path, ".down.sql") {
			if filepath.Dir(path) != want {
				t.Errorf("migration file %s is outside %s", path, want)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking internal/: %v", err)
	}
}

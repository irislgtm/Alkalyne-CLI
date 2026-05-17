package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type importRule struct {
	pkg       string
	forbidden []string
}

var rules = []importRule{
	{
		pkg: "github.com/alkalyne/alkalyne/internal/models",
		forbidden: []string{
			"github.com/alkalyne/alkalyne/internal/",
		},
	},
	{
		pkg: "github.com/alkalyne/alkalyne/internal/config",
		forbidden: []string{
			"github.com/alkalyne/alkalyne/internal/db",
			"github.com/alkalyne/alkalyne/internal/p2p",
			"github.com/alkalyne/alkalyne/internal/tui",
			"github.com/alkalyne/alkalyne/internal/mailbox",
		},
	},
	{
		pkg: "github.com/alkalyne/alkalyne/internal/db",
		forbidden: []string{
			"github.com/alkalyne/alkalyne/internal/p2p",
			"github.com/alkalyne/alkalyne/internal/tui",
			"github.com/alkalyne/alkalyne/internal/mailbox",
			"github.com/alkalyne/alkalyne/internal/config",
		},
	},
	{
		pkg: "github.com/alkalyne/alkalyne/internal/p2p",
		forbidden: []string{
			"github.com/alkalyne/alkalyne/internal/tui",
			"github.com/alkalyne/alkalyne/internal/mailbox",
			"github.com/alkalyne/alkalyne/internal/db",
		},
	},
	{
		pkg: "github.com/alkalyne/alkalyne/internal/tui",
		forbidden: []string{
			"github.com/alkalyne/alkalyne/internal/mailbox",
		},
	},
	{
		pkg: "github.com/alkalyne/alkalyne/internal/mailbox",
		forbidden: []string{
			"github.com/alkalyne/alkalyne/internal/tui",
		},
	},
}

func TestImportBoundaries(t *testing.T) {
	root := moduleRoot(t)

	for _, rule := range rules {
		t.Run(rule.pkg, func(t *testing.T) {
			imports := collectImports(t, root, rule.pkg)
			for _, forbid := range rule.forbidden {
				for _, imp := range imports {
					if strings.HasPrefix(imp, forbid) {
						t.Errorf("%s must not import %s (found: %s)", rule.pkg, forbid, imp)
					}
				}
			}
		})
	}
}

func collectImports(t *testing.T, root, pkgPath string) []string {
	t.Helper()
	rel := strings.TrimPrefix(pkgPath, "github.com/alkalyne/alkalyne/")
	pkgDir := filepath.Join(root, rel)

	entries, err := filepath.Glob(filepath.Join(pkgDir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", pkgDir, err)
	}

	seen := map[string]bool{}
	var result []string

	for _, entry := range entries {
		if strings.HasSuffix(entry, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, entry, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", entry, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !seen[path] {
				seen[path] = true
				result = append(result, path)
			}
		}
	}
	return result
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
			t.Fatal("go.mod not found from " + dir)
		}
		dir = parent
	}
}

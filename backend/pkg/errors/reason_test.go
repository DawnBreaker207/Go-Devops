package apperrors

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"testing"
)

// Every exported Err* sentinel must carry a stable Reason.
//
// This reads the SOURCE rather than the values: Go cannot enumerate a package's
// variables at run time, and the failure this guards against is a new sentinel added
// without a reason - which no value-based test could see, because the new sentinel
// would simply not be in whatever list the test knew about.
//
// Without this, the reason vocabulary decays back into exactly the silent
// fall-through it was introduced to replace: the frontend asks for a key that was
// never emitted, gets no translation, and shows the English sentence again.
func TestEverySentinelCarriesAReason(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "errors.go", nil, 0)
	if err != nil {
		t.Fatalf("parse errors.go: %v", err)
	}

	var missing []string
	var seen []string

	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range spec.Names {
			if !strings.HasPrefix(name.Name, "Err") || i >= len(spec.Values) {
				continue
			}
			seen = append(seen, name.Name)
			if !hasWithReason(spec.Values[i]) {
				missing = append(missing, name.Name)
			}
		}
		return true
	})

	if len(seen) == 0 {
		t.Fatal("found no Err* sentinels - the parser or the file layout changed")
	}
	if len(missing) > 0 {
		t.Fatalf("%d of %d sentinels have no .WithReason(...):\n  %s\n\n"+
			"Add one so the frontend can translate by key instead of by English sentence.",
			len(missing), len(seen), strings.Join(missing, "\n  "))
	}
	t.Logf("%d sentinels, all carrying a reason", len(seen))
}

var reasonKey = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Reasons are the frontend's translation keys, so they must be stable, unique and
// spelled predictably. A duplicate would silently make two different refusals read
// as the same sentence to the customer.
func TestReasonsAreUniqueAndWellFormed(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "errors.go", nil, 0)
	if err != nil {
		t.Fatalf("parse errors.go: %v", err)
	}

	owner := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range spec.Names {
			if !strings.HasPrefix(name.Name, "Err") || i >= len(spec.Values) {
				continue
			}
			reason := reasonOf(spec.Values[i])
			if reason == "" {
				continue // the other test reports this
			}
			if !reasonKey.MatchString(reason) {
				t.Errorf("%s: reason %q is not lower_snake_case", name.Name, reason)
			}
			if prev, dup := owner[reason]; dup {
				t.Errorf("reason %q is used by both %s and %s - two refusals would read the same", reason, prev, name.Name)
			}
			owner[reason] = name.Name
		}
		return true
	})
}

func hasWithReason(expr ast.Expr) bool { return reasonOf(expr) != "" }

// reasonOf digs the literal out of `Conflict("...").WithReason("key")`.
func reasonOf(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "WithReason" || len(call.Args) != 1 {
		return ""
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, `"`)
}

// The reason must survive the clone-based builders, or attaching details to a
// sentinel would quietly strip the key the frontend translates by.
func TestReasonSurvivesWithDetailsAndWrap(t *testing.T) {
	base := Conflict("nope").WithReason("test_reason")

	if got := base.WithDetails(map[string]string{"field": "bad"}).Reason; got != "test_reason" {
		t.Fatalf("WithDetails dropped the reason: %q", got)
	}
	if got := base.Wrap(ErrUserNotFound).Reason; got != "test_reason" {
		t.Fatalf("Wrap dropped the reason: %q", got)
	}
	if base.Reason != "test_reason" {
		t.Fatal("the builders must not mutate the original sentinel")
	}
}

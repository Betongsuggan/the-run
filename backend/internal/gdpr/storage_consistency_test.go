package gdpr_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BirgerRydback/the-run/backend/internal/gdpr"
)

// TestStorageConsistency parses infra/database/database.go via go/ast and
// verifies that every dynamodb.NewTable call has a matching entry in
// gdpr.Tables with the same SSE / PITR / TTL posture. If the Pulumi infra
// flips PITR on a table without the gdpr registry being updated, this fails.
func TestStorageConsistency(t *testing.T) {
	backendDir, err := findBackendDir()
	if err != nil {
		t.Fatalf("locate backend dir: %v", err)
	}
	repoRoot := filepath.Dir(backendDir)
	infraFile := filepath.Join(repoRoot, "infra", "database", "database.go")

	infraTables, err := extractInfraTables(infraFile)
	if err != nil {
		t.Fatalf("parse infra: %v", err)
	}
	if len(infraTables) == 0 {
		t.Fatalf("found no tables in %s — parser may be broken", infraFile)
	}

	registry := map[string]gdpr.TablePosture{}
	for _, p := range gdpr.Tables {
		registry[p.Name] = p
	}

	for name, infra := range infraTables {
		reg, ok := registry[name]
		if !ok {
			t.Errorf("infra defines table %q but gdpr.Tables has no entry — add one to internal/gdpr/storage.go", name)
			continue
		}
		if reg.SSE != infra.SSE {
			t.Errorf("table %q: SSE mismatch (gdpr=%v, infra=%v)", name, reg.SSE, infra.SSE)
		}
		if reg.PITR != infra.PITR {
			t.Errorf("table %q: PITR mismatch (gdpr=%v, infra=%v)", name, reg.PITR, infra.PITR)
		}
		if (reg.TTLAttribute == "") != (infra.TTLAttribute == "") {
			t.Errorf("table %q: TTL presence mismatch (gdpr=%q, infra=%q)", name, reg.TTLAttribute, infra.TTLAttribute)
		}
		if reg.TTLAttribute != "" && reg.TTLAttribute != infra.TTLAttribute {
			t.Errorf("table %q: TTL attribute mismatch (gdpr=%q, infra=%q)", name, reg.TTLAttribute, infra.TTLAttribute)
		}
	}

	for name := range registry {
		if _, ok := infraTables[name]; !ok {
			t.Errorf("gdpr.Tables has entry for %q but no matching infra table — remove the entry or restore the table", name)
		}
	}
}

type infraTable struct {
	SSE          bool
	PITR         bool
	TTLAttribute string
}

// extractInfraTables walks the AST of infra/database/database.go looking
// for dynamodb.NewTable(...) calls and reads:
//   - Name field (Pulumi.String(RunnersTableName)) — value resolved via the
//     file-level const declarations
//   - ServerSideEncryption: sseEnabled() — presence implies on
//   - PointInTimeRecovery: pitrEnabled() — presence implies on
//   - Ttl: &dynamodb.TableTtlArgs{ AttributeName: pulumi.String("...") }
//
// Anything we can't pattern-match falls back to false / empty so the test
// surfaces a real mismatch rather than a parse failure.
func extractInfraTables(path string) (map[string]infraTable, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	// First pass: collect file-level string constants. The Name field on
	// each NewTable call references one of these (e.g. RunnersTableName) so
	// we need the symbol → literal mapping to resolve table names.
	constants := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		gd, ok := n.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			return true
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok {
					continue
				}
				constants[name.Name] = strings.Trim(lit.Value, `"`)
			}
		}
		return true
	})

	out := map[string]infraTable{}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NewTable" {
			return true
		}
		if len(call.Args) < 3 {
			return true
		}
		// Third arg is `&dynamodb.TableArgs{...}` — unwrap the `&`.
		argsExpr := call.Args[2]
		if unary, ok := argsExpr.(*ast.UnaryExpr); ok {
			argsExpr = unary.X
		}
		argsLit, ok := argsExpr.(*ast.CompositeLit)
		if !ok {
			return true
		}
		name, posture := extractTableInfo(argsLit, constants)
		if name == "" {
			return true
		}
		out[name] = posture
		return true
	})
	return out, nil
}

func extractTableInfo(lit *ast.CompositeLit, constants map[string]string) (string, infraTable) {
	var name string
	posture := infraTable{}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Name":
			name = stringLiteralFromPulumiString(kv.Value, constants)
		case "ServerSideEncryption":
			// sseEnabled() — any non-nil expression here means SSE is on.
			if isFunctionCall(kv.Value, "sseEnabled") {
				posture.SSE = true
			}
		case "PointInTimeRecovery":
			if isFunctionCall(kv.Value, "pitrEnabled") {
				posture.PITR = true
			}
		case "Ttl":
			posture.TTLAttribute = ttlAttributeFromExpr(kv.Value, constants)
		}
	}
	return name, posture
}

// stringLiteralFromPulumiString handles both `pulumi.String("literal")` and
// `pulumi.String(ConstName)` by consulting the file's const map.
func stringLiteralFromPulumiString(expr ast.Expr, constants map[string]string) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	if len(call.Args) != 1 {
		return ""
	}
	switch v := call.Args[0].(type) {
	case *ast.BasicLit:
		return strings.Trim(v.Value, `"`)
	case *ast.Ident:
		return constants[v.Name]
	}
	return ""
}

func isFunctionCall(expr ast.Expr, name string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	if id, ok := call.Fun.(*ast.Ident); ok {
		return id.Name == name
	}
	return false
}

// ttlAttributeFromExpr expects `&dynamodb.TableTtlArgs{ AttributeName: pulumi.String("..."), ... }`.
func ttlAttributeFromExpr(expr ast.Expr, constants map[string]string) string {
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok {
		return ""
	}
	lit, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		return ""
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "AttributeName" {
			continue
		}
		return stringLiteralFromPulumiString(kv.Value, constants)
	}
	return ""
}

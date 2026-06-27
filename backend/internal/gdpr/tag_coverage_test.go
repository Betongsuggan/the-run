package gdpr_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// piiFieldNames lists struct-field names whose mere presence implies the
// field is personal data and therefore needs a gdpr:"..." tag. Imperfect
// (a field called Counter holding a phone number would slip through), but
// catches the most common "forgot to annotate Email/Name/BirthDate" failure
// mode.
var piiFieldNames = []string{
	"Email",
	"EmailAddress",
	"Name",
	"FullName",
	"FirstName",
	"LastName",
	"BirthDate",
	"DateOfBirth",
	"Gender",
	"PhoneNumber",
	"Phone",
	"Address",
	"PostalAddress",
	"PasswordHash",
	"Password",
	"MFASecret",
	"IP",
	"IPAddress",
	"SSN",
	"PersonalNumber",
	"Personnummer",
}

// knownNonPII excludes specific Model.Field pairs from the PII check. Each
// entry is a justified false-positive: the field name looks PII-shaped but
// the data actually held isn't personal data.
//
// Keep the list small and add a one-liner comment for each entry — if it
// grows unwieldy, the heuristic isn't working and we should add an
// explicit `gdpr:"non-pii"` opt-out tag instead.
var knownNonPII = map[string]bool{
	// Entity name, not a person's name.
	"Race.Name":  true,
	"Event.Name": true,
	// Derived gender bucket on the result-side Category struct; the
	// underlying value is tagged via Runner.Gender. Storing the bucket
	// on the Registration is denormalisation, not new PII.
	"Category.Gender": true,
}

// TestTagCoverage walks every struct in internal/models/*.go and flags any
// field whose name looks like PII (per piiFieldNames) but lacks a gdpr: tag.
// Catches the failure mode of adding a new field with a regulated name
// without annotating it.
//
// If the field is intentionally non-PII for some reason (e.g. an Address
// holding a server URL), refactor the name or add an explicit
// gdpr:"operational;purposes=..." annotation to make the intent visible.
func TestTagCoverage(t *testing.T) {
	backendDir, err := findBackendDir()
	if err != nil {
		t.Fatalf("locate backend dir: %v", err)
	}
	modelsDir := filepath.Join(backendDir, "internal", "models")

	piiSet := map[string]struct{}{}
	for _, n := range piiFieldNames {
		piiSet[n] = struct{}{}
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, modelsDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse models: %v", err)
	}

	var problems []string
	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}
				modelName := ts.Name.Name
				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						continue
					}
					fieldName := field.Names[0].Name
					if _, isPII := piiSet[fieldName]; !isPII {
						continue
					}
					if knownNonPII[modelName+"."+fieldName] {
						continue
					}
					tag := ""
					if field.Tag != nil {
						tag = reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Get("gdpr")
					}
					if tag == "" {
						problems = append(problems, formatProblem(filename, modelName, fieldName))
					}
				}
				return true
			})
		}
	}

	if len(problems) > 0 {
		t.Errorf("found %d PII-shaped field(s) without a gdpr: tag.\nAdd a tag (or rename the field if it isn't personal data):\n\n%s",
			len(problems), strings.Join(problems, "\n"))
	}
}

func formatProblem(filename, model, field string) string {
	return "  - " + filepath.Base(filename) + ": " + model + "." + field
}

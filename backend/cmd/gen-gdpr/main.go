// Command gen-gdpr renders docs/gdpr/ropa.md and docs/gdpr/dpia-screening.md
// from the gdpr:"..." struct tags on the model package + the registries in
// internal/gdpr. Run via `just gen-gdpr` or `go generate ./...`.
//
// Determinism matters: the drift test in internal/gdpr/drift_test.go re-runs
// the generator into a buffer and compares byte-for-byte against the committed
// files. So: no timestamps, no commit shas, no random ordering. Everything
// sorts via the explicit OrderedPurposeKeys / OrderedCategoryKeys lists.
//
// The DPIA scaffold round-trips operator-written prose between HTML comment
// markers (<!-- gdpr:prose:NAME start --> ... end -->). On re-run, existing
// prose between matching markers is preserved; everything else regenerates.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/BirgerRydback/the-run/backend/internal/gdpr"
)

func main() {
	modelsDir := flag.String("models", "internal/models", "Directory containing the model .go files")
	ropaOut := flag.String("ropa", "../docs/gdpr/ropa.md", "Output path for ROPA")
	dpiaOut := flag.String("dpia", "../docs/gdpr/dpia-screening.md", "Output path for DPIA screening")
	subprocOut := flag.String("subprocessors", "../docs/gdpr/subprocessors.md", "Output path for sub-processor list (B1.1)")
	retentionOut := flag.String("retention", "../docs/gdpr/retention.md", "Output path for retention schedule (B1.4)")
	flag.Parse()

	fields, err := loadAnnotatedFields(*modelsDir)
	if err != nil {
		log.Fatalf("parse model tags: %v", err)
	}
	if err := validate(fields); err != nil {
		log.Fatalf("validation: %v", err)
	}

	outputs := []struct {
		path string
		body string
	}{
		{*ropaOut, renderROPA(fields)},
		{*subprocOut, renderSubprocessors()},
		{*retentionOut, renderRetention()},
	}
	for _, o := range outputs {
		if err := os.MkdirAll(filepath.Dir(o.path), 0o755); err != nil {
			log.Fatalf("mkdir %s: %v", o.path, err)
		}
		if err := os.WriteFile(o.path, []byte(o.body), 0o644); err != nil {
			log.Fatalf("write %s: %v", o.path, err)
		}
		fmt.Printf("wrote %s (%d bytes)\n", o.path, len(o.body))
	}

	// DPIA is special — round-trips operator-written prose from the existing
	// file, so it has its own renderer signature.
	dpia, err := renderDPIA(fields, *dpiaOut)
	if err != nil {
		log.Fatalf("render dpia: %v", err)
	}
	if err := os.WriteFile(*dpiaOut, []byte(dpia), 0o644); err != nil {
		log.Fatalf("write dpia: %v", err)
	}
	fmt.Printf("wrote %s (%d bytes)\n", *dpiaOut, len(dpia))
}

// ── Annotation parsing ─────────────────────────────────────────────────

// fieldSpec is a single struct field carrying a gdpr:"..." tag.
type fieldSpec struct {
	Model    string // e.g. "Account"
	Field    string // e.g. "Email"
	Category string // e.g. "contact"
	Purposes []string
	Subject  string // optional override; empty = use purpose defaults
}

// loadAnnotatedFields walks every .go file in dir (non-recursive), looks at
// each struct field's tag, and returns the gdpr:-annotated ones.
func loadAnnotatedFields(dir string) ([]fieldSpec, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}

	var out []fieldSpec
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return true
				}
				modelName := typeSpec.Name.Name
				for _, field := range st.Fields.List {
					if field.Tag == nil || len(field.Names) == 0 {
						continue
					}
					// field.Tag.Value includes the surrounding backticks; strip
					// them before handing to reflect.StructTag.Get.
					raw := field.Tag.Value
					raw = strings.Trim(raw, "`")
					tag := reflect.StructTag(raw).Get("gdpr")
					if tag == "" {
						continue
					}
					spec, err := parseTag(tag)
					if err != nil {
						log.Fatalf("invalid gdpr tag on %s.%s: %v", modelName, field.Names[0].Name, err)
					}
					spec.Model = modelName
					spec.Field = field.Names[0].Name
					out = append(out, spec)
				}
				return true
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Model != out[j].Model {
			return out[i].Model < out[j].Model
		}
		return out[i].Field < out[j].Field
	})
	return out, nil
}

// parseTag interprets `<category>;purposes=<key>[,<key>]*[;<option>=<value>]*`.
func parseTag(raw string) (fieldSpec, error) {
	parts := strings.Split(raw, ";")
	if len(parts) < 2 {
		return fieldSpec{}, fmt.Errorf("expected at least category;purposes=..., got %q", raw)
	}
	spec := fieldSpec{Category: strings.TrimSpace(parts[0])}
	for _, p := range parts[1:] {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return fieldSpec{}, fmt.Errorf("malformed segment %q", p)
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case "purposes":
			for _, k := range strings.Split(val, ",") {
				k = strings.TrimSpace(k)
				if k != "" {
					spec.Purposes = append(spec.Purposes, k)
				}
			}
		case "subject":
			spec.Subject = val
		default:
			return fieldSpec{}, fmt.Errorf("unknown option %q", key)
		}
	}
	if spec.Category == "" {
		return fieldSpec{}, fmt.Errorf("missing category")
	}
	if len(spec.Purposes) == 0 {
		return fieldSpec{}, fmt.Errorf("missing purposes=")
	}
	return spec, nil
}

// validate ensures every category and purpose referenced by a tag actually
// exists in the registry. Catches typos at generate-time instead of in CI.
func validate(fields []fieldSpec) error {
	for _, f := range fields {
		if _, ok := gdpr.Categories[f.Category]; !ok {
			return fmt.Errorf("unknown category %q on %s.%s", f.Category, f.Model, f.Field)
		}
		for _, p := range f.Purposes {
			if _, ok := gdpr.Purposes[p]; !ok {
				return fmt.Errorf("unknown purpose %q on %s.%s", p, f.Model, f.Field)
			}
		}
	}
	return nil
}

// ── ROPA rendering ─────────────────────────────────────────────────────

func renderROPA(fields []fieldSpec) string {
	var b strings.Builder
	b.WriteString("# Records of Processing Activities (ROPA)\n\n")
	b.WriteString("> **Auto-generated.** To make changes, edit `gdpr:` struct tags\n")
	b.WriteString("> in `backend/internal/models/` or the registries in\n")
	b.WriteString("> `backend/internal/gdpr/`, then run `just gen-gdpr`. Do not edit\n")
	b.WriteString("> this file by hand — the drift test in CI will reject any change\n")
	b.WriteString("> that doesn't match what the generator would produce.\n\n")
	b.WriteString("**Personuppgiftsansvarig:** Ingmarsöloppet  \n")
	b.WriteString("**Kontakt:** kontakt@ingmarsoloppet.se  \n")
	b.WriteString("**Tillsynsmyndighet:** Integritetsskyddsmyndigheten (IMY)\n\n")

	b.WriteString("## Behandlingar\n\n")

	for idx, key := range gdpr.OrderedPurposeKeys {
		purpose, ok := gdpr.Purposes[key]
		if !ok {
			continue
		}
		matching := fieldsForPurpose(fields, key)
		fmt.Fprintf(&b, "### %d. %s (`%s`)\n\n", idx+1, firstSentence(purpose.DescSv), purpose.Key)
		fmt.Fprintf(&b, "%s\n\n", purpose.DescSv)
		fmt.Fprintf(&b, "- **Rättslig grund:** %s\n", purpose.Basis)
		fmt.Fprintf(&b, "- **Kategorier av registrerade:** %s\n", subjectsForPurpose(purpose, matching))

		if len(matching) > 0 {
			b.WriteString("- **Kategorier av personuppgifter:**\n")
			byCategory := groupByCategory(matching)
			for _, catKey := range gdpr.OrderedCategoryKeys {
				rows, ok := byCategory[catKey]
				if !ok {
					continue
				}
				cat := gdpr.Categories[catKey]
				var refs []string
				for _, r := range rows {
					refs = append(refs, fmt.Sprintf("`%s.%s`", r.Model, r.Field))
				}
				fmt.Fprintf(&b, "  - %s: %s\n", cat.DescSv, strings.Join(refs, ", "))
			}
		} else {
			b.WriteString("- **Kategorier av personuppgifter:** (denna behandling konsumerar inga taggade fält direkt — den styr cookies, sessioner eller skyddsåtgärder)\n")
		}

		if ret, ok := gdpr.Retentions[purpose.Retention]; ok {
			fmt.Fprintf(&b, "- **Lagringstid:** %s\n", ret.DescSv)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Underbiträden\n\n")
	b.WriteString("| Namn | Roll | Region | Tjänster | DPA |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, sp := range gdpr.Subprocessors {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			sp.Name, sp.Role, sp.Region, strings.Join(sp.Services, ", "), sp.DPA)
	}
	b.WriteString("\n")

	b.WriteString("## Tabeller och lagringsläge\n\n")
	b.WriteString("| Tabell | SSE | PITR | TTL | Innehåll |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, t := range gdpr.Tables {
		ttl := "–"
		if t.TTLAttribute != "" {
			ttl = "`" + t.TTLAttribute + "`"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n",
			t.Name, yesNo(t.SSE), yesNo(t.PITR), ttl, t.DescSv)
	}
	b.WriteString("\n")

	b.WriteString("## Tekniska och organisatoriska säkerhetsåtgärder\n\n")
	for _, m := range gdpr.SecurityMeasures {
		fmt.Fprintf(&b, "- %s\n", m.DescSv)
	}
	b.WriteString("\n")

	b.WriteString("## Överföringar till tredje land\n\n")
	b.WriteString("Inga. All data lagras i AWS region eu-north-1 (Stockholm) och distribueras via CloudFront-edge inom EES.\n")

	return b.String()
}

func fieldsForPurpose(fields []fieldSpec, purposeKey string) []fieldSpec {
	var out []fieldSpec
	for _, f := range fields {
		for _, p := range f.Purposes {
			if p == purposeKey {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

func groupByCategory(fields []fieldSpec) map[string][]fieldSpec {
	out := map[string][]fieldSpec{}
	for _, f := range fields {
		out[f.Category] = append(out[f.Category], f)
	}
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool {
			if out[k][i].Model != out[k][j].Model {
				return out[k][i].Model < out[k][j].Model
			}
			return out[k][i].Field < out[k][j].Field
		})
	}
	return out
}

// subjectsForPurpose unions the purpose's default subjects with any per-field
// overrides. If every matching field has the same single subject override,
// that wins; otherwise we use the purpose default.
func subjectsForPurpose(p gdpr.Purpose, matching []fieldSpec) string {
	set := map[string]struct{}{}
	overridden := false
	for _, f := range matching {
		if f.Subject != "" {
			set[f.Subject] = struct{}{}
			overridden = true
		}
	}
	if !overridden {
		for _, s := range p.Subjects {
			set[string(s)] = struct{}{}
		}
	}
	if len(set) == 0 {
		for _, s := range p.Subjects {
			set[string(s)] = struct{}{}
		}
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func firstSentence(s string) string {
	// "Anmälan och löpadministration — startlista, ..." → "Anmälan och löpadministration"
	if i := strings.Index(s, " — "); i > 0 {
		return s[:i]
	}
	if i := strings.Index(s, ". "); i > 0 {
		return s[:i]
	}
	return s
}

func yesNo(b bool) string {
	if b {
		return "✓"
	}
	return "–"
}

// ── Sub-processor list (B1.1) ──────────────────────────────────────────

// renderSubprocessors emits docs/gdpr/subprocessors.md. Same data as the
// "Underbiträden" section in the ROPA but stands alone as a discoverable
// artefact (linkable from the privacy policy, easy to share on request).
func renderSubprocessors() string {
	var b strings.Builder
	b.WriteString("# Underbiträden\n\n")
	b.WriteString("> **Auto-generated** from `backend/internal/gdpr/subprocessors.go`.\n")
	b.WriteString("> Edit that file and run `just gen-gdpr` to refresh.\n\n")
	b.WriteString("Ingmarsöloppet anlitar följande underbiträden för att driva tjänsten. ")
	b.WriteString("Inga personuppgifter delas med någon utöver dessa.\n\n")

	for _, sp := range gdpr.Subprocessors {
		fmt.Fprintf(&b, "## %s\n\n", sp.Name)
		fmt.Fprintf(&b, "- **Roll:** %s\n", sp.Role)
		fmt.Fprintf(&b, "- **Region:** %s\n", sp.Region)
		fmt.Fprintf(&b, "- **Tjänster som används:** %s\n", strings.Join(sp.Services, ", "))
		fmt.Fprintf(&b, "- **Personuppgiftsbiträdesavtal (DPA):** %s\n\n", sp.DPA)
	}

	b.WriteString("## Överföringar till tredje land\n\n")
	b.WriteString("Inga. All databehandling sker inom EES — primär region är ")
	b.WriteString("AWS eu-north-1 (Stockholm); statiska sidor distribueras via ")
	b.WriteString("CloudFront-edge inom EES.\n\n")

	b.WriteString("## Granskningsrytm\n\n")
	b.WriteString("Den här listan granskas:\n\n")
	b.WriteString("- Vid varje ny extern tjänst som integreras (PR-granskning).\n")
	b.WriteString("- Årligen som del av den allmänna policygranskningen.\n")
	b.WriteString("- Före publicering av en ny version av sekretesspolicyn ")
	b.WriteString("(`/admin/policies` → ny version), så att policytexten matchar listan.\n")

	return b.String()
}

// ── Retention schedule (B1.4) ──────────────────────────────────────────

// renderRetention emits docs/gdpr/retention.md from gdpr.Retentions +
// gdpr.Tables. Reads as a single-source operator reference for "how long
// do we keep X", deduplicating the windows scattered across cmd/retention/,
// internal/api/, and internal/auth/.
func renderRetention() string {
	var b strings.Builder
	b.WriteString("# Lagringsschema\n\n")
	b.WriteString("> **Auto-generated** från `backend/internal/gdpr/retention.go` + ")
	b.WriteString("`backend/internal/gdpr/storage.go`. Edit those files och kör ")
	b.WriteString("`just gen-gdpr` för att uppdatera.\n\n")
	b.WriteString("Det här schemat samlar alla lagringstider för personuppgifter ")
	b.WriteString("i tjänsten. Varje rad pekar tillbaka på en Go-konstant som styr ")
	b.WriteString("den faktiska beteendet — så schemat och koden kan inte drifta ifrån varandra.\n\n")

	b.WriteString("## Lagringstider per kategori\n\n")
	b.WriteString("| Nyckel | Beskrivning | Källa i kod |\n")
	b.WriteString("|---|---|---|\n")
	keys := make([]string, 0, len(gdpr.Retentions))
	for k := range gdpr.Retentions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		r := gdpr.Retentions[k]
		var source string
		switch {
		case r.Months > 0:
			source = fmt.Sprintf("%d månader", r.Months)
		case r.Duration > 0:
			source = r.Duration.String()
		default:
			source = "tills handling (samtycke återkallat, konto raderat, etc.)"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", k, r.DescSv, source)
	}
	b.WriteString("\n")

	b.WriteString("## Lagring per DynamoDB-tabell\n\n")
	b.WriteString("Tabeller med automatisk TTL får raderna automatiskt borttagna av DynamoDB. ")
	b.WriteString("Övriga tabeller städas av en daglig retention-Lambda (`backend/cmd/retention/`).\n\n")
	b.WriteString("| Tabell | TTL-attribut | PITR (35d) | Hanteras av |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, t := range gdpr.Tables {
		ttl := "–"
		mech := "retention-Lambda + DSR-flöde"
		if t.TTLAttribute != "" {
			ttl = "`" + t.TTLAttribute + "`"
			mech = "DynamoDB TTL (automatiskt)"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", t.Name, ttl, yesNo(t.PITR), mech)
	}
	b.WriteString("\n")

	b.WriteString("## Lagringskedjan från löparens perspektiv\n\n")
	b.WriteString("1. **Anmälan utan start (DNS/no-show):** raderas 6 månader efter loppdatumet.\n")
	b.WriteString("2. **Genomfört lopp + tillåtelse till publika resultat:** sparas tills vidare som löparhistorik.\n")
	b.WriteString("3. **Genomfört lopp utan tillåtelse till publika resultat:** anonymiseras 36 månader efter senaste loppet (rullande — klockan börjar om vid varje nytt lopp).\n")
	b.WriteString("4. **Konto utan kvarvarande löpare:** raderas 30 dagar efter att sista löparen tagits bort (ångerfönster).\n")
	b.WriteString("5. **Inaktivt konto:** mjuk-raderas efter 36 månader utan aktivitet (inloggning, anmälan, lopp). Användaren mejlas och har 30 dagar att återställa.\n")
	b.WriteString("6. **Adminaktivitetslogg:** 24 månader.\n")
	b.WriteString("7. **Säkerhetsloggar (CloudWatch):** 30 dagar.\n")
	b.WriteString("8. **Säkerhetskopior (DynamoDB PITR):** 35 dagar på PII-bärande tabeller.\n\n")

	b.WriteString("## Hur ändras schemat?\n\n")
	b.WriteString("1. Uppdatera konstanten i `backend/internal/gdpr/retention.go`.\n")
	b.WriteString("2. Kör `just gen-gdpr` — den här filen + ROPA uppdateras automatiskt.\n")
	b.WriteString("3. Befintliga rader påverkas vid nästa retention-Lambda-körning (dagligen).\n")
	b.WriteString("4. Notera ändringen i sekretesspolicyn (publicera en ny version via `/admin/policies` om förändringen är väsentlig).\n")

	return b.String()
}

// ── DPIA scaffold with prose preservation ──────────────────────────────

var proseMarkerRE = regexp.MustCompile(`<!-- gdpr:prose:([a-z0-9-]+) start -->([\s\S]*?)<!-- gdpr:prose:[a-z0-9-]+ end -->`)

// renderDPIA emits the DPIA-screening scaffold. If outPath already exists,
// extract every prose block between matching markers and re-splice them
// into the regenerated template so the operator's risk-analysis prose
// survives re-runs.
func renderDPIA(fields []fieldSpec, outPath string) (string, error) {
	existing := loadExistingProse(outPath)

	prose := func(name string) string {
		if v, ok := existing[name]; ok {
			return v
		}
		return fmt.Sprintf("\n_TODO: hand-author this section. The generator preserves whatever you write between the two `gdpr:prose:%s` markers on re-run._\n", name)
	}
	block := func(name, body string) string {
		return fmt.Sprintf("<!-- gdpr:prose:%s start -->%s<!-- gdpr:prose:%s end -->", name, body, name)
	}

	var b strings.Builder
	b.WriteString("# DPIA-screening — Ingmarsöloppet\n\n")
	b.WriteString("> Partially auto-generated. The data-flow + categories sections come\n")
	b.WriteString("> from struct tags in `backend/internal/models/` (re-run `just gen-gdpr`\n")
	b.WriteString("> to refresh). Risk-analysis prose between `<!-- gdpr:prose:NAME -->`\n")
	b.WriteString("> markers is preserved across regenerations — that's where the operator\n")
	b.WriteString("> writes the judgement calls.\n\n")

	b.WriteString("## 1. Översikt\n")
	b.WriteString(block("overview", prose("overview")))
	b.WriteString("\n\n")

	b.WriteString("## 2. Vad behandlas? (auto-genererad inventering)\n\n")
	b.WriteString("Följande personuppgiftskategorier behandlas, härledda från `gdpr:`-taggar på modellfält:\n\n")
	byCategory := groupByCategory(fields)
	for _, catKey := range gdpr.OrderedCategoryKeys {
		rows, ok := byCategory[catKey]
		if !ok {
			continue
		}
		cat := gdpr.Categories[catKey]
		fmt.Fprintf(&b, "- **%s:**", cat.DescSv)
		var refs []string
		for _, r := range rows {
			refs = append(refs, fmt.Sprintf("`%s.%s`", r.Model, r.Field))
		}
		fmt.Fprintf(&b, " %s\n", strings.Join(refs, ", "))
	}
	b.WriteString("\n")

	b.WriteString("## 3. För vilka ändamål? (auto-genererad)\n\n")
	for _, key := range gdpr.OrderedPurposeKeys {
		p, ok := gdpr.Purposes[key]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "- **%s** — %s _(rättslig grund: %s)_\n", firstSentence(p.DescSv), p.DescSv, p.Basis)
	}
	b.WriteString("\n")

	b.WriteString("## 4. Berörda registrerade och särskilda kategorier\n")
	b.WriteString(block("subjects", prose("subjects")))
	b.WriteString("\n\n")

	b.WriteString("## 5. Nödvändighet och proportionalitet\n")
	b.WriteString(block("necessity", prose("necessity")))
	b.WriteString("\n\n")

	b.WriteString("## 6. Identifierade risker\n")
	b.WriteString(block("risks", prose("risks")))
	b.WriteString("\n\n")

	b.WriteString("## 7. Åtgärder för att minska risker\n\n")
	b.WriteString("Tekniska åtgärder (auto-genererade från `internal/gdpr/storage.go`):\n\n")
	for _, m := range gdpr.SecurityMeasures {
		fmt.Fprintf(&b, "- %s\n", m.DescSv)
	}
	b.WriteString("\nOrganisatoriska och kompletterande åtgärder:\n")
	b.WriteString(block("mitigations", prose("mitigations")))
	b.WriteString("\n\n")

	b.WriteString("## 8. Slutsats — krävs full DPIA?\n")
	b.WriteString(block("conclusion", prose("conclusion")))
	b.WriteString("\n")

	return b.String(), nil
}

// loadExistingProse parses outPath (if it exists) and returns the contents
// of every gdpr:prose block keyed by name.
func loadExistingProse(outPath string) map[string]string {
	data, err := os.ReadFile(outPath)
	if err != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	matches := proseMarkerRE.FindAllStringSubmatch(string(data), -1)
	for _, m := range matches {
		name := m[1]
		body := m[2]
		out[name] = body
	}
	return out
}

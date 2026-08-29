package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// routeDataDocsPath is the reference page that documents the template context.
const routeDataDocsPath = "../../docs/content/docs/reference/route-data.md"

// documentedTypes lists every type the Route Data page has a section for.
// Adding an exported field to one of these without a table row fails the test.
var documentedTypes = []any{
	RouteData{}, Page{}, SiteContext{}, Collection{}, CollectionConfig{},
	SidebarConfig{}, TOCConfig{}, PrevNextConfig{}, LabsConfig{}, DocsTab{},
	Section{}, NavTree{}, NavNode{}, GlobalNav{}, GlobalNavItem{},
	BreadcrumbItem{}, PaginationLinks{}, PaginationLink{}, Paginator{},
	Taxonomy{}, TaxonomyTerm{}, TermEntry{}, HomepageData{}, HeroData{},
	HeroCTAData{}, HeroStatData{}, HeroCodeData{}, HeroImageData{},
	TranslationLink{}, VersionLink{}, VersionConfig{}, VersionDef{},
	Heading{}, Resource{}, Badge{}, PageBanner{}, ThemeConfig{}, Language{},
	LogoContext{}, LogoImage{}, IconLicense{}, PageSidebar{}, PageTOC{},
}

// TestRouteDataDocs_CoverEveryField is a drift guard between the template
// context structs and docs/content/docs/reference/route-data.md.
func TestRouteDataDocs_CoverEveryField(t *testing.T) {
	doc := readRouteDataDocs(t)
	for _, v := range documentedTypes {
		typ := reflect.TypeOf(v)
		section := docSection(doc, typ.Name())
		if section == "" {
			t.Errorf("%s: no heading naming the type (expected a heading containing `%s`)", typ.Name(), typ.Name())
			continue
		}
		for _, name := range promotedFieldNames(typ) {
			if !mentionsIdentifier(section, name) {
				t.Errorf("%s.%s is not documented in the %s section of %s", typ.Name(), name, typ.Name(), filepath.Base(routeDataDocsPath))
			}
		}
	}
}

// TestRouteDataDocs_NoPhantomFields checks the reverse direction for the two
// roots: every `.Name` and `.Page.Name` accessor written in their sections
// must exist on the struct (or be a method), so a typo in the docs fails too.
func TestRouteDataDocs_NoPhantomFields(t *testing.T) {
	doc := readRouteDataDocs(t)

	root := docSection(doc, "RouteData")
	rootFields := fieldSet(reflect.TypeOf(RouteData{}))
	for _, m := range regexp.MustCompile("`\\.([A-Z]\\w*)").FindAllStringSubmatch(root, -1) {
		if !rootFields[m[1]] {
			t.Errorf("RouteData section mentions `.%s`, which is not a RouteData field", m[1])
		}
	}

	page := docSection(doc, "Page")
	pageFields := fieldSet(reflect.TypeOf(Page{}))
	pageType := reflect.TypeOf(&Page{})
	for _, m := range regexp.MustCompile("`\\.Page\\.([A-Z]\\w*)").FindAllStringSubmatch(page, -1) {
		if pageFields[m[1]] {
			continue
		}
		if _, ok := pageType.MethodByName(m[1]); ok {
			continue
		}
		t.Errorf("Page section mentions `.Page.%s`, which is neither a Page field nor a method", m[1])
	}
}

func readRouteDataDocs(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(routeDataDocsPath)
	if err != nil {
		t.Skipf("route data docs not available (%v); skipping drift guard", err)
	}
	return string(data)
}

// docSection returns the text from the first heading whose text contains the
// backticked type name up to the next heading of the same or higher level.
func docSection(doc, typeName string) string {
	lines := strings.Split(doc, "\n")
	needle := "`" + typeName + "`"
	for i, line := range lines {
		if !strings.HasPrefix(line, "#") || !strings.Contains(line, needle) {
			continue
		}
		level := len(line) - len(strings.TrimLeft(line, "#"))
		var sb strings.Builder
		for _, l := range lines[i+1:] {
			if strings.HasPrefix(l, "#") {
				if lv := len(l) - len(strings.TrimLeft(l, "#")); lv <= level {
					break
				}
			}
			sb.WriteString(l)
			sb.WriteByte('\n')
		}
		return sb.String()
	}
	return ""
}

// promotedFieldNames returns exported field names of typ, descending into
// anonymous embedded structs the way html/template promotes them.
func promotedFieldNames(typ reflect.Type) []string {
	var out []string
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if f.Anonymous && ft.Kind() == reflect.Struct && ft.PkgPath() == typ.PkgPath() {
			out = append(out, promotedFieldNames(ft)...)
			continue
		}
		out = append(out, f.Name)
	}
	return out
}

func fieldSet(typ reflect.Type) map[string]bool {
	set := make(map[string]bool)
	for _, n := range promotedFieldNames(typ) {
		set[n] = true
	}
	return set
}

// mentionsIdentifier reports whether text contains name as a whole word inside
// a backticked dotted path, for example `Title`, `.Page.Title`, or `.Sidebar.Root`.
func mentionsIdentifier(text, name string) bool {
	re := regexp.MustCompile("`[.\\w]*\\b" + regexp.QuoteMeta(name) + "\\b[.\\w]*`")
	return re.MatchString(text)
}

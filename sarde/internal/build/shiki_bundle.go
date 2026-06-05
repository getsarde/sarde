package build

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/frostybee/sarde/internal/asset"
	"github.com/frostybee/sarde/internal/engine"
)

var langClassRe = regexp.MustCompile(`\blanguage-(\w[\w+-]*)\b`)

func collectCodeLanguages(pages []*engine.Page) []string {
	seen := make(map[string]bool)
	for _, p := range pages {
		for _, m := range langClassRe.FindAllStringSubmatch(string(p.Content), -1) {
			seen[m[1]] = true
		}
	}
	delete(seen, "mermaid")

	langs := make([]string, 0, len(seen))
	for l := range seen {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	return langs
}

func generateShikiEntryPoint(langs []string, lightTheme, darkTheme string) string {
	var b strings.Builder

	b.WriteString("import{createHighlighterCore}from'@shikijs/core';\n")
	b.WriteString("import{createJavaScriptRegexEngine}from'@shikijs/engine-javascript';\n")

	for _, lang := range langs {
		varName := sanitizeVarName(lang)
		b.WriteString(fmt.Sprintf("import %s from'@shikijs/langs/%s';\n", varName, lang))
	}

	b.WriteString(fmt.Sprintf("import themeLight from'@shikijs/themes/%s';\n", lightTheme))
	b.WriteString(fmt.Sprintf("import themeDark from'@shikijs/themes/%s';\n", darkTheme))

	b.WriteString("\nconst langs=[")
	for i, lang := range langs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(sanitizeVarName(lang))
	}
	b.WriteString("];\n")

	b.WriteString(fmt.Sprintf("const LIGHT='%s',DARK='%s';\n\n", lightTheme, darkTheme))

	b.WriteString(`const hl=await createHighlighterCore({
  engine:createJavaScriptRegexEngine(),
  langs,themes:[themeLight,themeDark],
});
const st=document.createElement('style');
st.textContent='[data-enhanced="shiki"] .sarde-code-line span[style]{color:var(--sl,inherit)}:root.dark [data-enhanced="shiki"] .sarde-code-line span[style]{color:var(--sd,inherit)}';
document.head.appendChild(st);
function esc(s){return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');}
function vars(o){
  if(!o)return'';
  if(typeof o==='string')return o;
  const light=o.color||'';
  const dark=o['--shiki-dark']||'';
  let r='';
  if(light)r+='--sl:'+light;
  if(dark)r+=(r?';':'')+'--sd:'+dark;
  return r;
}
document.querySelectorAll('code.sarde-chroma-code[class*="language-"]').forEach(el=>{
  const m=el.className.match(/language-(\S+)/);
  if(!m)return;
  const lang=m[1];
  if(!hl.getLoadedLanguages().includes(lang))return;
  const{tokens}=hl.codeToTokens(el.textContent,{
    lang,themes:{light:LIGHT,dark:DARK},
  });
  const lines=el.querySelectorAll('.sarde-code-line');
  tokens.forEach((lt,i)=>{
    if(i>=lines.length)return;
    lines[i].innerHTML=lt.map(t=>{
      const v=vars(t.htmlStyle);
      return v?'<span style="'+v+'">'+esc(t.content)+'</span>':'<span>'+esc(t.content)+'</span>';
    }).join('');
  });
  el.closest('.sarde-code-block')?.setAttribute('data-enhanced','shiki');
});
`)
	return b.String()
}

func sanitizeVarName(lang string) string {
	s := strings.ReplaceAll(lang, "-", "_")
	s = strings.ReplaceAll(s, "+", "p")
	s = strings.ReplaceAll(s, "#", "sharp")
	return "lang_" + s
}

func BundleShikiJS(vendorFS fs.FS, langs []string, lightTheme, darkTheme string, devMode bool) ([]byte, string, error) {
	if len(langs) == 0 {
		return nil, "", nil
	}

	langs = filterAvailableLangs(vendorFS, langs)

	tmpDir, err := os.MkdirTemp("", "sarde-shiki-*")
	if err != nil {
		return nil, "", fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	nodeModules := filepath.Join(tmpDir, "node_modules")
	if err := extractVendorToNodeModules(vendorFS, "shiki", nodeModules); err != nil {
		return nil, "", fmt.Errorf("extracting vendor: %w", err)
	}

	entrySource := generateShikiEntryPoint(langs, lightTheme, darkTheme)
	entryPath := filepath.Join(tmpDir, "shiki-enhance.mjs")
	if err := os.WriteFile(entryPath, []byte(entrySource), 0o644); err != nil {
		return nil, "", fmt.Errorf("writing entry point: %w", err)
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryPath},
		Bundle:      true,
		Format:      api.FormatESModule,
		Platform:    api.PlatformBrowser,
		Target:      api.ES2022,
		Write:       false,
		NodePaths:   []string{nodeModules},
		MinifyWhitespace:  !devMode,
		MinifyIdentifiers: !devMode,
		MinifySyntax:      !devMode,
		LogLevel:    api.LogLevelWarning,
	})
	if len(result.Errors) > 0 {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, e.Text)
		}
		return nil, "", fmt.Errorf("esbuild errors: %s", strings.Join(msgs, "; "))
	}
	if len(result.OutputFiles) == 0 {
		return nil, "", fmt.Errorf("esbuild produced no output")
	}

	content := result.OutputFiles[0].Contents
	hash := asset.Fingerprint(content)
	var filename string
	if devMode {
		filename = "shiki-enhance.js"
	} else {
		filename = asset.FingerprintedName("shiki-enhance.js", hash)
	}

	return content, filename, nil
}

func filterAvailableLangs(vendorFS fs.FS, langs []string) []string {
	var available []string
	for _, lang := range langs {
		path := fmt.Sprintf("shiki/@shikijs/langs/dist/%s.mjs", lang)
		if _, err := fs.Stat(vendorFS, path); err == nil {
			available = append(available, lang)
		}
	}
	return available
}

func extractVendorToNodeModules(vendorFS fs.FS, srcPrefix, destDir string) error {
	return fs.WalkDir(vendorFS, srcPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcPrefix, path)
		if err != nil || rel == "." {
			return nil
		}

		dest := filepath.Join(destDir, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}

		data, err := fs.ReadFile(vendorFS, path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o644)
	})
}

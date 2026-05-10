use regex::Regex;
use std::path::{Component, Path, PathBuf};
use std::sync::LazyLock;

/// Split a markdown file into frontmatter (YAML) and body.
/// Supports `---` delimited YAML frontmatter.
/// Returns (frontmatter_value, body_string). If no frontmatter found, returns (Null, full content).
pub fn parse_frontmatter(raw: &str) -> (serde_yaml::Value, String) {
    let content = raw.trim_start_matches('\u{feff}'); // strip BOM
    let content = content.trim_start_matches('\n').trim_start_matches('\r');

    if content.is_empty() {
        return (serde_yaml::Value::Mapping(Default::default()), String::new());
    }

    if let Some(rest) = content.strip_prefix("---") {
        // Skip the newline after opening delimiter
        let rest = match rest.find('\n') {
            Some(idx) => &rest[idx + 1..],
            None => return (serde_yaml::Value::Mapping(Default::default()), content.to_string()),
        };

        // Find closing ---
        if let Some(closing_idx) = rest.find("\n---") {
            let fm_text = &rest[..closing_idx];
            let mut body = &rest[closing_idx + 4..]; // skip "\n---"

            // Strip leading newline from body
            if let Some(b) = body.strip_prefix('\n') {
                body = b;
            } else if let Some(b) = body.strip_prefix("\r\n") {
                body = b;
            }

            match serde_yaml::from_str::<serde_yaml::Value>(fm_text) {
                Ok(val) => return (val, body.to_string()),
                Err(_) => return (serde_yaml::Value::Mapping(Default::default()), content.to_string()),
            }
        }
    }

    // No frontmatter found
    (serde_yaml::Value::Mapping(Default::default()), content.to_string())
}

/// Serialize frontmatter and body back into a markdown file string.
pub fn serialize_frontmatter(frontmatter: &serde_yaml::Value, body: &str) -> String {
    let mut sb = String::new();

    let has_fields = match frontmatter {
        serde_yaml::Value::Mapping(m) => !m.is_empty(),
        _ => false,
    };

    if has_fields {
        sb.push_str("---\n");
        if let Ok(yaml_str) = serde_yaml::to_string(frontmatter) {
            sb.push_str(&yaml_str);
        }
        sb.push_str("---\n");
    }

    sb.push_str(body);
    sb
}

static NON_ALNUM_HYPHEN: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"[^a-z0-9-]+").unwrap());
static MULTI_HYPHEN: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"-{2,}").unwrap());

/// Generate a URL-safe slug from a title.
pub fn slugify(title: &str) -> String {
    let s = title.to_lowercase();
    let s = s.replace(' ', "-");
    let s = NON_ALNUM_HYPHEN.replace_all(&s, "");
    let s = MULTI_HYPHEN.replace_all(&s, "-");
    s.trim_matches('-').to_string()
}

/// Extract summary metadata from parsed frontmatter and body.
pub fn extract_summary(
    fm: &serde_yaml::Value,
    body: &str,
) -> (String, bool, String, i64, usize, usize) {
    let title = fm
        .get("title")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string();

    let draft = fm
        .get("draft")
        .and_then(|v| v.as_bool())
        .unwrap_or(false);

    let date = fm
        .get("date")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string();

    let weight = fm
        .get("weight")
        .and_then(|v| v.as_i64())
        .unwrap_or(0);

    let word_count = body.split_whitespace().count();
    let reading_time = if word_count > 0 {
        ((word_count as f64 / 200.0).ceil() as usize).max(1)
    } else {
        0
    };

    (title, draft, date, weight, word_count, reading_time)
}

/// Validate that a relative path is safe (no traversal, within content dir).
pub fn validate_content_path(rel_path: &str) -> Result<(), String> {
    if rel_path.is_empty() {
        return Err("empty path".into());
    }

    let cleaned = rel_path.replace('\\', "/");
    if cleaned.starts_with('/') || cleaned.starts_with('~') {
        return Err(format!("absolute paths are not allowed: {}", rel_path));
    }
    if cleaned.contains(':') {
        return Err(format!("drive or scheme paths are not allowed: {}", rel_path));
    }
    if cleaned.split('/').any(|part| part.is_empty() || part == ".") {
        return Err(format!("empty path segments are not allowed: {}", rel_path));
    }
    if cleaned.split('/').any(|part| part == "..") {
        return Err(format!("path traversal not allowed: {}", rel_path));
    }

    let path = Path::new(rel_path);
    if path.is_absolute() {
        return Err(format!("absolute paths are not allowed: {}", rel_path));
    }
    for component in path.components() {
        match component {
            Component::Normal(_) => {}
            _ => return Err(format!("invalid path component: {}", rel_path)),
        }
    }

    Ok(())
}

/// Join a validated relative path to a base directory and prove it cannot escape the base.
pub fn safe_join(base: &Path, rel_path: &str, target_must_exist: bool) -> Result<PathBuf, String> {
    validate_content_path(rel_path)?;

    let base_canonical = base
        .canonicalize()
        .map_err(|e| format!("Resolving base directory: {}", e))?;
    let candidate = base_canonical.join(rel_path.replace('/', std::path::MAIN_SEPARATOR_STR));

    if target_must_exist {
        let candidate_canonical = candidate
            .canonicalize()
            .map_err(|e| format!("Resolving path: {}", e))?;
        if !candidate_canonical.starts_with(&base_canonical) {
            return Err("Path traversal not allowed".into());
        }
        return Ok(candidate_canonical);
    }

    let parent = candidate
        .parent()
        .ok_or_else(|| "Invalid path parent".to_string())?;
    let parent_canonical = parent
        .canonicalize()
        .map_err(|e| format!("Resolving parent directory: {}", e))?;
    if !parent_canonical.starts_with(&base_canonical) {
        return Err("Path traversal not allowed".into());
    }

    Ok(candidate)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_frontmatter_yaml() {
        let input = "---\ntitle: Hello\ndraft: true\n---\nBody here\n";
        let (fm, body) = parse_frontmatter(input);
        assert_eq!(fm.get("title").unwrap().as_str().unwrap(), "Hello");
        assert_eq!(fm.get("draft").unwrap().as_bool().unwrap(), true);
        assert_eq!(body, "Body here\n");
    }

    #[test]
    fn test_parse_frontmatter_none() {
        let input = "Just some markdown\n";
        let (fm, body) = parse_frontmatter(input);
        assert!(fm.as_mapping().unwrap().is_empty());
        assert_eq!(body, "Just some markdown\n");
    }

    #[test]
    fn test_slugify() {
        assert_eq!(slugify("Hello World"), "hello-world");
        assert_eq!(slugify("My  Great--Post!"), "my-great-post");
        assert_eq!(slugify("  Leading and Trailing  "), "leading-and-trailing");
    }

    #[test]
    fn test_serialize_roundtrip() {
        let input = "---\ntitle: Test\n---\nBody\n";
        let (fm, body) = parse_frontmatter(input);
        let output = serialize_frontmatter(&fm, &body);
        assert!(output.contains("title"));
        assert!(output.contains("Body"));
    }

    #[test]
    fn test_validate_content_path() {
        assert!(validate_content_path("blog/post.md").is_ok());
        assert!(validate_content_path("blog\\post.md").is_ok());
        assert!(validate_content_path("../etc/passwd").is_err());
        assert!(validate_content_path("blog/../post.md").is_err());
        assert!(validate_content_path("/blog/post.md").is_err());
        assert!(validate_content_path("C:/tmp/post.md").is_err());
        assert!(validate_content_path("blog//post.md").is_err());
        assert!(validate_content_path("./blog/post.md").is_err());
        assert!(validate_content_path("").is_err());
    }
}

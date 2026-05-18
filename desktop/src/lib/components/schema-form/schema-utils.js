// Schema utilities for the SchemaForm component.
// Pure functions — no Svelte reactivity, no side effects.

/** Map Go/YAML type aliases to canonical widget types. */
export const CANONICAL_TYPE = {
  string: 'string', text: 'string',
  textarea: 'textarea',
  int: 'int',
  float: 'float', number: 'float',
  bool: 'bool', toggle: 'bool',
  date: 'date',
  list: 'list', tags: 'list',
  enum: 'enum', select: 'enum',
  color: 'color',
  image: 'image',
  url: 'string',
}

/**
 * Infer a canonical widget type from a JS value when no schema exists.
 * @param {string} key - field name (unused, reserved for future heuristics)
 * @param {any} value
 * @returns {string}
 */
export function inferType(key, value) {
  if (typeof value === 'boolean') return 'bool'
  if (typeof value === 'number') return Number.isInteger(value) ? 'int' : 'float'
  if (Array.isArray(value)) return 'list'
  if (typeof value === 'string') {
    if (/^\d{4}-\d{2}-\d{2}(T|\s)/.test(value)) return 'date'
    if (value.length > 80) return 'textarea'
    return 'string'
  }
  return 'string'
}

/**
 * Merge schema field definitions with inferred types for values not covered by schema.
 * Returns a flat record of normalized field defs keyed by field name.
 *
 * @param {Record<string, object>} schemaFields - from fetchSchema().fields (may be empty)
 * @param {Record<string, any>} values - current frontmatter values
 * @returns {Record<string, {type: string, label: string, required: boolean, options: string[], min?: number, max?: number, maxLength?: number}>}
 */
export function normalizeFields(schemaFields, values) {
  const result = {}
  const allKeys = new Set([...Object.keys(schemaFields || {}), ...Object.keys(values || {})])

  for (const key of allKeys) {
    const schemaDef = schemaFields?.[key] || {}
    const rawType = schemaDef.type || inferType(key, values?.[key])
    const type = CANONICAL_TYPE[rawType] ?? 'string'

    result[key] = {
      type,
      label: schemaDef.label || humanizeKey(key),
      required: schemaDef.required || false,
      options: schemaDef.options || [],
      min: schemaDef.min,
      max: schemaDef.max,
      maxLength: schemaDef.maxLength,
      default: schemaDef.default,
    }
  }

  return result
}

/** Convert a camelCase or snake_case key to a human-readable label. */
export function humanizeKey(key) {
  return key
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/[_-]/g, ' ')
    .replace(/\b\w/g, c => c.toUpperCase())
}

/** Priority ordering for common frontmatter fields. */
const PRIORITY = { title: 0, draft: 1, date: 2 }

/**
 * Sort field keys: explicit order first, then priority, then alphabetical.
 * @param {string[]} keys
 * @param {string[]|null} explicitOrder - if provided, use this ordering (filter to existing keys)
 * @returns {string[]}
 */
export function sortFieldKeys(keys, explicitOrder = null) {
  if (explicitOrder) {
    const set = new Set(keys)
    const ordered = explicitOrder.filter(k => set.has(k))
    const remaining = keys.filter(k => !explicitOrder.includes(k))
    return [...ordered, ...remaining.sort((a, b) => a.localeCompare(b))]
  }

  return [...keys].sort((a, b) => {
    const pa = PRIORITY[a] ?? 999
    const pb = PRIORITY[b] ?? 999
    if (pa !== pb) return pa - pb
    return a.localeCompare(b)
  })
}

/**
 * Validate a single field value against its schema definition.
 * @param {{type: string, required?: boolean, min?: number, max?: number, maxLength?: number, options?: string[]}} fieldDef
 * @param {any} value
 * @returns {string} error message, or '' if valid
 */
export function validateField(fieldDef, value) {
  if (fieldDef.required) {
    if (value === undefined || value === null || value === '') return 'Required'
    if (Array.isArray(value) && value.length === 0) return 'Required'
  }

  if (value === undefined || value === null || value === '') return ''

  if ((fieldDef.type === 'int' || fieldDef.type === 'float') && typeof value === 'number') {
    if (fieldDef.min !== undefined && value < fieldDef.min) return `Min ${fieldDef.min}`
    if (fieldDef.max !== undefined && value > fieldDef.max) return `Max ${fieldDef.max}`
  }

  if (typeof value === 'string' && fieldDef.maxLength !== undefined) {
    if (value.length > fieldDef.maxLength) return `Max ${fieldDef.maxLength} chars`
  }

  if (fieldDef.type === 'enum' && fieldDef.options?.length > 0) {
    if (!fieldDef.options.includes(value)) return `Must be one of: ${fieldDef.options.join(', ')}`
  }

  return ''
}

/** Convert an ISO/RFC3339 date string to an HTML date input value (YYYY-MM-DD). */
export function toDateInput(val) {
  if (!val || typeof val !== 'string') return ''
  return val.slice(0, 10)
}

/** Convert an HTML date input value (YYYY-MM-DD) to RFC3339 format. */
export function toRFC3339(dateStr) {
  if (!dateStr) return ''
  return dateStr + 'T00:00:00Z'
}

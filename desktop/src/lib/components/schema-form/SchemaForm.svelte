<script>
  import { normalizeFields, sortFieldKeys, validateField } from './schema-utils.js'

  import StringField from './widgets/StringField.svelte'
  import BoolField from './widgets/BoolField.svelte'
  import IntField from './widgets/IntField.svelte'
  import FloatField from './widgets/FloatField.svelte'
  import DateField from './widgets/DateField.svelte'
  import TextareaField from './widgets/TextareaField.svelte'
  import EnumField from './widgets/EnumField.svelte'
  import ListField from './widgets/ListField.svelte'
  import ColorField from './widgets/ColorField.svelte'
  import ImageField from './widgets/ImageField.svelte'

  let { fields = {}, values = {}, onchange, order = null } = $props()

  const WIDGET_MAP = {
    string: StringField,
    bool: BoolField,
    int: IntField,
    float: FloatField,
    date: DateField,
    textarea: TextareaField,
    enum: EnumField,
    list: ListField,
    color: ColorField,
    image: ImageField,
  }

  let normalized = $derived(normalizeFields(fields, values))

  let sortedKeys = $derived.by(() => {
    return sortFieldKeys(Object.keys(normalized), order)
  })

  let errors = $derived.by(() => {
    const result = {}
    for (const key of sortedKeys) {
      result[key] = validateField(normalized[key], values?.[key])
    }
    return result
  })
</script>

<div class="schema-form">
  {#each sortedKeys as key (key)}
    {@const def = normalized[key]}
    {@const Widget = WIDGET_MAP[def.type] ?? StringField}
    <Widget
      name={key}
      label={def.label}
      value={values?.[key] ?? def.default ?? (def.type === 'bool' ? false : def.type === 'list' ? [] : '')}
      required={def.required}
      error={errors[key]}
      options={def.options}
      min={def.min}
      max={def.max}
      maxLength={def.maxLength}
      onchange={(v) => onchange(key, v)}
    />
  {/each}
</div>

<style>
  .schema-form {
    display: flex;
    flex-direction: column;
  }
</style>

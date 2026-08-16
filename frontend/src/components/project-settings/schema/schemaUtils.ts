// Minimal structural view of the JSON Schemas this project serves (a subset of
// draft-07). The backend derives these from Go structs via
// github.com/google/jsonschema-go, so they are stable and typed.
export interface SchemaNode {
  type?: string | string[];
  title?: string;
  description?: string;
  default?: unknown;
  enum?: unknown[];
  properties?: Record<string, SchemaNode>;
  items?: SchemaNode | SchemaNode[];
  additionalProperties?: boolean | SchemaNode;
  required?: string[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  minItems?: number;
  maxItems?: number;
  [key: string]: unknown;
}

export type SchemaValue = unknown;

// Secret-detection rule kept in sync with the backend's secretKeyPattern in
// internal/ai/api/configfiles.go so the form masks the same fields.
const SECRET_RE = /api[_-]?key|secret|token|authorization|password|credential|private[_-]?key/i;
export function isSecretName(name: string): boolean {
  return SECRET_RE.test(name);
}

// Humanize snake_case / camelCase keys for labels.
export function humanizeKey(key: string): string {
  const spaced = key
    .replace(/_/g, ' ')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/\s+/g, ' ')
    .trim();
  if (!spaced) return key;
  return spaced.charAt(0).toUpperCase() + spaced.slice(1);
}

export function isObjectSchema(s?: SchemaNode): boolean {
  if (!s) return false;
  const types = Array.isArray(s.type) ? s.type : s.type ? [s.type] : [];
  return (types.includes('object') || types.length === 0) && !!s.properties;
}

export function isMapSchema(s?: SchemaNode): boolean {
  if (!s) return false;
  const types = Array.isArray(s.type) ? s.type : s.type ? [s.type] : [];
  if (s.properties) return false;
  return (
    (types.includes('object') || types.length === 0) &&
    typeof s.additionalProperties === 'object' &&
    !ArrayIsSchemaArray(s.additionalProperties)
  );
}

export function isArraySchema(s?: SchemaNode): boolean {
  if (!s) return false;
  return (Array.isArray(s.type) ? s.type : []).includes('array') || !!s.items || s.type === 'array';
}

function ArrayIsSchemaArray(v: unknown): v is SchemaNode[] {
  return Array.isArray(v);
}

export function isEnumSchema(s?: SchemaNode): boolean {
  return !!s && Array.isArray(s.enum) && s.enum.length > 0;
}

export function isNumberSchema(s?: SchemaNode): boolean {
  if (!s) return false;
  const types = Array.isArray(s.type) ? s.type : s.type ? [s.type] : [];
  return types.includes('number') || types.includes('integer');
}

export function isBooleanSchema(s?: SchemaNode): boolean {
  if (!s) return false;
  return (Array.isArray(s.type) ? s.type : s.type ? [s.type] : []).includes('boolean');
}

export function fieldSchemaFor(
  schema: SchemaNode | undefined,
  key: string,
): SchemaNode | undefined {
  if (!schema?.properties) return undefined;
  return schema.properties[key];
}

// Return the subschema describing array items for a SchemaNode.
export function itemSchemaOf(s?: SchemaNode): SchemaNode | undefined {
  if (!s) return undefined;
  if (Array.isArray(s.items)) return s.items[0];
  return s.items;
}

// Build an object default populated with every declared property's default.
export function makeDefaultObject(s?: SchemaNode): Record<string, unknown> {
  if (!s?.properties) return {};
  const out: Record<string, unknown> = {};
  for (const key of Object.keys(s.properties)) {
    out[key] = makeDefault(s.properties[key]);
  }
  return out;
}

// Field-level shape detection used by SchemaField to dispatch rendering.
export function fieldIsObject(schema?: SchemaNode): boolean {
  return isObjectSchema(schema);
}
export function fieldIsMap(schema?: SchemaNode): boolean {
  return isMapSchema(schema);
}
export function fieldIsArray(schema?: SchemaNode): boolean {
  return isArraySchema(schema);
}

// Materialize an empty/undefined scalar default for a schema node so an
// "add field" action has a sensible value. Object/map/array containers
// become empty containers; scalars default to schema.default or type zero.
export function makeDefault(s?: SchemaNode): unknown {
  if (!s) return undefined;
  if ('default' in s && s.default !== undefined) return s.default;
  const type = Array.isArray(s.type) ? s.type[0] : s.type;
  switch (type) {
    case 'boolean':
      return false;
    case 'number':
    case 'integer':
      return typeof s.minimum === 'number' ? s.minimum : 0;
    case 'array':
      return [];
    case 'object':
      return {};
    default:
      if (s.enum && s.enum.length) return s.enum[0];
      return isNumberSchema(s) ? 0 : isBooleanSchema(s) ? false : '';
  }
}

// Copyright 2024-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonschema

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"unicode"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// An enumeration of the JSON Schema type names.
const (
	jsArray   = "array"
	jsBoolean = "boolean"
	jsInteger = "integer"
	jsNull    = "null"
	jsNumber  = "number"
	jsObject  = "object"
	jsString  = "string"

	defsPrefix = "#/$defs/"

	// Any integers greater or less than these extrema cannot be safely represented
	// according to RFC8259.
	jsMaxInt  = 1<<53 - 1
	jsMinInt  = -jsMaxInt
	jsMaxUint = uint64(jsMaxInt)
)

type FieldVisibility int

const (
	FieldVisible FieldVisibility = iota
	FieldHide
	FieldIgnore
)

type GeneratorOption func(*Generator)

// WithJSONNames sets the generator to use JSON field names as the primary name.
func WithJSONNames() GeneratorOption {
	return func(p *Generator) {
		p.useJSONNames = true
	}
}

// WithAdditionalProperties sets the generator to allow additional properties on messages.
func WithAdditionalProperties() GeneratorOption {
	return func(p *Generator) {
		p.additionalProperties = true
	}
}

// WithStrict sets the generator to require input be pre-normalized.
//
// When a JSON value is converted to protobuf, the converter uses the protobuf
// schema to normalize and validate it further. The default generated schema
// takes this into account, allowing for implicit default values, aliases, and
// other leniencies.
//
// When strict is enabled, the generated schema will not allow these leniencies.
// Specifically, the JSON schema:
//   - Requires implicit default values be explicitly set.
//     These fields are automatically populated in protobuf.
//   - Does not allow aliases for field names.
//   - Does not allow numbers to be represented as strings.
//   - Requires Infinity and NaN values to be exactly capitalized.
//   - Does not allow integers to be represented as strings.
//
// The "always emit fields without presence" option must be set for ProtoJSON to
// output to be valid when strict is enabled. See https://protobuf.dev/programming-guides/json/#json-options
func WithStrict() GeneratorOption {
	return func(p *Generator) {
		p.strict = true
	}
}

// WithBundle sets the generator to bundle all schemas references into the
// same file.
func WithBundle() GeneratorOption {
	return func(p *Generator) {
		p.bundle = true
	}
}

// Generator is a JSON schema generator for protobuf messages.
type Generator struct {
	schema               map[protoreflect.FullName]*msgSchema
	custom               map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldRules, map[string]any) error
	useJSONNames         bool
	additionalProperties bool
	strict               bool
	bundle               bool
}

// NewGenerator creates a new JSON schema generator with the given options.
func NewGenerator(opts ...GeneratorOption) *Generator {
	result := &Generator{
		schema: make(map[protoreflect.FullName]*msgSchema),
	}
	result.custom = result.makeWktGenerators()
	for _, opt := range opts {
		opt(result)
	}
	return result
}

// Add adds a message descriptor to the generator.
func (p *Generator) Add(desc protoreflect.MessageDescriptor) error {
	schema, err := p.generate(desc)
	if err != nil {
		return fmt.Errorf("failed to generate schema for %q: %w", desc.FullName(), err)
	}
	schema.added = true
	return nil
}

// Generate returns the generated JSON schema for all added message descriptors (and their dependencies if not bundling).
func (p *Generator) Generate() map[protoreflect.FullName]map[string]any {
	result := make(map[protoreflect.FullName]map[string]any, len(p.schema))
	if !p.bundle {
		// Not bundling, so return each schema separately.
		for name, entry := range p.schema {
			result[name] = entry.schema
		}
		return result
	}

	// Bundling so only return the bundled schemas of types that were explicitly added.
	for name, entry := range p.schema {
		if entry.added {
			result[name] = p.bundleSchema(entry)
		}
	}
	return result
}

// bundleSchema creates a bundled schema for the given entry.
func (p *Generator) bundleSchema(entry *msgSchema) map[string]any {
	defs := make(map[string]any, len(entry.refs)+1)
	defs[strings.TrimPrefix(entry.id, defsPrefix)] = entry.schema
	// Collect all referenced schemas.
	for ref := range entry.refs {
		p.bundleReferences(ref, defs)
	}
	return map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     p.getID(entry.desc, true),
		"$ref":    p.getID(entry.desc, false),
		"$defs":   defs,
	}
}

// bundleReferences recursively collects all referenced schemas and adds them to the defs map.
func (p *Generator) bundleReferences(name protoreflect.FullName, defs map[string]any) {
	entry, ok := p.schema[name]
	if !ok {
		return // Not found.
	}
	// Add the reference.
	defID := strings.TrimPrefix(entry.id, defsPrefix)
	if _, ok := defs[defID]; ok {
		return // Already added.
	}
	defs[defID] = entry.schema

	// Add transitive references.
	for ref := range entry.refs {
		p.bundleReferences(ref, defs)
	}
}

// msgSchema is the internal representation of a protobuf message's schema.
type msgSchema struct {
	// id is the unique identifier in the JSON schema for this message.
	id     string
	desc   protoreflect.MessageDescriptor
	schema map[string]any
	// refs is a map of all referenced message schemas.
	refs map[protoreflect.FullName]struct{}
	// added is true if this schema was explicitly added and false if it is a dependency.
	added bool
}

// getID returns the ID for the given descriptor.
//
// If bundleID is true, the ID for the bundle is returned.
func (p *Generator) getID(desc protoreflect.Descriptor, bundleID bool) string {
	var result string
	if !bundleID && p.bundle {
		result = defsPrefix
	}
	if p.useJSONNames {
		result += string(desc.FullName()) + ".jsonschema"
	} else {
		result += string(desc.FullName()) + ".schema"
	}
	if p.strict {
		result += ".strict"
	}
	if bundleID {
		result += ".bundle"
	}
	return result + ".json"
}

// getRef returns the reference ID for the given field descriptor.
func (p *Generator) getRef(fdesc protoreflect.FieldDescriptor) string {
	if !p.bundle && fdesc.Parent() == fdesc.Message() {
		return "#"
	}
	return p.getID(fdesc.Message(), false)
}

// generate is the entry point for (recursively) generating the schema for a message descriptor.
func (p *Generator) generate(desc protoreflect.MessageDescriptor) (*msgSchema, error) {
	if entry, ok := p.schema[desc.FullName()]; ok {
		return entry, nil // Already generated.
	}

	// Create a new entry for the message.
	entry := &msgSchema{
		desc:   desc,
		schema: make(map[string]any),
		id:     p.getID(desc, false),
	}
	entry.schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	if !p.bundle {
		entry.schema["$id"] = entry.id
	}
	entry.schema["title"] = nameToTitle(desc.Name())
	p.schema[desc.FullName()] = entry

	// Generate the schema.
	if custom, ok := p.custom[desc.FullName()]; ok {
		// Custom generator.
		return entry, custom(desc, nil, entry.schema)
	}
	// Default generator.
	return entry, p.generateMessage(entry)
}

func (p *Generator) generateMessage(entry *msgSchema) error {
	entry.schema["type"] = jsObject
	p.setDescription(entry.desc, entry.schema)

	var oneOfRules []*validate.MessageOneofRule
	rules, err := p.getMessageRules(entry.desc)
	if err != nil {
		return err
	}
	if rules != nil {
		oneOfRules = append(oneOfRules, rules.Oneof...)
	}

	var required []string
	properties := make(map[string]any)
	patternProperties := make(map[string]any)
	fieldPropertiesNames := make(map[protoreflect.Name]string)
	for i := range entry.desc.Fields().Len() {
		field := entry.desc.Fields().Get(i)

		// If this is a one fields add it as a oneOf message rule
		if oneOfRule, err := p.fieldOneOfToMessageRule(field); err != nil {
			return err
		} else if oneOfRule != nil {
			oneOfRules = append(oneOfRules, oneOfRule)
		}

		visibility := p.shouldIgnoreField(field)
		if visibility == FieldIgnore {
			fieldPropertiesNames[field.Name()] = ""
			continue
		}
		rules, err := p.getFieldRules(field)
		if err != nil {
			return err
		}
		if (rules.GetRequired() && rules.GetIgnore() != validate.Ignore_IGNORE_IF_ZERO_VALUE) || // Required by validate rules.
			(p.strict && p.hasImplicitDefault(field, field.IsList() || field.IsMap(), rules)) { // Required by strict mode.
			if p.useJSONNames {
				required = append(required, field.JSONName())
			} else {
				required = append(required, string(field.Name()))
			}
		}

		// Generate the schema.
		fieldSchema, err := p.generateField(entry, field, rules)
		if err != nil {
			return fmt.Errorf("failed to generate field %q: %w", field.FullName(), err)
		}
		// Add the field schema to the properties.
		fieldProperty, aliases := p.getFieldPropertyNames(field, visibility == FieldHide)
		fieldPropertiesNames[field.Name()] = fieldProperty
		if fieldProperty != "" {
			properties[fieldProperty] = fieldSchema
		}

		// Add any aliases to the pattern properties.
		if !p.strict && len(aliases) > 0 {
			pattern := "^(" + strings.Join(aliases, "|") + ")$"
			patternProperties[pattern] = fieldSchema
		}
	}
	entry.schema["properties"] = properties
	entry.schema["additionalProperties"] = p.additionalProperties
	if len(patternProperties) > 0 {
		entry.schema["patternProperties"] = patternProperties
	}
	if len(required) > 0 {
		entry.schema["required"] = required
	}

	return p.addOneOfConstraintsToSchema(entry.schema, fieldPropertiesNames, oneOfRules)
}

func (p *Generator) fieldOneOfToMessageRule(field protoreflect.FieldDescriptor) (*validate.MessageOneofRule, error) {
	oneOf := field.ContainingOneof()
	if oneOf == nil {
		return nil, nil
	}
	fields := oneOf.Fields()

	// Only transform the first field to avoid duplicate entries
	if fields.Len() == 0 || fields.Get(0).FullName() != field.FullName() {
		return nil, nil
	}

	rule := validate.MessageOneofRule_builder{}
	rule.Fields = make([]string, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		rule.Fields[i] = string(fields.Get(i).Name())
	}

	oneOfRule, err := protovalidate.ResolveOneofRules(oneOf)
	if err != nil {
		return nil, err
	}
	if oneOfRule != nil && oneOfRule.HasRequired() {
		rule.Required = proto.Bool(oneOfRule.GetRequired())
	}
	return rule.Build(), nil
}

func (p *Generator) addOneOfConstraintsToSchema(schema map[string]any, fieldProperties map[protoreflect.Name]string, rules []*validate.MessageOneofRule) error {
	if len(rules) == 0 {
		return nil
	}

	var required []string
	if r, found := schema["required"]; found {
		rs, ok := r.([]string)
		if !ok {
			return fmt.Errorf("invalid required field in schema")
		}
		required = rs
	}

	var allOf []any
	for _, rule := range rules {
		fields := rule.GetFields()
		properties := make([]string, 0, len(fields))
		for _, field := range fields {
			property, ok := fieldProperties[protoreflect.Name(field)]
			if !ok {
				return fmt.Errorf("failed to generate one of rule for unknown field %q", field)
			}
			if property == "" {
				continue
			}
			properties = append(properties, property)
		}
		if len(properties) == 0 {
			continue
		}

		if len(properties) == 1 {
			if rule.GetRequired() && !slices.Contains(required, properties[0]) {
				required = append(required, properties[0])
			}
			continue
		}

		anyOf := make([]map[string]any, 0, len(properties)+1)
		for j := range properties {
			notRequired := make([]string, 0, len(fields)-1)
			notRequired = append(notRequired, properties[:j]...)
			notRequired = append(notRequired, properties[j+1:]...)
			anyOf = append(anyOf, map[string]any{
				"required": []string{properties[j]},
				"not": map[string]any{
					"required": notRequired,
				},
			})
		}

		if !rule.GetRequired() {
			anyOf = append(anyOf, map[string]any{
				"not": map[string]any{
					"required": properties,
				},
			})
		}
		allOf = append(allOf, map[string]any{
			"anyOf": anyOf,
		})

		// Ensure none of the oneOf fields are required
		required = slices.DeleteFunc(required, func(s string) bool {
			return slices.Contains(properties, s)
		})
	}

	if len(required) == 0 {
		delete(schema, "required")
	} else {
		schema["required"] = required
	}
	if len(allOf) > 0 {
		schema["allOf"] = allOf
	}

	return nil
}

func (p *Generator) getFieldPropertyNames(
	field protoreflect.FieldDescriptor,
	hide bool,
) (name string, aliases []string) {
	// TODO: Add an option to include custom alias.
	aliases = make([]string, 0, 1)
	if p.useJSONNames {
		// Add the JSON name as the primary name.
		if hide {
			aliases = append(aliases, field.JSONName())
		} else {
			name = field.JSONName()
		}
		// Add the proto name as an alias.
		if field.JSONName() != string(field.Name()) {
			aliases = append(aliases, string(field.Name()))
		}
		return
	}

	// Add the proto name as the primary name.
	if hide {
		aliases = append(aliases, string(field.Name()))
	} else {
		name = string(field.Name())
	}
	// Add the JSON name as an alias.
	if field.JSONName() != string(field.Name()) {
		aliases = append(aliases, field.JSONName())
	}
	return
}

func (p *Generator) setDescription(desc protoreflect.Descriptor, schema map[string]any) {
	src := desc.ParentFile().SourceLocations().ByDescriptor(desc)
	if src.LeadingComments != "" {
		comments := strings.TrimSpace(src.LeadingComments)
		// JSON schema has two fields for 'comments': title and description
		// To support this, split the comments into to sections.
		// Sections are separated by two newlines.
		// The first 'section' is the title, the rest are the description.
		parts := strings.SplitN(comments, "\n\n", 2)
		if len(parts) < 2 {
			// Check for Windows line endings.
			parts = strings.SplitN(comments, "\r\n\r\n", 2)
		}
		if len(parts) == 2 {
			// Found at least two sections.
			// The first section is the title.
			schema["title"] = strings.TrimSpace(parts[0])
			// The rest are the description.
			schema["description"] = strings.TrimSpace(parts[1])
		} else {
			// Only one section.
			// Use the whole comment as the description.
			schema["description"] = comments
			// Leave the title as the default (empty for fields, the message name for messages).
		}
	}
}

func (p *Generator) generateField(entry *msgSchema, field protoreflect.FieldDescriptor, rules *validate.FieldRules) (map[string]any, error) {
	var schema = make(map[string]any)
	p.setDescription(field, schema)
	if err := p.generateFieldValidation(entry, field, false, rules, schema); err != nil {
		return nil, err
	}
	return schema, nil
}

func (p *Generator) generateFieldValidation(entry *msgSchema, field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) error {
	if field.IsList() {
		schema["type"] = jsArray
		items := make(map[string]any)
		schema["items"] = items
		schema = items
		rules = rules.GetRepeated().GetItems()
		hasImplicitPresence = true
	}
	switch field.Kind() {
	case protoreflect.BoolKind:
		p.generateBoolValidation(field, hasImplicitPresence, rules, schema)
	case protoreflect.EnumKind:
		p.generateEnumValidation(field, hasImplicitPresence, rules, schema)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		p.generateInt32Validation(field, hasImplicitPresence, rules, schema)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		p.generateInt64Validation(field, hasImplicitPresence, rules, schema)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		p.generateUint32Validation(field, hasImplicitPresence, rules, schema)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		p.generateUint64Validation(field, hasImplicitPresence, rules, schema)
	case protoreflect.FloatKind:
		p.generateFloatValidation(field, hasImplicitPresence, rules, schema, 32)
	case protoreflect.DoubleKind:
		p.generateFloatValidation(field, hasImplicitPresence, rules, schema, 64)
	case protoreflect.StringKind:
		p.generateStringValidation(field, hasImplicitPresence, rules, schema)
	case protoreflect.BytesKind:
		p.generateBytesValidation(field, hasImplicitPresence, rules, schema)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if field.IsMap() {
			schema["type"] = jsObject
			propertyNames := make(map[string]any)
			rules, err := p.getFieldRules(field)
			if err != nil {
				return err
			}
			if err := p.generateFieldValidation(entry, field.MapKey(), true, rules.GetMap().GetKeys(), propertyNames); err != nil {
				return err
			}
			schema["propertyNames"] = propertyNames
			properties := make(map[string]any)
			if err := p.generateFieldValidation(entry, field.MapValue(), true, rules.GetMap().GetValues(), properties); err != nil {
				return err
			}
			schema["additionalProperties"] = properties
		} else {
			return p.generateMessageValidation(entry, field, schema)
		}
	}
	return nil
}

func (p *Generator) getMessageRules(msg protoreflect.MessageDescriptor) (*validate.MessageRules, error) {
	return protovalidate.ResolveMessageRules(msg)
}

func (p *Generator) getFieldRules(field protoreflect.FieldDescriptor) (*validate.FieldRules, error) {
	rules, err := protovalidate.ResolveFieldRules(field)
	if err != nil {
		return nil, err
	}
	if rules.GetIgnore() == validate.Ignore_IGNORE_ALWAYS {
		rules = nil
	}
	return rules, nil
}

// hasImplicitDefault checks if the field has an implicit default value.
//
// A field has an implicit default value if:
// 1. It does not have presence tracking. This is only true for non-optional proto3 scalar fields.
// 2. It does not have implicit presence tracking. This is true for repeated fields and map key/value fields.
// 3. It is not required.
//
// If all these conditions are met, if the field is absent, protobuf will interpret it as having the default value.
func (p *Generator) hasImplicitDefault(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules) bool {
	if field.HasPresence() || hasImplicitPresence {
		return false // Default values is absence.
	}
	if rules.GetRequired() && rules.GetIgnore() != validate.Ignore_IGNORE_IF_ZERO_VALUE {
		return false // A value is required.
	}
	// The value is always present so has an implicit default.
	return true
}

// generateDefault sets the 'default' value in the JSON schema, if applicable.
func (p *Generator) generateDefault(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	if !p.strict && p.hasImplicitDefault(field, hasImplicitPresence, rules) {
		// Explicitly define the implicit protobuf default value in the JSON schema.
		schema["default"] = field.Default().Interface()
	}
}

func nameToTitle(name protoreflect.Name) string {
	// Convert camel case to space separated words.
	var result strings.Builder
	for i, chr := range name {
		isUpper := unicode.IsUpper(chr)
		nextIsUpper := i+1 >= len(name) || unicode.IsUpper(rune(name[i+1]))
		if i > 0 && isUpper && !nextIsUpper {
			result.WriteRune(' ')
		}
		result.WriteRune(chr)
	}
	return result.String()
}

func (p *Generator) generateBoolValidation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	schema["type"] = jsBoolean
	if !field.HasPresence() && rules.GetRequired() {
		// False is not allowed.
		schema["enum"] = []bool{true}
		return
	}
	p.generateDefault(field, hasImplicitPresence, rules, schema)
	if rules.GetBool() != nil && rules.GetBool().Const != nil {
		schema["enum"] = []bool{rules.GetBool().GetConst()}
	}
}

type enumValueSelector struct {
	remove bool
	number int32
	name   protoreflect.Name
}

func (p *Generator) generateEnumValidation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	allowZero := true
	hideZero := false
	if !field.HasPresence() && !hasImplicitPresence {
		// The field is a non-optional, non-oneof proto3 enum field.
		if rules.GetRequired() && rules.GetIgnore() != validate.Ignore_IGNORE_IF_ZERO_VALUE {
			// It is required, so zero is not allowed.
			allowZero = false
		} else if !p.strict {
			// Zero is allowed, but absence is preferred.
			hideZero = true
		}
	}

	// Enumerate the values.
	enumValues := make([]enumValueSelector, field.Enum().Values().Len())
	for i := range field.Enum().Values().Len() {
		val := field.Enum().Values().Get(i)
		enumValues[i] = enumValueSelector{
			remove: !allowZero && val.Number() == 0,
			number: int32(val.Number()),
			name:   val.Name(),
		}
	}

	// Apply const.
	if rules.GetEnum().HasConst() {
		for i, enumValue := range enumValues {
			if enumValue.number != rules.GetEnum().GetConst() {
				enumValues[i].remove = true
			}
		}
	}
	// Apply In.
	if len(rules.GetEnum().GetIn()) > 0 {
		for i, enumValue := range enumValues {
			if !enumValue.remove && !slices.Contains(rules.GetEnum().GetIn(), enumValue.number) {
				enumValues[i].remove = true
			}
		}
	}
	// Apply NotIn.
	if len(rules.GetEnum().GetNotIn()) > 0 {
		for i, enumValue := range enumValues {
			if !enumValue.remove && slices.Contains(rules.GetEnum().GetNotIn(), enumValue.number) {
				enumValues[i].remove = true
			}
		}
	}

	anyOf := []map[string]any{}

	// Add the selected enum names to the schema, in order of declaration.
	int32Values := make([]int32, 0, len(enumValues))
	stringValues := make([]string, 0, len(enumValues))
	for _, enumValue := range enumValues {
		if enumValue.remove {
			continue
		}
		int32Values = append(int32Values, enumValue.number)
		if hideZero && enumValue.number == 0 {
			// Use a pattern so IDEs don't suggest the zero value, but it is considered valid when explicitly specified.
			anyOf = append(anyOf, map[string]any{"type": jsString, "pattern": "^" + string(enumValue.name) + "$"})
		} else {
			stringValues = append(stringValues, string(enumValue.name))
		}
	}
	if len(stringValues) > 0 {
		anyOf = append(anyOf, map[string]any{"type": jsString, "enum": stringValues})
	}

	if !p.strict {
		// Add the integer values to the schema, in order of value.
		switch {
		case rules.GetEnum().GetDefinedOnly(),
			rules.GetEnum().HasConst(),
			len(rules.GetEnum().GetIn()) > 0:
			anyOf = p.generateEnumInt32Validation(int32Values, anyOf)
		case allowZero:
			anyOf = append(anyOf, map[string]any{"type": jsInteger, "minimum": math.MinInt32, "maximum": math.MaxInt32})
		default:
			anyOf = append(anyOf,
				map[string]any{"type": jsInteger, "minimum": math.MinInt32, "exclusiveMaximum": 0},
				map[string]any{"type": jsInteger, "exclusiveMinimum": 0, "maximum": math.MaxInt32})
		}
	}

	if len(anyOf) == 1 {
		maps.Copy(schema, anyOf[0])
	} else {
		schema["anyOf"] = anyOf
	}

	schema["title"] = nameToTitle(field.Enum().Name())
	p.generateDefault(field, hasImplicitPresence, rules, schema)
}

func (p *Generator) generateEnumInt32Validation(int32Values []int32, anyOf []map[string]any) []map[string]any {
	if len(int32Values) == 0 {
		return anyOf
	}
	// Use ranges instead of an enum so IDEs only suggest names, not numbers.
	slices.Sort(int32Values)
	int32Values = slices.Compact(int32Values)
	start := int32Values[0]
	last := start
	for _, intVal := range int32Values[1:] {
		if intVal != last+1 {
			anyOf = append(anyOf, map[string]any{"type": jsInteger, "minimum": start, "maximum": last})
			start = intVal
		}
		last = intVal
	}
	anyOf = append(anyOf, map[string]any{"type": jsInteger, "minimum": start, "maximum": last})
	return anyOf
}

type baseRule[T comparable] interface {
	HasConst() bool
	GetConst() T
	GetIn() []T
}

type numberRule[T comparable] interface {
	baseRule[T]

	HasGte() bool
	GetGte() T
	HasGt() bool
	GetGt() T
	HasLte() bool
	GetLte() T
	HasLt() bool
	GetLt() T
}

func generateConstInValidation[T comparable](rules baseRule[T], schema map[string]any) {
	if rules.HasConst() {
		schema["enum"] = []T{rules.GetConst()}
	} else if len(rules.GetIn()) > 0 {
		schema["enum"] = rules.GetIn()
	}
}

func generateIntValidation[T int32 | int64](
	strict bool,
	rules numberRule[T],
	bits int,
	schema map[string]any,
) {
	// TODO: Consider suppressing the number output if all valid values
	// are out of the range [jsMinInt, jsMaxInt].
	numberSchema := map[string]any{
		"type": jsInteger,
	}
	minVal := -(1 << (bits - 1))
	maxExclVal := uint64(1) << (bits - 1)
	var orNumberSchema map[string]any

	generateConstInValidation(rules, numberSchema)
	switch {
	case rules.HasGt():
		var isOr bool
		switch {
		case rules.HasLt():
			isOr = rules.GetLt() <= rules.GetGt()
		case rules.HasLte():
			isOr = rules.GetLte() <= rules.GetGt()
		}
		if isOr {
			orNumberSchema = make(map[string]any)
			if int64(rules.GetGt()) >= jsMinInt {
				orNumberSchema["exclusiveMinimum"] = rules.GetGt()
			}
		} else if int64(rules.GetGt()) >= jsMinInt {
			numberSchema["exclusiveMinimum"] = rules.GetGt()
		}
	case rules.HasGte():
		var isOr bool
		switch {
		case rules.HasLt():
			isOr = rules.GetLt() <= rules.GetGte()
		case rules.HasLte():
			isOr = rules.GetLte() < rules.GetGte()
		}
		if isOr {
			orNumberSchema = make(map[string]any)
			if int64(rules.GetGte()) > jsMinInt {
				orNumberSchema["minimum"] = rules.GetGte()
			}
		} else if int64(rules.GetGte()) > jsMinInt {
			numberSchema["minimum"] = rules.GetGte()
		}
	default:
		if bits <= 53 {
			numberSchema["minimum"] = minVal
		}
	}
	switch {
	case rules.HasLt():
		if int64(rules.GetLt()) <= jsMaxInt {
			numberSchema["exclusiveMaximum"] = rules.GetLt()
		}
	case rules.HasLte():
		if int64(rules.GetLte()) < jsMaxInt {
			numberSchema["maximum"] = rules.GetLte()
		}
	default:
		if bits < 53 {
			numberSchema["exclusiveMaximum"] = maxExclVal
		}
	}

	anyOf := []map[string]any{
		numberSchema,
	}

	if orNumberSchema != nil {
		if bits < 53 {
			numberSchema["minimum"] = minVal
			orNumberSchema["exclusiveMaximum"] = maxExclVal
		}
		orNumberSchema["type"] = jsInteger
		anyOf = append(anyOf, orNumberSchema)
	}

	if !strict {
		// Always allow string representation of numbers to match
		// https://protobuf.dev/programming-guides/json/
		anyOf = append(anyOf, map[string]any{
			"type":    jsString,
			"pattern": "^-?[0-9]+$",
		})
	}

	if len(anyOf) > 1 {
		schema["anyOf"] = anyOf
	} else {
		maps.Copy(schema, numberSchema)
	}
}

func (p *Generator) generateInt32Validation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	switch {
	default:
		if p.strict {
			schema["type"] = jsInteger
			schema["minimum"] = math.MinInt32
			schema["maximum"] = math.MaxInt32
		} else {
			schema["anyOf"] = []map[string]any{
				{"type": jsInteger, "minimum": math.MinInt32, "maximum": math.MaxInt32},
				{"type": jsString, "pattern": "^-?[0-9]+$"},
			}
		}
	case rules.GetInt32() != nil:
		generateIntValidation(p.strict, rules.GetInt32(), 32, schema)
	case rules.GetSint32() != nil:
		generateIntValidation(p.strict, rules.GetSint32(), 32, schema)
	case rules.GetSfixed32() != nil:
		generateIntValidation(p.strict, rules.GetSfixed32(), 32, schema)
	}
	p.generateDefault(field, hasImplicitPresence, rules, schema)
}

func (p *Generator) generateInt64Validation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	switch {
	default:
		if p.strict {
			schema["type"] = jsInteger
		} else {
			schema["anyOf"] = []map[string]any{
				{"type": jsInteger},
				{"type": jsString, "pattern": "^-?[0-9]+$"},
			}
		}
	case rules.GetInt64() != nil:
		generateIntValidation(p.strict, rules.GetInt64(), 64, schema)
	case rules.GetSint64() != nil:
		generateIntValidation(p.strict, rules.GetSint64(), 64, schema)
	case rules.GetSfixed64() != nil:
		generateIntValidation(p.strict, rules.GetSfixed64(), 64, schema)
	}
	p.generateDefault(field, hasImplicitPresence, rules, schema)
}

func generateUintValidation[T uint32 | uint64](
	strict bool,
	rules numberRule[T],
	bits int,
	schema map[string]any,
) {
	// TODO: Consider suppressing the number output if all valid values
	// are out of the range [0, jsMaxUint].
	numberSchema := map[string]any{
		"type": jsInteger,
	}
	var orNumberSchema map[string]any
	maxExclVal := float64(uint64(1)<<(bits-1)) * 2
	generateConstInValidation(rules, numberSchema)
	switch {
	case rules.HasGt():
		var isOr bool
		switch {
		case rules.HasLt():
			isOr = rules.GetLt() <= rules.GetGt()
		case rules.HasLte():
			isOr = rules.GetLte() <= rules.GetGt()
		}
		if isOr {
			orNumberSchema = make(map[string]any)
			if uint64(rules.GetGt()) <= jsMaxUint {
				orNumberSchema["exclusiveMinimum"] = rules.GetGt()
			}
		} else {
			numberSchema["exclusiveMinimum"] = rules.GetGt()
		}
	case rules.HasGte():
		var isOr bool
		switch {
		case rules.HasLt():
			isOr = rules.GetLt() <= rules.GetGte()
		case rules.HasLte():
			isOr = rules.GetLte() < rules.GetGte()
		}
		if isOr {
			orNumberSchema = map[string]any{"minimum": rules.GetGte()}
		} else {
			numberSchema["minimum"] = rules.GetGte()
		}
	default:
		numberSchema["minimum"] = 0
	}
	switch {
	case rules.HasLt():
		if uint64(rules.GetLt()) <= jsMaxUint {
			numberSchema["exclusiveMaximum"] = rules.GetLt()
		}
	case rules.HasLte():
		if uint64(rules.GetLte()) < jsMaxUint {
			numberSchema["maximum"] = rules.GetLte()
		}
	case bits < 53:
		numberSchema["exclusiveMaximum"] = maxExclVal
	}

	anyOf := []map[string]any{
		numberSchema,
	}
	if orNumberSchema != nil {
		numberSchema["minimum"] = 0
		if bits < 53 {
			orNumberSchema["exclusiveMaximum"] = maxExclVal
		}
		orNumberSchema["type"] = jsInteger
		anyOf = append(anyOf, orNumberSchema)
	}

	if !strict {
		// Always allow string representation of uints to match
		// https://protobuf.dev/programming-guides/json/
		anyOf = append(anyOf, map[string]any{
			"type":    jsString,
			"pattern": "^[0-9]+$",
		})
	}

	if len(anyOf) > 1 {
		schema["anyOf"] = anyOf
	} else {
		maps.Copy(schema, numberSchema)
	}
}
func (p *Generator) generateUint32Validation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	switch {
	default:
		if p.strict {
			schema["type"] = jsInteger
			schema["minimum"] = 0
			schema["maximum"] = math.MaxUint32
		} else {
			schema["anyOf"] = []map[string]any{
				{"type": jsInteger, "minimum": 0, "maximum": math.MaxUint32},
				{"type": jsString, "pattern": "^[0-9]+$"},
			}
		}
	case rules.GetUint32() != nil:
		generateUintValidation(p.strict, rules.GetUint32(), 32, schema)
	case rules.GetFixed32() != nil:
		generateUintValidation(p.strict, rules.GetFixed32(), 32, schema)
	}
	p.generateDefault(field, hasImplicitPresence, rules, schema)
}

func (p *Generator) generateUint64Validation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	switch {
	default:
		if p.strict {
			schema["type"] = jsInteger
			schema["minimum"] = 0
		} else {
			schema["anyOf"] = []map[string]any{
				{"type": jsInteger, "minimum": 0},
				{"type": jsString, "pattern": "^[0-9]+$"},
			}
		}
	case rules.GetUint64() != nil:
		generateUintValidation(p.strict, rules.GetUint64(), 64, schema)
	case rules.GetFixed64() != nil:
		generateUintValidation(p.strict, rules.GetFixed64(), 64, schema)
	}
	p.generateDefault(field, hasImplicitPresence, rules, schema)
}

// nolint: gocyclo
func (p *Generator) generateFloatValidation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any, bits int) {
	includePosInf := true
	includeNegInf := true
	includeNaN := true

	numberSchema := map[string]any{
		"type": jsNumber,
	}
	var orNumberSchema map[string]any

	switch {
	default:
		if bits == 32 {
			numberSchema["minimum"] = -math.MaxFloat32
			numberSchema["maximum"] = math.MaxFloat32
		}
	case rules.GetFloat() != nil:
		if rules.GetFloat().GetFinite() {
			includePosInf = false
			includeNegInf = false
			includeNaN = false
		}
		if rules.GetFloat().Const != nil {
			numberSchema["enum"] = []float32{rules.GetFloat().GetConst()}
			includePosInf = false
			includeNegInf = false
			includeNaN = false
			if math.IsInf(float64(rules.GetFloat().GetConst()), 1) {
				includePosInf = true
			}
			if math.IsInf(float64(rules.GetFloat().GetConst()), -1) {
				includeNegInf = true
			}
			if math.IsNaN(float64(rules.GetFloat().GetConst())) {
				includeNaN = true
			}
		}
		if len(rules.GetFloat().GetIn()) > 0 {
			numberSchema["enum"] = rules.GetFloat().GetIn()
			includePosInf = false
			includeNegInf = false
			includeNaN = false
			for _, value := range rules.GetFloat().GetIn() {
				if math.IsInf(float64(value), 1) {
					includePosInf = true
				}
				if math.IsInf(float64(value), -1) {
					includeNegInf = true
				}
				if math.IsNaN(float64(value)) {
					includeNaN = true
				}
			}
		}
		switch greaterThan := rules.GetFloat().GetGreaterThan().(type) {
		case *validate.FloatRules_Gt:
			includeNaN = false
			var isOr bool
			switch lessThan := rules.GetFloat().GetLessThan().(type) {
			case *validate.FloatRules_Lt:
				isOr = lessThan.Lt <= greaterThan.Gt
			case *validate.FloatRules_Lte:
				isOr = lessThan.Lte <= greaterThan.Gt
			}
			if isOr {
				orNumberSchema = map[string]any{
					"type":             jsNumber,
					"exclusiveMinimum": greaterThan.Gt,
				}
			} else {
				includeNegInf = false
				numberSchema["exclusiveMinimum"] = greaterThan.Gt
			}
		case *validate.FloatRules_Gte:
			includeNaN = false
			isOr := false
			switch lessThan := rules.GetFloat().GetLessThan().(type) {
			case *validate.FloatRules_Lt:
				isOr = lessThan.Lt <= greaterThan.Gte
			case *validate.FloatRules_Lte:
				isOr = lessThan.Lte < greaterThan.Gte
			}
			if isOr {
				orNumberSchema = map[string]any{
					"type":    jsNumber,
					"minimum": greaterThan.Gte,
				}
			} else {
				if greaterThan.Gte != float32(math.Inf(-1)) {
					includeNegInf = false
				}
				numberSchema["minimum"] = greaterThan.Gte
			}
		default:
			numberSchema["minimum"] = -math.MaxFloat32
		}
		switch lessThan := rules.GetFloat().GetLessThan().(type) {
		case *validate.FloatRules_Lt:
			includeNaN = false
			if orNumberSchema == nil {
				includePosInf = false
			}
			numberSchema["exclusiveMaximum"] = lessThan.Lt
		case *validate.FloatRules_Lte:
			includeNaN = false
			if lessThan.Lte != float32(math.Inf(1)) && orNumberSchema == nil {
				includePosInf = false
			}
			numberSchema["maximum"] = lessThan.Lte
		default:
			numberSchema["maximum"] = math.MaxFloat32
		}
	case rules.GetDouble() != nil:
		if rules.GetDouble().GetFinite() {
			includePosInf = false
			includeNegInf = false
			includeNaN = false
		}
		if rules.GetDouble().Const != nil {
			numberSchema["enum"] = []float64{rules.GetDouble().GetConst()}
			includePosInf = false
			includeNegInf = false
			includeNaN = false
			if math.IsInf(rules.GetDouble().GetConst(), 1) {
				includePosInf = true
			}
			if math.IsInf(rules.GetDouble().GetConst(), -1) {
				includeNegInf = true
			}
			if math.IsNaN(rules.GetDouble().GetConst()) {
				includeNaN = true
			}
		}
		if len(rules.GetDouble().GetIn()) > 0 {
			numberSchema["enum"] = rules.GetDouble().GetIn()
			includePosInf = false
			includeNegInf = false
			includeNaN = false
			for _, value := range rules.GetDouble().GetIn() {
				if math.IsInf(value, 1) {
					includePosInf = true
				}
				if math.IsInf(value, -1) {
					includeNegInf = true
				}
				if math.IsNaN(value) {
					includeNaN = true
				}
			}
		}
		switch greaterThan := rules.GetDouble().GetGreaterThan().(type) {
		case *validate.DoubleRules_Gt:
			includeNaN = false
			var isOr bool
			switch lessThan := rules.GetDouble().GetLessThan().(type) {
			case *validate.DoubleRules_Lt:
				isOr = lessThan.Lt <= greaterThan.Gt
			case *validate.DoubleRules_Lte:
				isOr = lessThan.Lte <= greaterThan.Gt
			}
			if isOr {
				orNumberSchema = map[string]any{
					"type":             jsNumber,
					"exclusiveMinimum": greaterThan.Gt,
				}
			} else {
				includeNegInf = false
				numberSchema["exclusiveMinimum"] = greaterThan.Gt
			}
		case *validate.DoubleRules_Gte:
			includeNaN = false
			isOr := false
			switch lessThan := rules.GetDouble().GetLessThan().(type) {
			case *validate.DoubleRules_Lt:
				isOr = lessThan.Lt <= greaterThan.Gte
			case *validate.DoubleRules_Lte:
				isOr = lessThan.Lte < greaterThan.Gte
			}
			if isOr {
				orNumberSchema = map[string]any{
					"type":    jsNumber,
					"minimum": greaterThan.Gte,
				}
			} else {
				if greaterThan.Gte != math.Inf(-1) {
					includeNegInf = false
				}
				numberSchema["minimum"] = greaterThan.Gte
			}
		}
		switch lessThan := rules.GetDouble().GetLessThan().(type) {
		case *validate.DoubleRules_Lt:
			includeNaN = false
			if orNumberSchema == nil {
				includePosInf = false
			}
			numberSchema["exclusiveMaximum"] = lessThan.Lt
		case *validate.DoubleRules_Lte:
			includeNaN = false
			if lessThan.Lte != math.Inf(1) && orNumberSchema == nil {
				includePosInf = false
			}
			numberSchema["maximum"] = lessThan.Lte
		}
	}

	anyOf := []map[string]any{
		numberSchema,
	}
	if orNumberSchema != nil {
		anyOf = append(anyOf, orNumberSchema)
	}

	extremaEnum := []any{}
	if includePosInf {
		extremaEnum = append(extremaEnum, "Infinity")
	}
	if includeNegInf {
		extremaEnum = append(extremaEnum, "-Infinity")
	}
	if includeNaN {
		extremaEnum = append(extremaEnum, "NaN")
	}
	if len(extremaEnum) > 0 {
		anyOf = append(anyOf, map[string]any{
			"type": jsString,
			"enum": extremaEnum,
		})
		if !p.strict {
			// Allow other form of NaN, -Infinity, and Infinity.
			anyOf = append(anyOf, map[string]any{"type": jsString})
		}
	} else if !p.strict {
		anyOf = append(anyOf, map[string]any{
			"type":    jsString,
			"pattern": "^-?[0-9]+(\\.[0-9]+)?([eE][+-]?[0-9]+)?$",
		})
	}

	if len(anyOf) > 1 {
		schema["anyOf"] = anyOf
	} else {
		maps.Copy(schema, numberSchema)
	}
	p.generateDefault(field, hasImplicitPresence, rules, schema)
}

const (
	ipv4PatternBit     = "((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)"
	ipv6PatternBit     = "(([0-9a-fA-F]{1,4}::?){1,7}([0-9a-fA-F]{1,4})|([0-9a-fA-F]{1,4}:){1,7}:|:((([0-9a-fA-F]{1,4}:){1,6})?[0-9a-fA-F]{1,4})?|::)"
	ipv4LenPatternBit  = "/([0-9]|[12][0-9]|3[0-2])"
	ipv6LenPatternBit  = "/([0-9]|[1-9][0-9]|1[0-1][0-9]|12[0-8])"
	portPatternBit     = "([1-9][0-9]{0,4}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])"
	hostnamePatternBit = "[A-Za-z0-9][A-Za-z0-9-]{0,63}(\\.[A-Za-z0-9-][A-Za-z0-9-]{0,63})*"

	ipv4Pattern          = "^" + ipv4PatternBit + "$"
	ipv6Pattern          = "^" + ipv6PatternBit + "$"
	hostnamePattern      = "^" + hostnamePatternBit + "$"
	uriPattern           = "^(?:(?:[a-zA-Z][a-zA-Z\\d+\\-.]*):)?(?://(?:[A-Za-z0-9\\-\\.]+(?::\\d+)?))?(/[^\\?#]*)?(?:\\?([^\\#]*))?(?:\\#(.*))?$"
	uriRefPattern        = "^(?:(?:[a-zA-Z][a-zA-Z\\d+\\-.]*):)?(?:\\/\\/(?:[A-Za-z0-9\\-\\.]+(?::\\d+)?))?(/[^\\?#]*)?(?:\\?([^\\#]*))?(?:\\#(.*))?$"
	uuidPattern          = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
	tuuidPattern         = "^[0-9a-fA-F]{32}$"
	ipv4PrefixLenPattern = "^" + ipv4PatternBit + ipv4LenPatternBit + "$"
	ipv6PrefixLenPattern = "^" + ipv6PatternBit + ipv6LenPatternBit + "$"
	ipv4PrefixPattern    = "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}0/([0-9]|[12][0-9]|3[0-2])$"
	ipv6PrefixPattern    = "^(([0-9a-fA-F]{1,4}:){1,7}:|::)/([0-9]|[1-9][0-9]|1[0-1][0-9]|12[0-8])$"
	hostAndPortPattern   = "^(" + hostnamePatternBit + "|" + ipv4PatternBit + "|\\[" + ipv6PatternBit + "\\]):" + portPatternBit + "$"
)

// nolint: gocyclo
func generateWellKnownPattern(rules *validate.FieldRules, schema map[string]any) {
	switch wellKnown := rules.GetString().GetWellKnown().(type) {
	case *validate.StringRules_Hostname:
		if wellKnown.Hostname {
			schema["pattern"] = hostnamePattern
		}
	case *validate.StringRules_Email:
		if wellKnown.Email {
			schema["format"] = "email"
		}
	case *validate.StringRules_Ip:
		if wellKnown.Ip {
			schema["pattern"] = fmt.Sprintf("%s|%s", ipv4Pattern, ipv6Pattern)
		}
	case *validate.StringRules_Ipv4:
		if wellKnown.Ipv4 {
			schema["format"] = "ipv4"
		}
	case *validate.StringRules_Ipv6:
		if wellKnown.Ipv6 {
			schema["format"] = "ipv6"
		}
	case *validate.StringRules_Uri:
		if wellKnown.Uri {
			schema["pattern"] = uriPattern
		}
	case *validate.StringRules_UriRef:
		if wellKnown.UriRef {
			schema["pattern"] = uriRefPattern
		}
	case *validate.StringRules_Address:
		if wellKnown.Address {
			schema["pattern"] = fmt.Sprintf("%s|%s|%s", ipv4Pattern, ipv6Pattern, hostnamePattern)
		}
	case *validate.StringRules_Uuid:
		if wellKnown.Uuid {
			schema["pattern"] = uuidPattern
		}
	case *validate.StringRules_Tuuid:
		if wellKnown.Tuuid {
			schema["pattern"] = tuuidPattern
		}
	case *validate.StringRules_Ipv4WithPrefixlen:
		if wellKnown.Ipv4WithPrefixlen {
			schema["pattern"] = ipv4PrefixLenPattern
		}
	case *validate.StringRules_Ipv6WithPrefixlen:
		if wellKnown.Ipv6WithPrefixlen {
			schema["pattern"] = ipv6PrefixLenPattern
		}
	case *validate.StringRules_IpWithPrefixlen:
		if wellKnown.IpWithPrefixlen {
			schema["pattern"] = fmt.Sprintf("%s|%s", ipv4PrefixLenPattern, ipv6PrefixLenPattern)
		}
	case *validate.StringRules_Ipv4Prefix:
		if wellKnown.Ipv4Prefix {
			schema["pattern"] = ipv4PrefixPattern
		}
	case *validate.StringRules_Ipv6Prefix:
		if wellKnown.Ipv6Prefix {
			schema["pattern"] = ipv6PrefixPattern
		}
	case *validate.StringRules_IpPrefix:
		if wellKnown.IpPrefix {
			schema["pattern"] = fmt.Sprintf("%s|%s", ipv4PrefixPattern, ipv6PrefixPattern)
		}
	case *validate.StringRules_HostAndPort:
		if wellKnown.HostAndPort {
			schema["pattern"] = hostAndPortPattern
		}
	case *validate.StringRules_WellKnownRegex:
		if wellKnown.WellKnownRegex == validate.KnownRegex_KNOWN_REGEX_HTTP_HEADER_NAME &&
			rules.GetString().GetStrict() {
			schema["pattern"] = "^:?[0-9a-zA-Z!#$%&\\'*+-.^_|~\\x60]+$"
		}
	}
}

func (p *Generator) generateStringValidation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	schema["type"] = jsString
	p.generateDefault(field, hasImplicitPresence, rules, schema)
	if rules.GetString() == nil {
		return
	}

	// Bytes are <= Characters, so we can only enforce an upper bound.
	if rules.GetString().LenBytes != nil {
		schema["maxLength"] = rules.GetString().GetMaxBytes()
	} else if rules.GetString().MaxBytes != nil {
		schema["maxLength"] = rules.GetString().GetMaxBytes()
	}

	if rules.GetString().Len != nil {
		schema["minLength"] = rules.GetString().GetLen()
		schema["maxLength"] = rules.GetString().GetLen()
	} else {
		if rules.GetString().MinLen != nil && rules.GetString().GetMinLen() > 0 {
			schema["minLength"] = rules.GetString().GetMinLen()
		} else if rules.GetRequired() {
			schema["minLength"] = 1
		}
		if rules.GetString().MaxLen != nil {
			schema["maxLength"] = rules.GetString().GetMaxLen()
		}
	}

	generateWellKnownPattern(rules, schema)

	switch {
	case rules.GetString().Pattern != nil:
		schema["pattern"] = rules.GetString().GetPattern()
	case rules.GetString().Prefix != nil,
		rules.GetString().Suffix != nil,
		rules.GetString().Contains != nil:
		pattern := ""
		if rules.GetString().Prefix != nil {
			pattern += "^" + rules.GetString().GetPrefix()
		}
		pattern += ".*"
		if rules.GetString().Contains != nil {
			pattern += rules.GetString().GetContains()
			pattern += ".*"
		}
		if rules.GetString().Suffix != nil {
			pattern += rules.GetString().GetSuffix() + "$"
		}
		schema["pattern"] = pattern
	}

	if rules.GetString().Const != nil {
		schema["enum"] = []string{rules.GetString().GetConst()}
	} else if len(rules.GetString().GetIn()) > 0 {
		schema["enum"] = rules.GetString().GetIn()
	}
}

func base64EncodedLength(inputSize uint64) (uint64, uint64) {
	// Base64 encoding is 4/3 the size of the input.
	// Padding is added to make the output size a multiple of 4.
	// For example 5 bytes is encoded as
	characters := inputSize * 4 / 3
	if inputSize%3 != 0 {
		characters++
	}
	padding := 4 - (characters % 4)
	return characters, padding
}

func (p *Generator) generateBytesValidation(field protoreflect.FieldDescriptor, hasImplicitPresence bool, rules *validate.FieldRules, schema map[string]any) {
	schema["type"] = jsString
	// Set a regex to match base64 encoded strings.
	schema["pattern"] = "^[A-Za-z0-9+/]*={0,2}$"
	p.generateDefault(field, hasImplicitPresence, rules, schema)
	if rules.GetBytes() == nil {
		return
	}

	if rules.GetBytes().Len != nil {
		size, padding := base64EncodedLength(rules.GetBytes().GetLen())
		schema["minLength"] = size
		schema["maxLength"] = size + padding
	} else {
		if rules.GetBytes().MaxLen != nil {
			size, padding := base64EncodedLength(rules.GetBytes().GetMaxLen())
			schema["maxLength"] = size + padding
		}
		if rules.GetBytes().MinLen != nil {
			size, _ := base64EncodedLength(rules.GetBytes().GetMinLen())
			schema["minLength"] = size
		} else if rules.GetRequired() {
			schema["minLength"] = 1
		}
	}
}

func (p *Generator) generateMessageValidation(entry *msgSchema, field protoreflect.FieldDescriptor, schema map[string]any) error {
	// Create a reference to the message type.
	if entry != nil {
		if entry.refs == nil {
			entry.refs = map[protoreflect.FullName]struct{}{field.Message().FullName(): {}}
		} else {
			entry.refs[field.Message().FullName()] = struct{}{}
		}
	}
	schema["$ref"] = p.getRef(field)
	// Ensure the schema for the message type is generated.
	_, err := p.generate(field.Message())
	return err
}

func (p *Generator) generateWrapperValidation(
	desc protoreflect.MessageDescriptor,
	rules *validate.FieldRules,
	schema map[string]any,
) error {
	field := desc.Fields().Get(0)
	p.setDescription(field, schema)
	return p.generateFieldValidation(nil, field, true, rules, schema)
}

func (p *Generator) makeWktGenerators() map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldRules, map[string]any) error {
	var result = make(map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldRules, map[string]any) error)
	result["google.protobuf.Any"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldRules, schema map[string]any) error { // nolint: unparam
		schema["type"] = jsObject
		schema["properties"] = map[string]any{
			"@type": map[string]any{
				"type": "string",
			},
		}
		return nil
	}

	result["google.protobuf.Duration"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldRules, schema map[string]any) error { // nolint: unparam
		schema["type"] = jsString
		schema["format"] = "duration"
		return nil
	}
	result["google.protobuf.Timestamp"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldRules, schema map[string]any) error { // nolint: unparam
		schema["type"] = jsString
		schema["format"] = "date-time"
		return nil
	}

	result["google.protobuf.Value"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldRules, _ map[string]any) error { // nolint: unparam
		return nil
	}
	result["google.protobuf.ListValue"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldRules, schema map[string]any) error { // nolint: unparam
		schema["type"] = jsArray
		return nil
	}
	result["google.protobuf.NullValue"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldRules, schema map[string]any) error { // nolint: unparam
		schema["type"] = jsNull
		return nil
	}
	result["google.protobuf.Struct"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldRules, schema map[string]any) error { // nolint: unparam
		schema["type"] = jsObject
		return nil
	}

	result["google.protobuf.BoolValue"] = p.generateWrapperValidation
	result["google.protobuf.BytesValue"] = p.generateWrapperValidation
	result["google.protobuf.DoubleValue"] = p.generateWrapperValidation
	result["google.protobuf.FloatValue"] = p.generateWrapperValidation
	result["google.protobuf.Int32Value"] = p.generateWrapperValidation
	result["google.protobuf.Int64Value"] = p.generateWrapperValidation
	result["google.protobuf.StringValue"] = p.generateWrapperValidation
	result["google.protobuf.UInt32Value"] = p.generateWrapperValidation
	result["google.protobuf.UInt64Value"] = p.generateWrapperValidation
	return result
}

func (p *Generator) shouldIgnoreField(fdesc protoreflect.FieldDescriptor) FieldVisibility {
	const ignoreComment = "jsonschema:ignore"
	const hideComment = "jsonschema:hide"
	srcLoc := fdesc.ParentFile().SourceLocations().ByDescriptor(fdesc)
	switch {
	case strings.Contains(srcLoc.LeadingComments, ignoreComment),
		strings.Contains(srcLoc.TrailingComments, ignoreComment):
		return FieldIgnore
	case strings.Contains(srcLoc.LeadingComments, hideComment),
		strings.Contains(srcLoc.TrailingComments, hideComment):
		return FieldHide
	default:
		return FieldVisible
	}
}

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
	"math/big"
	"slices"
	"strings"
	"unicode"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go/resolve"
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
)

type FieldVisibility int

const (
	FieldVisible FieldVisibility = iota
	FieldHide
	FieldIgnore
)

var (
	exclusiveMaxUint64 = big.NewInt(0).Exp(big.NewInt(2), big.NewInt(64), nil)
)

type GeneratorOption func(*jsonSchemaGenerator)

// WithJSONNames sets the generator to use JSON field names as the primary name.
func WithJSONNames() GeneratorOption {
	return func(p *jsonSchemaGenerator) {
		p.useJSONNames = true
	}
}

// Generate generates a JSON schema for the given message descriptor, with protobuf field names.
func Generate(input protoreflect.MessageDescriptor, opts ...GeneratorOption) map[protoreflect.FullName]map[string]any {
	generator := &jsonSchemaGenerator{
		schema: make(map[protoreflect.FullName]map[string]any),
	}
	generator.custom = generator.makeWktGenerators()
	for _, opt := range opts {
		opt(generator)
	}
	generator.generate(input)
	return generator.schema
}

type jsonSchemaGenerator struct {
	schema       map[protoreflect.FullName]map[string]any
	custom       map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldConstraints, map[string]any)
	useJSONNames bool
}

func (p *jsonSchemaGenerator) getID(desc protoreflect.Descriptor) string {
	if p.useJSONNames {
		return string(desc.FullName()) + ".jsonschema.json"
	}
	return string(desc.FullName()) + ".schema.json"
}

func (p *jsonSchemaGenerator) getRef(fdesc protoreflect.FieldDescriptor) string {
	if fdesc.Parent() == fdesc.Message() {
		return "#"
	}
	return p.getID(fdesc.Message())
}

func (p *jsonSchemaGenerator) generate(desc protoreflect.MessageDescriptor) {
	if _, ok := p.schema[desc.FullName()]; ok {
		return // Already generated.
	}
	schema := make(map[string]any)
	schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	schema["$id"] = p.getID(desc)
	schema["title"] = generateTitle(desc.Name())
	p.schema[desc.FullName()] = schema
	if custom, ok := p.custom[desc.FullName()]; ok { // Custom generator.
		custom(desc, nil, schema)
	} else { // Default generator.
		p.generateDefault(desc, schema)
	}
}

func (p *jsonSchemaGenerator) generateDefault(desc protoreflect.MessageDescriptor, schema map[string]any) {
	schema["type"] = jsObject
	p.setDescription(desc, schema)
	var required []string
	properties := make(map[string]any)
	patternProperties := make(map[string]any)
	for i := range desc.Fields().Len() {
		field := desc.Fields().Get(i)
		visibility := p.shouldIgnoreField(field)
		if visibility == FieldIgnore {
			continue
		}
		constraints := p.getFieldConstraints(field)
		if constraints.GetRequired() && constraints.GetIgnore() != validate.Ignore_IGNORE_IF_UNPOPULATED {
			required = append(required, string(field.Name()))
		}

		// Generate the schema.
		fieldSchema := p.generateField(field, constraints)

		// TODO: Add an option to include custom alias.
		aliases := make([]string, 0, 1)

		switch {
		case visibility == FieldHide:
			aliases = append(aliases, string(field.Name()))
			if field.JSONName() != string(field.Name()) {
				aliases = append(aliases, field.JSONName())
			}
		case p.useJSONNames:
			// Use the JSON name as the primary name.
			properties[field.JSONName()] = fieldSchema
			if field.JSONName() != string(field.Name()) {
				aliases = append(aliases, string(field.Name()))
			}
		default:
			// Use the proto name as the primary name.
			properties[string(field.Name())] = fieldSchema
			if field.JSONName() != string(field.Name()) {
				aliases = append(aliases, field.JSONName())
			}
		}

		if len(aliases) > 0 {
			pattern := "^(" + strings.Join(aliases, "|") + ")$"
			patternProperties[pattern] = fieldSchema
		}
	}
	schema["properties"] = properties
	schema["additionalProperties"] = false
	if len(patternProperties) > 0 {
		schema["patternProperties"] = patternProperties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
}

func (p *jsonSchemaGenerator) setDescription(desc protoreflect.Descriptor, schema map[string]any) {
	src := desc.ParentFile().SourceLocations().ByDescriptor(desc)
	if src.LeadingComments != "" {
		comments := strings.TrimSpace(src.LeadingComments)
		// JSON schema has two fields for 'comments': title and description
		// To support this, split the comments into two sections.
		// Sections are separated by two newlines.
		// The first 'section' is the title, the rest are the description.
		if parts := strings.SplitN(comments, "\n\n", 2); len(parts) >= 2 {
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

func (p *jsonSchemaGenerator) generateField(field protoreflect.FieldDescriptor, constraints *validate.FieldConstraints) map[string]any {
	var schema = make(map[string]any)
	p.setDescription(field, schema)
	p.generateValidation(field, constraints, schema)
	return schema
}

func (p *jsonSchemaGenerator) generateValidation(field protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	if field.IsList() {
		schema["type"] = jsArray
		items := make(map[string]any)
		schema["items"] = items
		schema = items
		constraints = constraints.GetRepeated().GetItems()
	}
	switch field.Kind() {
	case protoreflect.BoolKind:
		p.generateBoolValidation(field, constraints, schema)
	case protoreflect.EnumKind:
		p.generateEnumValidation(field, constraints, schema)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		p.generateInt32Validation(field, constraints, schema)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		p.generateInt64Validation(field, constraints, schema)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		p.generateUint32Validation(field, constraints, schema)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		p.generateUint64Validation(field, constraints, schema)
	case protoreflect.FloatKind:
		p.generateFloatValidation(field, constraints, schema, 32)
	case protoreflect.DoubleKind:
		p.generateFloatValidation(field, constraints, schema, 64)
	case protoreflect.StringKind:
		p.generateStringValidation(field, constraints, schema)
	case protoreflect.BytesKind:
		p.generateBytesValidation(field, constraints, schema)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if field.IsMap() {
			schema["type"] = jsObject
			propertyNames := make(map[string]any)
			constraints := p.getFieldConstraints(field)
			p.generateValidation(field.MapKey(), constraints.GetMap().GetKeys(), propertyNames)
			schema["propertyNames"] = propertyNames
			properties := make(map[string]any)
			p.generateValidation(field.MapValue(), constraints.GetMap().GetValues(), properties)
			schema["additionalProperties"] = properties
		} else {
			p.generateMessageValidation(field, schema)
		}
	}
}

func (p *jsonSchemaGenerator) getFieldConstraints(field protoreflect.FieldDescriptor) *validate.FieldConstraints {
	constraints := resolve.FieldConstraints(field)
	if constraints == nil || constraints.GetIgnore() == validate.Ignore_IGNORE_ALWAYS {
		return nil
	}
	return constraints
}

func (p *jsonSchemaGenerator) generateBoolValidation(field protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	schema["type"] = jsBoolean
	if !field.HasPresence() && constraints.GetRequired() && constraints.GetIgnore() != validate.Ignore_IGNORE_IF_DEFAULT_VALUE {
		// False is not allowed.
		schema["enum"] = []bool{true}
	} else if constraints.GetBool() != nil && constraints.GetBool().Const != nil {
		schema["enum"] = []bool{constraints.GetBool().GetConst()}
	}
}

func generateTitle(name protoreflect.Name) string {
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

type enumFieldSelector struct {
	selected bool
	index    int
}

func (p *jsonSchemaGenerator) generateEnumValidation(field protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	enumFieldSelectors := make(map[int32]enumFieldSelector, field.Enum().Values().Len())
	for i := range field.Enum().Values().Len() {
		val := field.Enum().Values().Get(i)
		enumFieldSelectors[int32(val.Number())] = enumFieldSelector{
			selected: true,
			index:    i,
		}
	}

	if constraints.GetEnum() != nil && constraints.GetEnum().HasConst() {
		for number := range enumFieldSelectors {
			if number != constraints.GetEnum().GetConst() {
				enumFieldSelectors[number] = enumFieldSelector{}
			}
		}
	}

	if constraints.GetEnum() != nil && len(constraints.GetEnum().GetIn()) > 0 {
		inMap := make(map[int32]struct{}, len(constraints.GetEnum().GetIn()))
		for _, value := range constraints.GetEnum().GetIn() {
			inMap[value] = struct{}{}
		}

		for number := range enumFieldSelectors {
			if _, ok := inMap[number]; !ok {
				enumFieldSelectors[number] = enumFieldSelector{}
			}
		}
	}

	if constraints.GetEnum() != nil && len(constraints.GetEnum().GetNotIn()) > 0 {
		for _, value := range constraints.GetEnum().GetNotIn() {
			enumFieldSelectors[value] = enumFieldSelector{}
		}
	}

	onlySelectIntValues := constraints.GetEnum() != nil &&
		(constraints.GetEnum().GetDefinedOnly() ||
			constraints.GetEnum().HasConst() ||
			constraints.GetEnum().GetIn() != nil)

	validIntegers := map[string]any{"type": jsInteger, "minimum": math.MinInt32, "maximum": math.MaxInt32}
	if onlySelectIntValues {
		var integerValues = make([]int32, 0)
		for number, val := range enumFieldSelectors {
			if val.selected {
				integerValues = append(integerValues, number)
			}
		}
		slices.Sort(integerValues)

		validIntegers = map[string]any{"type": jsInteger, "enum": integerValues}
	}

	validIndexes := make([]int, 0, len(enumFieldSelectors))
	for _, val := range enumFieldSelectors {
		if val.selected {
			validIndexes = append(validIndexes, val.index)
		}
	}
	slices.Sort(validIndexes)

	var stringValues = make([]string, 0)
	for _, index := range validIndexes {
		stringValues = append(stringValues, string(field.Enum().Values().Get(index).Name()))
	}

	validStrings := map[string]any{"type": jsString, "enum": stringValues, "title": generateTitle(field.Enum().Name())}

	schema["anyOf"] = []map[string]any{validStrings, validIntegers}
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

func generateConstInValidation[T comparable](constraints baseRule[T], schema map[string]any) {
	if constraints.HasConst() {
		schema["enum"] = []T{constraints.GetConst()}
	} else if len(constraints.GetIn()) > 0 {
		schema["enum"] = constraints.GetIn()
	}
}

func generateIntValidation[T int32 | int64](
	constraints numberRule[T],
	bits int,
	schema map[string]any,
) {
	numberSchema := map[string]any{
		"type": jsInteger,
	}
	minVal := -(1 << (bits - 1))
	maxExclVal := uint64(1) << (bits - 1)
	var orNumberSchema map[string]any

	generateConstInValidation(constraints, numberSchema)
	switch {
	case constraints.HasGt():
		var isOr bool
		switch {
		case constraints.HasLt():
			isOr = constraints.GetLt() <= constraints.GetGt()
		case constraints.HasLte():
			isOr = constraints.GetLte() <= constraints.GetGt()
		}
		if isOr {
			orNumberSchema = map[string]any{"exclusiveMinimum": constraints.GetGt()}
		} else {
			numberSchema["exclusiveMinimum"] = constraints.GetGt()
		}
	case constraints.HasGte():
		var isOr bool
		switch {
		case constraints.HasLt():
			isOr = constraints.GetLt() <= constraints.GetGte()
		case constraints.HasLte():
			isOr = constraints.GetLte() < constraints.GetGte()
		}
		if isOr {
			orNumberSchema = map[string]any{"minimum": constraints.GetGte()}
		} else {
			numberSchema["minimum"] = constraints.GetGte()
		}
	default:
		numberSchema["minimum"] = minVal
	}
	switch {
	case constraints.HasLt():
		numberSchema["exclusiveMaximum"] = constraints.GetLt()
	case constraints.HasLte():
		numberSchema["maximum"] = constraints.GetLte()
	default:
		numberSchema["exclusiveMaximum"] = maxExclVal
	}

	anyOf := []map[string]any{
		numberSchema,
	}

	if orNumberSchema != nil {
		numberSchema["minimum"] = minVal
		orNumberSchema["exclusiveMaximum"] = maxExclVal
		orNumberSchema["type"] = jsInteger
		anyOf = append(anyOf, orNumberSchema)
	}
	if bits > 52 {
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

func (p *jsonSchemaGenerator) generateInt32Validation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	switch {
	default:
		schema["type"] = jsInteger
		schema["minimum"] = math.MinInt32
		schema["maximum"] = math.MaxInt32
	case constraints.GetInt32() != nil:
		generateIntValidation(constraints.GetInt32(), 32, schema)
	case constraints.GetSint32() != nil:
		generateIntValidation(constraints.GetSint32(), 32, schema)
	case constraints.GetSfixed32() != nil:
		generateIntValidation(constraints.GetSfixed32(), 32, schema)
	}
}

func (p *jsonSchemaGenerator) generateInt64Validation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	switch {
	default:
		schema["anyOf"] = []map[string]any{
			{"type": jsInteger, "minimum": math.MinInt64, "exclusiveMaximum": uint64(math.MaxInt64) + 1},
			{"type": jsString, "pattern": "^-?[0-9]+$"},
		}
	case constraints.GetInt64() != nil:
		generateIntValidation(constraints.GetInt64(), 64, schema)
	case constraints.GetSint64() != nil:
		generateIntValidation(constraints.GetSint64(), 64, schema)
	case constraints.GetSfixed64() != nil:
		generateIntValidation(constraints.GetSfixed64(), 64, schema)
	}
}

func generateUintValidation[T uint32 | uint64](
	constraints numberRule[T],
	bits int,
	schema map[string]any,
) {
	numberSchema := map[string]any{
		"type": jsInteger,
	}
	var orNumberSchema map[string]any
	maxExclVal := float64(uint64(1)<<(bits-1)) * 2
	generateConstInValidation(constraints, numberSchema)
	switch {
	case constraints.HasGt():
		var isOr bool
		switch {
		case constraints.HasLt():
			isOr = constraints.GetLt() <= constraints.GetGt()
		case constraints.HasLte():
			isOr = constraints.GetLte() <= constraints.GetGt()
		}
		if isOr {
			orNumberSchema = map[string]any{"exclusiveMinimum": constraints.GetGt()}
		} else {
			numberSchema["exclusiveMinimum"] = constraints.GetGt()
		}
	case constraints.HasGte():
		var isOr bool
		switch {
		case constraints.HasLt():
			isOr = constraints.GetLt() <= constraints.GetGte()
		case constraints.HasLte():
			isOr = constraints.GetLte() < constraints.GetGte()
		}
		if isOr {
			orNumberSchema = map[string]any{"minimum": constraints.GetGte()}
		} else {
			numberSchema["minimum"] = constraints.GetGte()
		}
	default:
		numberSchema["minimum"] = 0
	}
	switch {
	case constraints.HasLt():
		numberSchema["exclusiveMaximum"] = constraints.GetLt()
	case constraints.HasLte():
		numberSchema["maximum"] = constraints.GetLte()
	default:
		numberSchema["exclusiveMaximum"] = maxExclVal
	}

	anyOf := []map[string]any{
		numberSchema,
	}
	if bits > 52 {
		anyOf = append(anyOf, map[string]any{
			"type":    jsString,
			"pattern": "^[0-9]+$",
		})
	}
	if orNumberSchema != nil {
		numberSchema["minimum"] = 0
		orNumberSchema["exclusiveMaximum"] = maxExclVal
		orNumberSchema["type"] = jsInteger
		anyOf = append(anyOf, orNumberSchema)
	}
	if len(anyOf) > 1 {
		schema["anyOf"] = anyOf
	} else {
		maps.Copy(schema, numberSchema)
	}
}
func (p *jsonSchemaGenerator) generateUint32Validation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	switch {
	default:
		schema["type"] = jsInteger
		schema["minimum"] = 0
		schema["maximum"] = math.MaxUint32
	case constraints.GetUint32() != nil:
		generateUintValidation(constraints.GetUint32(), 32, schema)
	case constraints.GetFixed32() != nil:
		generateUintValidation(constraints.GetFixed32(), 32, schema)
	}
}

func (p *jsonSchemaGenerator) generateUint64Validation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	switch {
	default:
		schema["anyOf"] = []map[string]any{
			{"type": jsInteger, "minimum": 0, "exclusiveMaximum": exclusiveMaxUint64},
			{"type": jsString, "pattern": "^[0-9]+$"},
		}
	case constraints.GetUint64() != nil:
		generateUintValidation(constraints.GetUint64(), 64, schema)
	case constraints.GetFixed64() != nil:
		generateUintValidation(constraints.GetFixed64(), 64, schema)
	}
}

// nolint: gocyclo
func (p *jsonSchemaGenerator) generateFloatValidation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any, bits int) {
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
	case constraints.GetFloat() != nil:
		if constraints.GetFloat().GetFinite() {
			includePosInf = false
			includeNegInf = false
			includeNaN = false
		}
		if constraints.GetFloat().Const != nil {
			numberSchema["enum"] = []float32{constraints.GetFloat().GetConst()}
			includePosInf = false
			includeNegInf = false
			includeNaN = false
			if math.IsInf(float64(constraints.GetFloat().GetConst()), 1) {
				includePosInf = true
			}
			if math.IsInf(float64(constraints.GetFloat().GetConst()), -1) {
				includeNegInf = true
			}
			if math.IsNaN(float64(constraints.GetFloat().GetConst())) {
				includeNaN = true
			}
		}
		if len(constraints.GetFloat().GetIn()) > 0 {
			numberSchema["enum"] = constraints.GetFloat().GetIn()
			includePosInf = false
			includeNegInf = false
			includeNaN = false
			for _, value := range constraints.GetFloat().GetIn() {
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
		switch greaterThan := constraints.GetFloat().GetGreaterThan().(type) {
		case *validate.FloatRules_Gt:
			includeNaN = false
			var isOr bool
			switch lessThan := constraints.GetFloat().GetLessThan().(type) {
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
			switch lessThan := constraints.GetFloat().GetLessThan().(type) {
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
		switch lessThan := constraints.GetFloat().GetLessThan().(type) {
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
	case constraints.GetDouble() != nil:
		if constraints.GetDouble().GetFinite() {
			includePosInf = false
			includeNegInf = false
			includeNaN = false
		}
		if constraints.GetDouble().Const != nil {
			numberSchema["enum"] = []float64{constraints.GetDouble().GetConst()}
			includePosInf = false
			includeNegInf = false
			includeNaN = false
			if math.IsInf(constraints.GetDouble().GetConst(), 1) {
				includePosInf = true
			}
			if math.IsInf(constraints.GetDouble().GetConst(), -1) {
				includeNegInf = true
			}
			if math.IsNaN(constraints.GetDouble().GetConst()) {
				includeNaN = true
			}
		}
		if len(constraints.GetDouble().GetIn()) > 0 {
			numberSchema["enum"] = constraints.GetDouble().GetIn()
			includePosInf = false
			includeNegInf = false
			includeNaN = false
			for _, value := range constraints.GetDouble().GetIn() {
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
		switch greaterThan := constraints.GetDouble().GetGreaterThan().(type) {
		case *validate.DoubleRules_Gt:
			includeNaN = false
			var isOr bool
			switch lessThan := constraints.GetDouble().GetLessThan().(type) {
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
			switch lessThan := constraints.GetDouble().GetLessThan().(type) {
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
		switch lessThan := constraints.GetDouble().GetLessThan().(type) {
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
		}, map[string]any{
			"type": jsString, // Allow other form of NaN, -Infinity, and Infinity.
		})
	} else {
		anyOf = append(anyOf, map[string]any{
			"type":    jsString,
			"pattern": "^-?[0-9]+(\\.[0-9]+)?([eE][+-]?[0-9]+)?$",
		})
	}

	schema["anyOf"] = anyOf
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
func generateWellKnownPattern(constraints *validate.FieldConstraints, schema map[string]any) {
	switch wellKnown := constraints.GetString().GetWellKnown().(type) {
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
			constraints.GetString().GetStrict() {
			schema["pattern"] = "^:?[0-9a-zA-Z!#$%&\\'*+-.^_|~\\x60]+$"
		}
	}
}

func (p *jsonSchemaGenerator) generateStringValidation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	schema["type"] = jsString
	if constraints.GetString() == nil {
		return
	}

	// Bytes are <= Characters, so we can only enforce an upper bound.
	if constraints.GetString().LenBytes != nil {
		schema["maxLength"] = constraints.GetString().GetMaxBytes()
	} else if constraints.GetString().MaxBytes != nil {
		schema["maxLength"] = constraints.GetString().GetMaxBytes()
	}

	if constraints.GetString().Len != nil {
		schema["minLength"] = constraints.GetString().GetLen()
		schema["maxLength"] = constraints.GetString().GetLen()
	} else {
		if constraints.GetString().MinLen != nil && constraints.GetString().GetMinLen() > 0 {
			schema["minLength"] = constraints.GetString().GetMinLen()
		} else if constraints.GetRequired() && constraints.GetIgnore() != validate.Ignore_IGNORE_IF_DEFAULT_VALUE {
			schema["minLength"] = 1
		}
		if constraints.GetString().MaxLen != nil {
			schema["maxLength"] = constraints.GetString().GetMaxLen()
		}
	}

	generateWellKnownPattern(constraints, schema)

	switch {
	case constraints.GetString().Pattern != nil:
		schema["pattern"] = constraints.GetString().GetPattern()
	case constraints.GetString().Prefix != nil,
		constraints.GetString().Suffix != nil,
		constraints.GetString().Contains != nil:
		pattern := ""
		if constraints.GetString().Prefix != nil {
			pattern += "^" + constraints.GetString().GetPrefix()
		}
		pattern += ".*"
		if constraints.GetString().Contains != nil {
			pattern += constraints.GetString().GetContains()
			pattern += ".*"
		}
		if constraints.GetString().Suffix != nil {
			pattern += constraints.GetString().GetSuffix() + "$"
		}
		schema["pattern"] = pattern
	}

	if constraints.GetString().Const != nil {
		schema["enum"] = []string{constraints.GetString().GetConst()}
	} else if len(constraints.GetString().GetIn()) > 0 {
		schema["enum"] = constraints.GetString().GetIn()
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

func (p *jsonSchemaGenerator) generateBytesValidation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]any) {
	schema["type"] = jsString
	// Set a regex to match base64 encoded strings.
	schema["pattern"] = "^[A-Za-z0-9+/]*={0,2}$"
	if constraints.GetBytes() == nil {
		return
	}

	if constraints.GetBytes().Len != nil {
		size, padding := base64EncodedLength(constraints.GetBytes().GetLen())
		schema["minLength"] = size
		schema["maxLength"] = size + padding
	} else {
		if constraints.GetBytes().MaxLen != nil {
			size, padding := base64EncodedLength(constraints.GetBytes().GetMaxLen())
			schema["maxLength"] = size + padding
		}
		if constraints.GetBytes().MinLen != nil {
			size, _ := base64EncodedLength(constraints.GetBytes().GetMinLen())
			schema["minLength"] = size
		} else if constraints.GetRequired() && constraints.GetIgnore() != validate.Ignore_IGNORE_IF_DEFAULT_VALUE {
			schema["minLength"] = 1
		}
	}
}

func (p *jsonSchemaGenerator) generateMessageValidation(field protoreflect.FieldDescriptor, schema map[string]any) {
	// Create a reference to the message type.
	schema["$ref"] = p.getRef(field)
	p.generate(field.Message())
}

func (p *jsonSchemaGenerator) generateWrapperValidation(
	desc protoreflect.MessageDescriptor,
	constraints *validate.FieldConstraints,
	schema map[string]any,
) {
	field := desc.Fields().Get(0)
	p.setDescription(field, schema)
	p.generateValidation(field, constraints, schema)
}

func (p *jsonSchemaGenerator) makeWktGenerators() map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldConstraints, map[string]any) {
	var result = make(map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldConstraints, map[string]any))
	result["google.protobuf.Any"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]any) {
		schema["type"] = jsObject
		schema["properties"] = map[string]any{
			"@type": map[string]any{
				"type": "string",
			},
		}
	}

	result["google.protobuf.Duration"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]any) {
		schema["type"] = jsString
		schema["format"] = "duration"
	}
	result["google.protobuf.Timestamp"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]any) {
		schema["type"] = jsString
		schema["format"] = "date-time"
	}

	result["google.protobuf.Value"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, _ map[string]any) {}
	result["google.protobuf.ListValue"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]any) {
		schema["type"] = jsArray
	}
	result["google.protobuf.NullValue"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]any) {
		schema["type"] = jsNull
	}
	result["google.protobuf.Struct"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]any) {
		schema["type"] = jsObject
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

func (p *jsonSchemaGenerator) shouldIgnoreField(fdesc protoreflect.FieldDescriptor) FieldVisibility {
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

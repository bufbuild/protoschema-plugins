// Copyright 2024 Buf Technologies, Inc.
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
	"math"
	"strings"
	"unicode"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go/resolver"
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

type GeneratorOption func(*jsonSchemaGenerator)

// WithJSONNames sets the generator to use JSON field names as the primary name.
func WithJSONNames() GeneratorOption {
	return func(p *jsonSchemaGenerator) {
		p.useJSONNames = true
	}
}

// Generate generates a JSON schema for the given message descriptor, with protobuf field names.
func Generate(input protoreflect.MessageDescriptor, opts ...GeneratorOption) map[protoreflect.FullName]map[string]interface{} {
	generator := &jsonSchemaGenerator{
		schema: make(map[protoreflect.FullName]map[string]interface{}),
	}
	generator.custom = generator.makeWktGenerators()
	for _, opt := range opts {
		opt(generator)
	}
	generator.generate(input)
	return generator.schema
}

type jsonSchemaGenerator struct {
	schema       map[protoreflect.FullName]map[string]interface{}
	custom       map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldConstraints, map[string]interface{})
	useJSONNames bool
	resolver     resolver.DefaultResolver
}

func (p *jsonSchemaGenerator) getID(desc protoreflect.Descriptor) string {
	if p.useJSONNames {
		return string(desc.FullName()) + ".jsonschema.json"
	}
	return string(desc.FullName()) + ".schema.json"
}

func (p *jsonSchemaGenerator) generate(desc protoreflect.MessageDescriptor) {
	if _, ok := p.schema[desc.FullName()]; ok {
		return // Already generated.
	}
	schema := make(map[string]interface{})
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

func (p *jsonSchemaGenerator) generateDefault(desc protoreflect.MessageDescriptor, schema map[string]interface{}) {
	schema["type"] = jsObject
	p.setDescription(desc, schema)
	var required []string
	properties := make(map[string]interface{})
	patternProperties := make(map[string]interface{})
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

func (p *jsonSchemaGenerator) setDescription(desc protoreflect.Descriptor, schema map[string]interface{}) {
	src := desc.ParentFile().SourceLocations().ByDescriptor(desc)
	if src.LeadingComments != "" {
		schema["description"] = strings.TrimSpace(src.LeadingComments)
	}
}

func (p *jsonSchemaGenerator) generateField(field protoreflect.FieldDescriptor, constraints *validate.FieldConstraints) map[string]interface{} {
	var schema = make(map[string]interface{})
	p.setDescription(field, schema)
	p.generateValidation(field, constraints, schema)
	return schema
}

func (p *jsonSchemaGenerator) generateValidation(field protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]interface{}) {
	if field.IsList() {
		schema["type"] = jsArray
		items := make(map[string]interface{})
		schema["items"] = items
		schema = items
		constraints = constraints.GetRepeated().GetItems()
	}
	switch field.Kind() {
	case protoreflect.BoolKind:
		p.generateBoolValidation(field, constraints, schema)
	case protoreflect.EnumKind:
		p.generateEnumValidation(field, schema)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		p.generateIntValidation(field, schema, 32)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		p.generateIntValidation(field, schema, 64)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		p.generateUintValidation(field, schema, 32)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		p.generateUintValidation(field, schema, 64)
	case protoreflect.FloatKind:
		p.generateFloatValidation(field, schema, 32)
	case protoreflect.DoubleKind:
		p.generateFloatValidation(field, schema, 64)
	case protoreflect.StringKind:
		p.generateStringValidation(field, constraints, schema)
	case protoreflect.BytesKind:
		p.generateBytesValidation(field, constraints, schema)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if field.IsMap() {
			schema["type"] = jsObject
			propertyNames := make(map[string]interface{})
			constraints := p.getFieldConstraints(field)
			p.generateValidation(field.MapKey(), constraints.GetMap().GetKeys(), propertyNames)
			schema["propertyNames"] = propertyNames
			properties := make(map[string]interface{})
			p.generateValidation(field.MapValue(), constraints.GetMap().GetValues(), properties)
			schema["additionalProperties"] = properties
		} else {
			p.generateMessageValidation(field, schema)
		}
	}
}

func (p *jsonSchemaGenerator) getFieldConstraints(field protoreflect.FieldDescriptor) (constraints *validate.FieldConstraints) {
	constraints = p.resolver.ResolveFieldConstraints(field)
	if constraints == nil || constraints.GetIgnore() == validate.Ignore_IGNORE_ALWAYS {
		return nil
	}
	return constraints
}

func (p *jsonSchemaGenerator) generateBoolValidation(field protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]interface{}) {
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

func (p *jsonSchemaGenerator) generateEnumValidation(field protoreflect.FieldDescriptor, schema map[string]interface{}) {
	var enum = make([]interface{}, 0)
	for i := range field.Enum().Values().Len() {
		enum = append(enum, field.Enum().Values().Get(i).Name())
	}
	anyOf := []map[string]interface{}{
		{"type": jsString, "enum": enum, "title": generateTitle(field.Enum().Name())},
		{"type": jsInteger, "minimum": math.MinInt32, "maximum": math.MaxInt32},
	}
	schema["anyOf"] = anyOf
}

func (p *jsonSchemaGenerator) generateIntValidation(_ protoreflect.FieldDescriptor, schema map[string]interface{}, bitSize int) {
	// Use floats to handle integer overflow.
	minSize := -math.Pow(2, float64(bitSize-1))
	maxSize := math.Pow(2, float64(bitSize-1))
	if bitSize <= 53 {
		schema["type"] = jsInteger
		schema["minimum"] = minSize
		schema["exclusiveMaximum"] = maxSize
	} else {
		schema["anyOf"] = []map[string]interface{}{
			{"type": jsInteger, "minimum": minSize, "maximum": maxSize},
			{"type": jsString, "pattern": "^[0-9]+$"},
		}
	}
}

func (p *jsonSchemaGenerator) generateUintValidation(_ protoreflect.FieldDescriptor, schema map[string]interface{}, bitSize int) {
	schema["type"] = jsInteger
	schema["minimum"] = 0
	schema["exclusiveMaximum"] = math.Pow(2, float64(bitSize))
}

func (p *jsonSchemaGenerator) generateFloatValidation(_ protoreflect.FieldDescriptor, schema map[string]interface{}, _ int) {
	schema["anyOf"] = []map[string]interface{}{
		{"type": jsNumber},
		{"type": jsString},
		{"type": jsString, "enum": []interface{}{"NaN", "Infinity", "-Infinity"}},
	}
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

func (p *jsonSchemaGenerator) generateStringValidation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]interface{}) {
	schema["type"] = jsString
	if constraints.GetString_() == nil {
		return
	}

	// Bytes are <= Characters, so we can only enforce an upper bound.
	if constraints.GetString_().LenBytes != nil {
		schema["maxLength"] = constraints.GetString_().GetMaxBytes()
	} else if constraints.GetString_().MaxBytes != nil {
		schema["maxLength"] = constraints.GetString_().GetMaxBytes()
	}

	if constraints.GetString_().Len != nil {
		schema["minLength"] = constraints.GetString_().GetLen()
		schema["maxLength"] = constraints.GetString_().GetLen()
	} else {
		if constraints.GetString_().MinLen != nil && constraints.GetString_().GetMinLen() > 0 {
			schema["minLength"] = constraints.GetString_().GetMinLen()
		} else if constraints.GetRequired() && constraints.GetIgnore() != validate.Ignore_IGNORE_IF_DEFAULT_VALUE {
			schema["minLength"] = 1
		}
		if constraints.GetString_().MaxLen != nil {
			schema["maxLength"] = constraints.GetString_().GetMaxLen()
		}
	}

	switch wellKnown := constraints.GetString_().GetWellKnown().(type) {
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
		switch wellKnown.WellKnownRegex {
		case validate.KnownRegex_KNOWN_REGEX_HTTP_HEADER_NAME:
			if constraints.GetString_().GetStrict() {
				schema["pattern"] = "^:?[0-9a-zA-Z!#$%&\\'*+-.^_|~\\x60]+$"
			}
		}
	}

	if constraints.GetString_().Pattern != nil {
		schema["pattern"] = constraints.GetString_().GetPattern()
	} else if constraints.GetString_().Prefix != nil ||
		constraints.GetString_().Suffix != nil ||
		constraints.GetString_().Contains != nil {
		pattern := ""
		if constraints.GetString_().Prefix != nil {
			pattern += "^" + constraints.GetString_().GetPrefix()
		}
		pattern += ".*"
		if constraints.GetString_().Contains != nil {
			pattern += constraints.GetString_().GetContains()
			pattern += ".*"
		}
		if constraints.GetString_().Suffix != nil {
			pattern += constraints.GetString_().GetSuffix() + "$"
		}
		schema["pattern"] = pattern
	} else if constraints.GetString_().Contains != nil {
		schema["pattern"] = ".*" + constraints.GetString_().GetContains() + ".*"
	}

	if constraints.GetString_().Const != nil {
		schema["enum"] = []string{constraints.GetString_().GetConst()}
	} else if len(constraints.GetString_().In) > 0 {
		schema["enum"] = constraints.GetString_().GetIn()
	}
}

func base64EncodedLength(inputSize int) (characters, padding int) {
	// Base64 encoding is 4/3 the size of the input.
	// Padding is added to make the output size a multiple of 4.
	// For example 5 bytes is encoded as
	characters = inputSize * 4 / 3
	if inputSize%3 != 0 {
		characters++
	}
	padding = 4 - (characters % 4)
	return characters, padding
}

func (p *jsonSchemaGenerator) generateBytesValidation(_ protoreflect.FieldDescriptor, constraints *validate.FieldConstraints, schema map[string]interface{}) {
	schema["type"] = jsString
	// Set a regex to match base64 encoded strings.
	schema["pattern"] = "^[A-Za-z0-9+/]*={0,2}$"
	if constraints.GetBytes() == nil {
		return
	}

	if constraints.GetBytes().Len != nil {
		size, padding := base64EncodedLength(int(constraints.GetBytes().GetLen()))
		schema["minLength"] = size
		schema["maxLength"] = size + padding
	} else {
		if constraints.GetBytes().MaxLen != nil {
			size, padding := base64EncodedLength(int(constraints.GetBytes().GetMaxLen()))
			schema["maxLength"] = size + padding
		}
		if constraints.GetBytes().MinLen != nil {
			size, _ := base64EncodedLength(int(constraints.GetBytes().GetMinLen()))
			schema["minLength"] = size
		} else if constraints.GetRequired() && constraints.GetIgnore() != validate.Ignore_IGNORE_IF_DEFAULT_VALUE {
			schema["minLength"] = 1
		}
	}
}

func (p *jsonSchemaGenerator) generateMessageValidation(field protoreflect.FieldDescriptor, schema map[string]interface{}) {
	// Create a reference to the message type.
	schema["$ref"] = p.getID(field.Message())
	p.generate(field.Message())
}

func (p *jsonSchemaGenerator) generateWrapperValidation(
	desc protoreflect.MessageDescriptor,
	constraints *validate.FieldConstraints,
	schema map[string]interface{},
) {
	field := desc.Fields().Get(0)
	p.setDescription(field, schema)
	p.generateValidation(field, constraints, schema)
}

func (p *jsonSchemaGenerator) makeWktGenerators() map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldConstraints, map[string]interface{}) {
	var result = make(map[protoreflect.FullName]func(protoreflect.MessageDescriptor, *validate.FieldConstraints, map[string]interface{}))
	result["google.protobuf.Any"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]interface{}) {
		schema["type"] = jsObject
		schema["properties"] = map[string]interface{}{
			"@type": map[string]interface{}{
				"type": "string",
			},
		}
	}

	result["google.protobuf.Duration"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]interface{}) {
		schema["type"] = jsString
		schema["format"] = "duration"
	}
	result["google.protobuf.Timestamp"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]interface{}) {
		schema["type"] = jsString
		schema["format"] = "date-time"
	}

	result["google.protobuf.Value"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, _ map[string]interface{}) {}
	result["google.protobuf.ListValue"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]interface{}) {
		schema["type"] = jsArray
	}
	result["google.protobuf.NullValue"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]interface{}) {
		schema["type"] = jsNull
	}
	result["google.protobuf.Struct"] = func(_ protoreflect.MessageDescriptor, _ *validate.FieldConstraints, schema map[string]interface{}) {
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

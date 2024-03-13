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
	"math"

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

// Generate generates a JSON schema for the given message descriptor.
func Generate(input protoreflect.MessageDescriptor) map[protoreflect.FullName]map[string]interface{} {
	generator := &jsonSchemaGenerator{
		result: make(map[protoreflect.FullName]map[string]interface{}),
	}
	generator.custom = generator.makeWktGenerators()
	generator.generate(input)
	return generator.result
}

type jsonSchemaGenerator struct {
	result map[protoreflect.FullName]map[string]interface{}
	custom map[protoreflect.FullName]func(map[string]interface{}, protoreflect.MessageDescriptor)
}

func (p *jsonSchemaGenerator) getID(desc protoreflect.Descriptor) string {
	return string(desc.FullName()) + ".schema.json"
}

func (p *jsonSchemaGenerator) generate(desc protoreflect.MessageDescriptor) {
	if _, ok := p.result[desc.FullName()]; ok {
		return // Already generated.
	}
	result := make(map[string]interface{})
	result["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	result["$id"] = p.getID(desc)
	p.result[desc.FullName()] = result
	if custom, ok := p.custom[desc.FullName()]; ok { // Custom generator.
		custom(result, desc)
	} else { // Default generator.
		p.generateDefault(result, desc)
	}
}

func (p *jsonSchemaGenerator) generateDefault(result map[string]interface{}, desc protoreflect.MessageDescriptor) {
	result["type"] = jsObject
	p.setDescription(desc, result)
	var properties = make(map[string]interface{})
	for i := 0; i < desc.Fields().Len(); i++ {
		field := desc.Fields().Get(i)
		name := string(field.Name())
		properties[name] = p.generateField(field)
	}
	result["properties"] = properties
	result["additionalProperties"] = false
}

func (p *jsonSchemaGenerator) setDescription(desc protoreflect.Descriptor, result map[string]interface{}) {
	src := desc.ParentFile().SourceLocations().ByDescriptor(desc)
	if src.LeadingComments != "" {
		result["description"] = src.LeadingComments
	}
}

func (p *jsonSchemaGenerator) generateField(field protoreflect.FieldDescriptor) map[string]interface{} {
	var result = make(map[string]interface{})
	p.setDescription(field, result)
	p.generateValidation(field, result)
	return result
}

func (p *jsonSchemaGenerator) generateValidation(field protoreflect.FieldDescriptor, entry map[string]interface{}) {
	if field.IsList() {
		entry["type"] = jsArray
		items := make(map[string]interface{})
		entry["items"] = items
		entry = items
	}
	switch field.Kind() {
	case protoreflect.BoolKind:
		p.generateBoolValidation(field, entry)
	case protoreflect.EnumKind:
		p.generateEnumValidation(field, entry)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		p.generateIntValidation(field, entry, 32)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		p.generateIntValidation(field, entry, 64)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		p.generateUintValidation(field, entry, 32)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		p.generateUintValidation(field, entry, 64)
	case protoreflect.FloatKind:
		p.generateFloatValidation(field, entry, 32)
	case protoreflect.DoubleKind:
		p.generateFloatValidation(field, entry, 64)
	case protoreflect.StringKind:
		p.generateStringValidation(field, entry)
	case protoreflect.BytesKind:
		p.generateBytesValidation(field, entry)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if field.IsMap() {
			entry["type"] = jsObject
			propertyNames := make(map[string]interface{})
			p.generateValidation(field.MapKey(), propertyNames)
			entry["propertyNames"] = propertyNames
			properties := make(map[string]interface{})
			p.generateValidation(field.MapValue(), properties)
			entry["additionalProperties"] = properties
		} else {
			p.generateMessageValidation(field, entry)
		}
	}
}

func (p *jsonSchemaGenerator) generateBoolValidation(_ protoreflect.FieldDescriptor, entry map[string]interface{}) {
	entry["type"] = jsBoolean
}

func (p *jsonSchemaGenerator) generateEnumValidation(field protoreflect.FieldDescriptor, entry map[string]interface{}) {
	var enum = make([]interface{}, 0)
	for i := 0; i < field.Enum().Values().Len(); i++ {
		enum = append(enum, field.Enum().Values().Get(i).Name())
	}
	anyOf := []map[string]interface{}{
		{"type": jsString, "enum": enum},
		{"type": jsInteger, "minimum": math.MinInt32, "maximum": math.MaxInt32},
	}
	entry["anyOf"] = anyOf
}

func (p *jsonSchemaGenerator) generateIntValidation(_ protoreflect.FieldDescriptor, entry map[string]interface{}, bitSize int) {
	entry["type"] = jsInteger
	// Use floats to handle integer overflow.
	entry["minimum"] = -math.Pow(2, float64(bitSize-1))
	entry["exclusiveMaximum"] = math.Pow(2, float64(bitSize-1))
}

func (p *jsonSchemaGenerator) generateUintValidation(_ protoreflect.FieldDescriptor, entry map[string]interface{}, bitSize int) {
	entry["type"] = jsInteger
	entry["minimum"] = 0
	entry["exclusiveMaximum"] = math.Pow(2, float64(bitSize))
}

func (p *jsonSchemaGenerator) generateFloatValidation(_ protoreflect.FieldDescriptor, entry map[string]interface{}, _ int) {
	entry["anyOf"] = []map[string]interface{}{
		{"type": jsNumber},
		{"type": jsString},
		{"type": jsString, "enum": []interface{}{"NaN", "Infinity", "-Infinity"}},
	}
}

func (p *jsonSchemaGenerator) generateStringValidation(_ protoreflect.FieldDescriptor, entry map[string]interface{}) {
	entry["type"] = jsString
}

func (p *jsonSchemaGenerator) generateBytesValidation(_ protoreflect.FieldDescriptor, entry map[string]interface{}) {
	entry["type"] = jsString
	// Set a regex to match base64 encoded strings.
	entry["pattern"] = "^[A-Za-z0-9+/]*={0,2}$"
}

func (p *jsonSchemaGenerator) generateMessageValidation(field protoreflect.FieldDescriptor, entry map[string]interface{}) {
	// Create a reference to the message type.
	entry["$ref"] = p.getID(field.Message())
	p.generate(field.Message())
}

func (p *jsonSchemaGenerator) generateWrapperValidation(result map[string]interface{}, desc protoreflect.MessageDescriptor) {
	anyOf := []map[string]interface{}{
		make(map[string]interface{}),
		p.generateField(desc.Fields().ByName("value")),
	}
	p.generateDefault(anyOf[0], desc)
	result["anyOf"] = anyOf
}

func (p *jsonSchemaGenerator) makeWktGenerators() map[protoreflect.FullName]func(map[string]interface{}, protoreflect.MessageDescriptor) {
	var result = make(map[protoreflect.FullName]func(map[string]interface{}, protoreflect.MessageDescriptor))
	result["google.protobuf.Any"] = func(result map[string]interface{}, _ protoreflect.MessageDescriptor) {
		result["type"] = jsObject
		result["properties"] = map[string]interface{}{
			"@type": map[string]interface{}{
				"type": "string",
			},
		}
	}

	result["google.protobuf.Duration"] = func(result map[string]interface{}, desc protoreflect.MessageDescriptor) {
		anyOf := []map[string]interface{}{
			make(map[string]interface{}),
			{"type": jsString, "format": "duration"},
		}
		p.generateDefault(anyOf[0], desc)
		result["anyOf"] = anyOf
	}
	result["google.protobuf.Timestamp"] = func(result map[string]interface{}, desc protoreflect.MessageDescriptor) {
		anyOf := []map[string]interface{}{
			make(map[string]interface{}),
			{"type": jsString, "format": "date-time"},
		}
		p.generateDefault(anyOf[0], desc)
		result["anyOf"] = anyOf
	}

	result["google.protobuf.Value"] = func(_ map[string]interface{}, _ protoreflect.MessageDescriptor) {}
	result["google.protobuf.ListValue"] = func(result map[string]interface{}, _ protoreflect.MessageDescriptor) {
		result["type"] = jsArray
	}
	result["google.protobuf.NullValue"] = func(result map[string]interface{}, _ protoreflect.MessageDescriptor) {
		result["type"] = jsNull
	}
	result["google.protobuf.Struct"] = func(result map[string]interface{}, _ protoreflect.MessageDescriptor) {
		result["type"] = jsObject
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

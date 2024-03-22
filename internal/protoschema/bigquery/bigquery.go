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

package bigquery

import (
	"fmt"

	"cloud.google.com/go/bigquery"
	bqproto "github.com/GoogleCloudPlatform/protoc-gen-bq-schema/protos"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/normalize"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// FileExtension is the file extension for BigQuery schema files.
	FileExtension = "TableSchema.json"
)

var (
	typeOverrideToType = map[string]bigquery.FieldType{
		"STRING":     bigquery.StringFieldType,
		"INT64":      bigquery.IntegerFieldType,
		"DOUBLE":     bigquery.FloatFieldType,
		"BYTES":      bigquery.BytesFieldType,
		"BOOL":       bigquery.BooleanFieldType,
		"TIMESTAMP":  bigquery.TimestampFieldType,
		"DATE":       bigquery.DateFieldType,
		"TIME":       bigquery.TimeFieldType,
		"DATETIME":   bigquery.DateTimeFieldType,
		"GEOGRAPHY":  bigquery.GeographyFieldType,
		"NUMERIC":    bigquery.NumericFieldType,
		"BIGNUMERIC": bigquery.BigNumericFieldType,
		"INTERVAL":   bigquery.IntervalFieldType,
		"JSON":       bigquery.JSONFieldType,
		"RECORD":     bigquery.RecordFieldType,
		"STRUCT":     bigquery.RecordFieldType,
	}
	reflectKindToType = map[protoreflect.Kind]bigquery.FieldType{
		protoreflect.BoolKind:     bigquery.BooleanFieldType,
		protoreflect.Int32Kind:    bigquery.IntegerFieldType,
		protoreflect.Sint32Kind:   bigquery.IntegerFieldType,
		protoreflect.Sfixed32Kind: bigquery.IntegerFieldType,
		protoreflect.Uint32Kind:   bigquery.IntegerFieldType,
		protoreflect.Fixed32Kind:  bigquery.IntegerFieldType,
		protoreflect.Int64Kind:    bigquery.IntegerFieldType,
		protoreflect.Sint64Kind:   bigquery.IntegerFieldType,
		protoreflect.Sfixed64Kind: bigquery.IntegerFieldType,
		protoreflect.Uint64Kind:   bigquery.BigNumericFieldType,
		protoreflect.Fixed64Kind:  bigquery.BigNumericFieldType,
		protoreflect.FloatKind:    bigquery.FloatFieldType,
		protoreflect.DoubleKind:   bigquery.FloatFieldType,
		protoreflect.StringKind:   bigquery.StringFieldType,
		protoreflect.BytesKind:    bigquery.BytesFieldType,
	}
	wktToType = map[protoreflect.FullName]bigquery.FieldType{
		"google.protobuf.Timestamp":   bigquery.TimestampFieldType,
		"google.protobuf.Duration":    bigquery.IntervalFieldType,
		"google.protobuf.StringValue": bigquery.StringFieldType,
		"google.protobuf.BytesValue":  bigquery.BytesFieldType,
		"google.protobuf.Int32Value":  bigquery.IntegerFieldType,
		"google.protobuf.Int64Value":  bigquery.IntegerFieldType,
		"google.protobuf.UInt32Value": bigquery.IntegerFieldType,
		"google.protobuf.UInt64Value": bigquery.BigNumericFieldType,
		"google.protobuf.FloatValue":  bigquery.FloatFieldType,
		"google.protobuf.DoubleValue": bigquery.FloatFieldType,
		"google.protobuf.BoolValue":   bigquery.BooleanFieldType,
		"google.protobuf.NullValue":   bigquery.JSONFieldType,
		"google.protobuf.Struct":      bigquery.JSONFieldType,
		"google.protobuf.Value":       bigquery.JSONFieldType,
		"google.protobuf.ListValue":   bigquery.JSONFieldType,
	}
)

// Generate generates a BigQuery schema for the given message descriptor.
func Generate(input protoreflect.MessageDescriptor, opts ...GenerateOptions) (bigquery.Schema, *descriptorpb.DescriptorProto, error) {
	options := new(generateOptions)
	for _, opt := range opts {
		opt.applyGenerateOptions(options)
	}
	normalizer := normalize.NewNormalizer(normalize.WithSkipTypes(
		// The well-known JSON types are not support in BigQuery so we tell the normalizer to skip them.
		"google.protobuf.Struct",
		"google.protobuf.Value",
		"google.protobuf.ListValue",
	))
	generator := &bigQuerySchemaGenerator{
		maxDepth:            options.maxDepth,
		maxRecursionDepth:   options.maxRecursionDepth,
		generateAllMessages: options.generateAllMessages,
		seen:                make(map[protoreflect.FullName]int),
		normalizer:          normalizer,
	}
	if generator.maxDepth == 0 || generator.maxDepth > 15 {
		generator.maxDepth = 15
	}
	if generator.maxRecursionDepth == 0 {
		generator.maxRecursionDepth = 1
	}
	normalized, err := generator.normalizer.Normalize(input)
	if err != nil {
		return nil, nil, err
	}
	schema, err := generator.generate(input, 0)
	if err != nil {
		return nil, nil, err
	}
	return schema, normalized, nil
}

type bigQuerySchemaGenerator struct {
	maxDepth            int
	maxRecursionDepth   int
	generateAllMessages bool
	seen                map[protoreflect.FullName]int
	normalizer          *normalize.Normalizer
}

func (p *bigQuerySchemaGenerator) generate(msgDesc protoreflect.MessageDescriptor, depth int) (bigquery.Schema, error) {
	schema, err := p.generateFields(msgDesc, depth)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func (p *bigQuerySchemaGenerator) generateFields(msgDesc protoreflect.MessageDescriptor, depth int) (bigquery.Schema, error) {
	msgOptions, err := p.getBqMessageOptions(msgDesc)
	if err != nil {
		return nil, err
	}
	if msgOptions == nil && !p.generateAllMessages {
		return nil, nil
	}
	msgPb, err := p.normalizer.FindDescriptorProto(msgDesc)
	if err != nil {
		return nil, err
	}

	result := make(bigquery.Schema, 0, msgDesc.Fields().Len())
	p.seen[msgDesc.FullName()]++
	var i int
	for ; i < len(msgPb.GetField()); i++ { // msgPb.Field may be modified in the loop.
		fieldPb := msgPb.GetField()[i]
		fieldDesc := msgDesc.Fields().ByNumber(protoreflect.FieldNumber(fieldPb.GetNumber()))
		if fieldDesc == nil {
			return nil, fmt.Errorf("could not find field %d in message %s", fieldPb.GetNumber(), msgDesc.FullName())
		}
		field, err := p.generateField(msgOptions, fieldDesc, fieldPb, depth+1)
		switch {
		case err != nil:
			return nil, err
		case field != nil:
			result = append(result, field)
		default:
			for idx, fieldPb := range msgPb.Field {
				if fieldPb.GetNumber() == int32(fieldDesc.Number()) {
					msgPb.Field = append(msgPb.Field[:idx], msgPb.Field[idx+1:]...)
					break
				}
			}
		}
	}
	p.seen[msgDesc.FullName()]--
	return result, nil
}

func (p *bigQuerySchemaGenerator) generateField(
	msgOptions *bqproto.BigQueryMessageOptions,
	fieldDesc protoreflect.FieldDescriptor,
	fieldPb *descriptorpb.FieldDescriptorProto,
	depth int,
) (*bigquery.FieldSchema, error) {
	if depth >= p.maxDepth {
		return nil, nil
	}
	fieldOptions, err := p.getBqFieldOptions(fieldDesc)
	if err != nil {
		return nil, err
	}
	if fieldOptions.GetIgnore() {
		return nil, nil
	}

	fieldSchema := p.generateFieldSchaema(msgOptions, fieldOptions, fieldDesc, fieldPb)

	switch fieldDesc.Kind() {
	default:
		if fieldType, ok := reflectKindToType[fieldDesc.Kind()]; ok {
			fieldSchema.Type = fieldType
			if fieldType == bigquery.BigNumericFieldType {
				fieldSchema.Precision = 21
				fieldSchema.Scale = 20
			}
		} else {
			return nil, fmt.Errorf("invalid type: %s", fieldDesc.Kind())
		}
	case protoreflect.EnumKind:
		if fieldDesc.Syntax() == protoreflect.Proto3 {
			fieldSchema.Type = bigquery.IntegerFieldType
		} else {
			fieldSchema.Type = bigquery.StringFieldType
		}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if fieldOptions.GetTypeOverride() != "RECORD" {
			if bqType, ok := wktToType[fieldDesc.Message().FullName()]; ok {
				if bqType == "" {
					return nil, nil
				}
				fieldSchema.Type = bqType
				break
			}
		}
		if count, ok := p.seen[fieldDesc.Message().FullName()]; ok && count >= p.maxRecursionDepth {
			return nil, nil
		}
		fieldSchema.Type = bigquery.RecordFieldType
		var err error
		fieldSchema.Schema, err = p.generateFields(fieldDesc.Message(), depth)
		if fieldDesc.IsMap() && len(fieldSchema.Schema) < 2 {
			return nil, nil
		}
		if err != nil {
			return nil, err
		} else if len(fieldSchema.Schema) == 0 {
			return nil, nil
		}
	}
	if fieldOptions.GetTypeOverride() != "" {
		if fieldType, ok := typeOverrideToType[fieldOptions.GetTypeOverride()]; ok {
			fieldSchema.Type = fieldType
		} else {
			return nil, fmt.Errorf("invalid type override: %s", fieldOptions.GetTypeOverride())
		}
	}
	return fieldSchema, nil
}

func (p *bigQuerySchemaGenerator) generateFieldSchaema(
	msgOptions *bqproto.BigQueryMessageOptions, fieldOptions *bqproto.BigQueryFieldOptions,
	fieldDesc protoreflect.FieldDescriptor, fieldPb *descriptorpb.FieldDescriptorProto) *bigquery.FieldSchema {
	fieldSchema := &bigquery.FieldSchema{}
	switch {
	case fieldOptions.GetName() != "":
		fieldSchema.Name = fieldOptions.GetName()
		fieldPb.Name = proto.String(fieldOptions.GetName())
		fieldPb.JsonName = nil
	case msgOptions.GetUseJsonNames():
		fieldSchema.Name = fieldDesc.JSONName()
		fieldPb.Name = proto.String(fieldDesc.JSONName())
	default:
		fieldSchema.Name = string(fieldDesc.Name())
	}

	switch fieldDesc.Cardinality() {
	case protoreflect.Repeated:
		fieldSchema.Repeated = true
	case protoreflect.Required:
		fieldSchema.Required = true
	case protoreflect.Optional:
		if fieldOptions.GetRequire() {
			fieldSchema.Required = true
		}
	}
	return fieldSchema
}

func (p *bigQuerySchemaGenerator) getBqFieldOptions(field protoreflect.FieldDescriptor) (*bqproto.BigQueryFieldOptions, error) {
	if !field.Options().ProtoReflect().Has(bqproto.E_Bigquery.TypeDescriptor()) {
		return nil, nil
	}
	value := field.Options().ProtoReflect().Get(bqproto.E_Bigquery.TypeDescriptor()).Message().Interface()
	if result, ok := value.(*bqproto.BigQueryFieldOptions); ok {
		return result, nil
	}
	// Serialize and deserialize to get the correct type.
	data, err := proto.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := &bqproto.BigQueryFieldOptions{}
	if err := proto.Unmarshal(data, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *bigQuerySchemaGenerator) getBqMessageOptions(msg protoreflect.MessageDescriptor) (*bqproto.BigQueryMessageOptions, error) {
	if !msg.Options().ProtoReflect().Has(bqproto.E_BigqueryOpts.TypeDescriptor()) {
		return nil, nil
	}
	value := msg.Options().ProtoReflect().Get(bqproto.E_BigqueryOpts.TypeDescriptor()).Message().Interface()
	if result, ok := value.(*bqproto.BigQueryMessageOptions); ok {
		return result, nil
	}
	// Serialize and deserialize to get the correct type.
	data, err := proto.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := &bqproto.BigQueryMessageOptions{}
	if err := proto.Unmarshal(data, result); err != nil {
		return nil, err
	}
	return result, nil
}

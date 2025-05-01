package protoschemaplugins

import (
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// GenerateJSONSchema generates a JSON schema for the given message descriptor, with protobuf field names.
func GenerateJSONSchema(
	input protoreflect.MessageDescriptor,
	opts ...jsonschema.GeneratorOption,
) map[protoreflect.FullName]map[string]any {
	return jsonschema.Generate(input, opts...)
}

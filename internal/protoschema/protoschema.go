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

package protoschema

import (
	"fmt"
	"runtime/debug"
	"strings"
	"unicode"

	"github.com/bufbuild/protoschema-plugins/internal/gen/proto/buf/protoschema"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func Version() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if ok && buildInfo != nil && buildInfo.Main.Version != "" {
		return strings.TrimSpace(buildInfo.Main.Version)
	}
	return "devel"
}

func GetFieldSchema(field protoreflect.FieldDescriptor) (*protoschema.FieldSchema, error) {
	return getExt[*protoschema.FieldSchema](field.Options(), protoschema.E_Field)
}

func GetFieldAliases(fieldSchema *protoschema.FieldSchema) ([]protoreflect.Name, []string, error) {
	aliases, err := getFieldAliases(fieldSchema)
	if err != nil {
		return nil, nil, err
	}
	return aliases, getFieldAliasesJSON(fieldSchema.GetAliasJson(), aliases), nil
}

func getFieldAliases(fieldSchema *protoschema.FieldSchema) ([]protoreflect.Name, error) {
	result := make([]protoreflect.Name, len(fieldSchema.GetAlias()))
	for i, alias := range fieldSchema.GetAlias() {
		aliasName := protoreflect.Name(alias)
		if !aliasName.IsValid() {
			return nil, fmt.Errorf("invalid alias %q", alias)
		}
		result[i] = aliasName
	}
	return result, nil
}

func getFieldAliasesJSON(jsonAliases []string, aliases []protoreflect.Name) []string {
	if len(jsonAliases) > 0 {
		// Use the provided JSON aliases if they are present.
		return jsonAliases
	}
	// Otherwise, generate JSON aliases from the field aliases.
	result := make([]string, len(aliases))
	for i, alias := range aliases {
		result[i] = JSONName(string(alias))
	}
	return result
}

func getExt[T proto.Message](options proto.Message, extType protoreflect.ExtensionType) (T, error) {
	ext := proto.GetExtension(options, extType)
	if extProto, ok := ext.(T); ok {
		return extProto, nil
	}
	var extProto T
	extDyn, ok := ext.(proto.Message)
	if !ok {
		return extProto, fmt.Errorf("unexpected extension type %T", ext)
	}
	extData, err := proto.Marshal(extDyn)
	if err != nil {
		return extProto, err
	}
	extProto, ok = extProto.ProtoReflect().New().Interface().(T)
	if !ok {
		return extProto, fmt.Errorf("unexpected extension type %T", extProto)
	}
	return extProto, proto.Unmarshal(extData, extProto)
}

// JSONName returns the default JSON name for a field with the given name.
// This mirrors the algorithm in protoc:
//
//	https://github.com/protocolbuffers/protobuf/blob/v21.3/src/google/protobuf/descriptor.cc#L95
func JSONName(name string) string {
	var jsonName []rune
	nextUpper := false
	for _, chr := range name {
		if chr == '_' {
			nextUpper = true
			continue
		}
		if nextUpper {
			nextUpper = false
			jsonName = append(jsonName, unicode.ToUpper(chr))
		} else {
			jsonName = append(jsonName, chr)
		}
	}
	return string(jsonName)
}

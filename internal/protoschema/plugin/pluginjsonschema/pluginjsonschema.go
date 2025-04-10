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

package pluginjsonschema

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bufbuild/protoplugin"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Handle implements protoplugin.Handler and is the main entry point for the plugin.
func Handle(
	_ context.Context,
	_ protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) error {
	fileDescriptors, err := request.FileDescriptorsToGenerate()
	if err != nil {
		return err
	}
	seenIdentifiers := make(map[string]bool)
	for _, fileDescriptor := range fileDescriptors {
		for i := range fileDescriptor.Messages().Len() {
			messageDescriptor := fileDescriptor.Messages().Get(i)
			protoNameSchema := jsonschema.Generate(messageDescriptor)
			if err := writeFiles(responseWriter, messageDescriptor, protoNameSchema, seenIdentifiers); err != nil {
				return err
			}
			jsonNameSchema := jsonschema.Generate(messageDescriptor, jsonschema.WithJSONNames())
			if err := writeFiles(responseWriter, messageDescriptor, jsonNameSchema, seenIdentifiers); err != nil {
				return err
			}
		}
	}

	responseWriter.SetFeatureProto3Optional()
	responseWriter.SetFeatureSupportsEditions(descriptorpb.Edition_EDITION_2023, descriptorpb.Edition_EDITION_2023)
	return nil
}

func writeFiles(
	responseWriter protoplugin.ResponseWriter,
	messageDescriptor protoreflect.MessageDescriptor,
	schema map[protoreflect.FullName]map[string]any,
	seenIdentifiers map[string]bool,
) error {
	for _, entry := range schema {
		data, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return err
		}
		identifier, ok := entry["$id"].(string)
		if !ok {
			return fmt.Errorf("expected unique id for message %q to be a string, got type %T", messageDescriptor.FullName(), entry["$id"])
		}
		if identifier == "" {
			return fmt.Errorf("expected unique id for message %q to be a non-empty string", messageDescriptor.FullName())
		}
		if seenIdentifiers[identifier] {
			continue
		}
		responseWriter.AddFile(
			identifier,
			string(data)+"\n",
		)
		seenIdentifiers[identifier] = true
	}
	return nil
}

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

package pluginjsonschema

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bufbuild/protoplugin"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema"
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

			for _, entry := range jsonschema.Generate(messageDescriptor) {
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
					string(data),
				)
				seenIdentifiers[identifier] = true
			}
		}
	}

	responseWriter.SetFeatureProto3Optional()
	return nil
}

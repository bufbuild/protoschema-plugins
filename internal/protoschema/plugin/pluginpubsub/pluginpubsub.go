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

package pluginpubsub

import (
	"context"
	"fmt"

	"github.com/bufbuild/protoplugin"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/pubsub"
)

// Handle implements protoplugin.Handler and is the main entry point for the plugin.
func Handle(
	_ context.Context,
	responseWriter *protoplugin.ResponseWriter,
	request *protoplugin.Request,
) error {
	fileDescriptors, err := request.FileDescriptorsToGenerate()
	if err != nil {
		return err
	}
	for _, fileDescriptor := range fileDescriptors {
		for i := range fileDescriptor.Messages().Len() {
			messageDescriptor := fileDescriptor.Messages().Get(i)
			data, err := pubsub.Generate(messageDescriptor)
			if err != nil {
				return err
			}
			responseWriter.AddFile(
				fmt.Sprintf("%s.%s", messageDescriptor.FullName(), pubsub.FileExtension),
				data,
			)
		}
	}

	responseWriter.SetFeatureProto3Optional()
	return nil
}

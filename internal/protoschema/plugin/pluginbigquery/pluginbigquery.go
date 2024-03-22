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

package pluginbigquery

import (
	"context"
	"path/filepath"
	"strings"

	bqproto "github.com/GoogleCloudPlatform/protoc-gen-bq-schema/protos"
	"github.com/bufbuild/protoplugin"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/bigquery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	for _, fileDescriptor := range fileDescriptors {
		for i := range fileDescriptor.Messages().Len() {
			messageDescriptor := fileDescriptor.Messages().Get(i)

			tableName := tryGetTableNameFromOptions(messageDescriptor)
			if tableName == "" {
				tableName = string(messageDescriptor.Name())
			}
			schema, _, err := bigquery.Generate(messageDescriptor)
			if err != nil {
				return err
			}
			data, err := schema.ToJSONFields()
			if err != nil {
				return err
			}
			if len(data) == 0 || string(data) == "null" {
				continue
			}
			name := tableName + "." + bigquery.FileExtension
			filename := strings.ReplaceAll(string(fileDescriptor.Package()), ".", "/")
			responseWriter.AddFile(
				filepath.Join(filename, name),
				string(data),
			)
		}
	}

	responseWriter.SetFeatureProto3Optional()
	return nil
}

func tryGetTableNameFromOptions(messageDescriptor protoreflect.MessageDescriptor) string {
	if !proto.HasExtension(messageDescriptor.Options(), bqproto.E_BigqueryOpts) {
		return ""
	}
	messageOptions, ok := proto.GetExtension(
		messageDescriptor.Options(),
		bqproto.E_BigqueryOpts,
	).(*bqproto.BigQueryMessageOptions)
	if !ok || messageOptions == nil {
		return ""
	}
	return messageOptions.GetTableName()
}

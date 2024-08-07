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

package pluginsourceinfo

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/protoplugin"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const FileExtension = ".sourceinfo.json"

// GetSourceInfoPath returns the path to the source info file for the given file descriptor.
func GetSourceInfoPath(fileDescriptor protoreflect.FileDescriptor) string {
	path := fileDescriptor.Path()
	path = strings.TrimSuffix(path, ".proto")
	return fmt.Sprintf("%s%s", path, FileExtension)
}

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
		// Write the YAML string to the response.
		data, err := GenFileContents(fileDescriptor)
		if err != nil {
			return err
		}
		if data == "" {
			continue
		}
		name := GetSourceInfoPath(fileDescriptor)
		responseWriter.AddFile(name, data)
	}

	responseWriter.SetFeatureProto3Optional()
	return nil
}

func GenFileContents(fileDescriptor protoreflect.FileDescriptor) (string, error) {
	// Convert the file descriptor to a descriptorpb.FileDescriptorProto.
	fileDescProto := protodesc.ToFileDescriptorProto(fileDescriptor)
	if len(fileDescProto.GetSourceCodeInfo().GetLocation()) == 0 {
		return "", nil
	}
	data, err := protojson.Marshal(fileDescProto.GetSourceCodeInfo())
	return string(data), err
}
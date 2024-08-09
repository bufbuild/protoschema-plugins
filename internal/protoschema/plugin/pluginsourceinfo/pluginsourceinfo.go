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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FileExtension is the file extension for the source info files.
const FileExtension = ".sourceinfo.binpb"

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
		name := GetSourceInfoPath(fileDescriptor)
		responseWriter.AddFile(name, string(data))
	}

	responseWriter.SetFeatureProto3Optional()
	return nil
}

// GetSourceInfoPath returns the path to the source info file for the given file descriptor.
func GetSourceInfoPath(fileDescriptor protoreflect.FileDescriptor) string {
	path := fileDescriptor.Path()
	path = strings.TrimSuffix(path, ".proto")
	return fmt.Sprintf("%s%s", path, FileExtension)
}

// GenFileContents generates the source info file contents for the given file descriptor.
func GenFileContents(fileDescriptor protoreflect.FileDescriptor) ([]byte, error) {
	// Convert the file descriptor to a descriptorpb.FileDescriptorProto.
	fileDescProto := protodesc.ToFileDescriptorProto(fileDescriptor)
	return proto.Marshal(fileDescProto.SourceCodeInfo)
}

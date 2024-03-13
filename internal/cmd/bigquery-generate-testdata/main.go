// Copyright 2023 Buf Technologies, Inc.
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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/bigquery"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoprint"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	if err := run(); err != nil {
		if errString := err.Error(); errString != "" {
			_, _ = fmt.Fprintln(os.Stderr, errString)
		}
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s [directory]", os.Args[0])
	}
	dirPath := os.Args[1]

	// Make sure the directory exists
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		return err
	} else if !fileInfo.IsDir() {
		return fmt.Errorf("expected %s to be a directory", dirPath)
	}

	// Generate the testdata
	for _, testDesc := range golden.GetTestDescriptors() {
		schemaPath := filepath.Join(dirPath, fmt.Sprintf("%s.%s", testDesc.FullName(), bigquery.FileExtension))
		protoPath := filepath.Join(dirPath, fmt.Sprintf("%s.bigquery.proto", testDesc.FullName()))
		schema, pbDesc, err := bigquery.Generate(testDesc)
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
		if err := golden.GenerateGolden(schemaPath, string(data)); err != nil {
			return err
		}

		// Generate the root file
		file := &descriptorpb.FileDescriptorProto{
			Name:   proto.String("bigquery.proto"),
			Syntax: proto.String("proto2"),
			Dependency: []string{
				"google/protobuf/struct.proto",
			},
			MessageType: []*descriptorpb.DescriptorProto{
				pbDesc,
			},
		}

		depMap, err := desc.CreateFileDescriptors([]*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(durationpb.File_google_protobuf_duration_proto),
			protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
			protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto),
			protodesc.ToFileDescriptorProto(wrapperspb.File_google_protobuf_wrappers_proto),
		})
		if err != nil {
			return err
		}
		deps := make([]*desc.FileDescriptor, 0, len(depMap))
		for _, dep := range depMap {
			deps = append(deps, dep)
		}

		// Print the file.
		fileDesc, err := desc.CreateFileDescriptor(file, deps...)
		if err != nil {
			return err
		}
		printer := &protoprint.Printer{
			Compact: true,
		}
		pbData, err := printer.PrintProtoToString(fileDesc)
		if err != nil {
			return err
		}
		if err := golden.GenerateGolden(protoPath, pbData); err != nil {
			return err
		}
	}

	return nil
}

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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/bufbuild/protoschema-plugins/internal/gen/proto/bufext/cel/expr/conformance/proto3"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/plugin/pluginsourceinfo"
	"google.golang.org/protobuf/reflect/protoreflect"
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
		return fmt.Errorf("usage: %s [output dir]", os.Args[0])
	}
	outputDir := os.Args[1]
	// Make sure the directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	testFiles, err := golden.GetTestFiles("./internal/testdata")
	if err != nil {
		return err
	}
	// TODO: Use normal the plugin to generate golden files
	includePrefixes := []string{
		filepath.FromSlash("buf/protoschema/test/"),
		filepath.FromSlash("bufext/cel/expr/conformance/proto3/"),
	}
	err = nil
	testFiles.RangeFiles(func(testDesc protoreflect.FileDescriptor) bool {
		if !shouldIncludeFile(testDesc, includePrefixes) {
			return true
		}
		fileName := pluginsourceinfo.GetSourceInfoPath(testDesc)
		filePath := filepath.Join(outputDir, fileName)
		// Create any missing directories
		if err = os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			err = fmt.Errorf("failed to create directory for %s: %w", filePath, err)
			return false
		}
		var data []byte
		data, err = pluginsourceinfo.GenFileContents(testDesc)
		if err != nil {
			err = fmt.Errorf("failed to generate source info for %s: %w", testDesc.FullName(), err)
			return false
		}
		if err = os.WriteFile(filePath, data, 0600); err != nil {
			err = fmt.Errorf("failed to write source info to %s: %w", filePath, err)
			return false
		}
		return true
	})
	return err
}

func shouldIncludeFile(fileDesc protoreflect.FileDescriptor, includePrefixes []string) bool {
	for _, prefix := range includePrefixes {
		if strings.HasPrefix(fileDesc.Path(), prefix) {
			return true
		}
	}
	return false
}

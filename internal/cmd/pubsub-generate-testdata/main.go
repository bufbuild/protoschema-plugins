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

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/pubsub"
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
	testDescs, err := golden.GetTestDescriptors("./internal/testdata")
	if err != nil {
		return err
	}
	for _, testDesc := range testDescs {
		filePath := filepath.Join(dirPath, fmt.Sprintf("%s.%s", testDesc.FullName(), pubsub.FileExtension))
		data, err := pubsub.Generate(testDesc)
		if err != nil {
			return err
		}
		if err := golden.GenerateGolden(filePath, data); err != nil {
			return err
		}
	}

	return nil
}

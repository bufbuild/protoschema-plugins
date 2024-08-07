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
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/plugin/pluginsourceinfo"
	"github.com/jhump/protoreflect/desc/sourceinfo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// SourceInfoGlobalTypes is a replacement for protoregistry.GlobalTypes that includes
	// all registered source info.
	SourceInfoGlobalTypes = sourceinfo.GlobalTypes
	// SourceInfoGlobalFiles is a replacement for protoregistry.GlobalFiles that includes
	// all registered source info.
	SourceInfoGlobalFiles = sourceinfo.GlobalFiles
)

// RegisterAllSourceInfo registers all sourceinfo files under the given output path.
func RegisterAllSourceInfo(outputPath string) error {
	return filepath.Walk(outputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, pluginsourceinfo.FileExtension) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sourceInfo := &descriptorpb.SourceCodeInfo{}
		if err := protojson.Unmarshal(data, sourceInfo); err != nil {
			return err
		}
		protoPath, err := filepath.Rel(outputPath, path)
		if err != nil {
			return err
		}
		protoPath = filepath.ToSlash(protoPath)
		protoPath = strings.TrimSuffix(protoPath, pluginsourceinfo.FileExtension)
		protoPath += ".proto"
		sourceinfo.RegisterSourceInfo(protoPath, sourceInfo)
		return nil
	})
}

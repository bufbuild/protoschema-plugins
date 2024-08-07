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

package sourceinfo

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/plugin/pluginsourceinfo"
	"github.com/jhump/protoreflect/desc/sourceinfo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// SourceInfoGlobalTypes is a replacement for protoregistry.GlobalTypes that includes
	// all registered source info.
	GlobalFiles protodesc.Resolver = sourceinfo.GlobalFiles

	// SourceInfoGlobalFiles is a replacement for protoregistry.GlobalFiles that includes
	// all registered source info.
	GlobalTypes interface {
		protoregistry.MessageTypeResolver
		protoregistry.ExtensionTypeResolver
		RangeExtensionsByMessage(message protoreflect.FullName, f func(protoreflect.ExtensionType) bool)
	} = sourceinfo.GlobalTypes
)

// RegisterAllSourceInfo registers all sourceinfo files under the given output path.
func RegisterAll(root string) error {
	return RegisterAllFS(os.DirFS(root), ".")
}

// RegisterAllSourceInfoFS registers all sourceinfo files under the given output path, using the given fs.FS.
func RegisterAllFS(fsys fs.FS, root string) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, pluginsourceinfo.FileExtension) {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return registerSourceInfoData(relPath, data)
	})
}

func registerSourceInfoData(path string, data []byte) error {
	sourceInfo := &descriptorpb.SourceCodeInfo{}
	if err := proto.Unmarshal(data, sourceInfo); err != nil {
		return err
	}
	protoPath := filepath.ToSlash(path)
	protoPath = strings.TrimSuffix(protoPath, pluginsourceinfo.FileExtension)
	protoPath += ".proto"
	sourceinfo.RegisterSourceInfo(protoPath, sourceInfo)
	return nil
}

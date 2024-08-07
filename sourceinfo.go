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
	"embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/plugin/pluginsourceinfo"
	"github.com/jhump/protoreflect/desc/sourceinfo"
	"google.golang.org/protobuf/proto"
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
func RegisterAllSourceInfo(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return registerSourceInfoData(relPath, data)
	})
}

// RegisterEmbeddedSourceInfo registers all sourceinfo files embedded in the given FS.
func RegisterEmbeddedSourceInfo(files embed.FS, root string) error {
	contents, err := files.ReadDir(root)
	if err != nil {
		return err
	}
	return registerEmbeddedDirs(files, root, contents, "")
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

func registerEmbeddedDirs(files embed.FS, root string, dir []os.DirEntry, prefix string) error {
	for _, entry := range dir {
		fullName := filepath.Join(prefix, entry.Name())
		if entry.IsDir() {
			subDir, err := files.ReadDir(filepath.Join(root, fullName))
			if err != nil {
				return err
			}
			if err := registerEmbeddedDirs(files, root, subDir, fullName); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(fullName, pluginsourceinfo.FileExtension) {
			continue
		}
		data, err := files.ReadFile(filepath.Join(root, fullName))
		if err != nil {
			return err
		}
		if err := registerSourceInfoData(fullName, data); err != nil {
			return err
		}
	}
	return nil
}

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

package pluginsourceinfo_test

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/protoplugin"
	_ "github.com/bufbuild/protoschema-plugins/internal/gen/proto/buf/protoschema/test/v1"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/plugin/pluginsourceinfo"
	"github.com/bufbuild/protoschema-plugins/sourceinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestSourceInfoHandler(t *testing.T) {
	t.Parallel()

	goldenPath := filepath.FromSlash("../../../testdata/sourceinfo")
	inputImage := filepath.FromSlash("../../../testdata/codegenrequest/input.json")

	by, err := os.ReadFile(inputImage)
	require.NoError(t, err)
	protoImage := new(imagev1.Image)
	err = protojson.Unmarshal(by, protoImage)
	require.NoError(t, err)
	image, err := bufimage.NewImageForProto(protoImage)
	require.NoError(t, err)
	codeGeneratorRequest, err := bufimage.ImageToCodeGeneratorRequest(image, "", nil, false, false)
	require.NoError(t, err)

	request, err := protoencoding.NewWireMarshaler().Marshal(codeGeneratorRequest)
	require.NoError(t, err)
	stdin := bytes.NewReader(request)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	err = protoplugin.Run(
		context.Background(),
		protoplugin.Env{
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
		},
		protoplugin.HandlerFunc(pluginsourceinfo.Handle),
	)
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	response := new(pluginpb.CodeGeneratorResponse)
	err = protoencoding.NewWireUnmarshaler(nil).Unmarshal(stdout.Bytes(), response)
	require.NoError(t, err)

	wantFiles := make([]string, 0, len(response.File))
	for _, file := range response.File {
		wantFiles = append(wantFiles, filepath.FromSlash(file.GetName()))
	}
	slices.Sort(wantFiles)
	require.Equal(t, wantFiles, gatherGoldenFiles(t, goldenPath))

	for _, file := range response.File {
		filename := path.Join(goldenPath, file.GetName())
		want, err := os.ReadFile(filename)
		require.NoError(t, err)

		actualInfo := &descriptorpb.SourceCodeInfo{}
		err = proto.Unmarshal([]byte(file.GetContent()), actualInfo)
		require.NoError(t, err)

		wantInfo := &descriptorpb.SourceCodeInfo{}
		err = proto.Unmarshal(want, wantInfo)
		require.NoError(t, err)

		require.True(t, proto.Equal(wantInfo, actualInfo), "file %s did not match", file.GetName())
	}

	err = sourceinfo.RegisterAll(goldenPath)
	require.NoError(t, err)

	msgType, err := sourceinfo.GlobalTypes.FindMessageByName(
		"buf.protoschema.test.v1.NestedReference",
	)
	require.NoError(t, err)
	parentFile := msgType.Descriptor().ParentFile()
	locs := parentFile.SourceLocations().ByDescriptor(msgType.Descriptor())
	assert.Equal(t, " A message comment.\n", locs.LeadingComments)
	fieldDesc := msgType.Descriptor().Fields().ByName("nested_message")
	require.NotNil(t, fieldDesc)
	locs = parentFile.SourceLocations().ByDescriptor(fieldDesc)
	assert.Equal(t, " A field comment.\n", locs.LeadingComments)
}

func gatherGoldenFiles(t *testing.T, dir string) []string {
	t.Helper()

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, pluginsourceinfo.FileExtension) {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	require.NoError(t, err)
	slices.Sort(files)
	return files
}

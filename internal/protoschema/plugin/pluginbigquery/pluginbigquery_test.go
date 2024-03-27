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
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/protoplugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestBigQueryHandler(t *testing.T) {
	t.Parallel()

	goldenPath := filepath.FromSlash("../../../testdata/bigquery")
	inputImage := filepath.FromSlash("../../../testdata/codegenrequest/input.json")

	by, err := os.ReadFile(inputImage)
	require.NoError(t, err)
	protoImage := new(imagev1.Image)
	err = protojson.Unmarshal(by, protoImage)
	require.NoError(t, err)
	image, err := bufimage.NewImageForProto(protoImage)
	require.NoError(t, err)
	codeGeneratorRequest := bufimage.ImageToCodeGeneratorRequest(image, "", nil, false, false)

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
		protoplugin.HandlerFunc(Handle),
	)
	require.NoError(t, err)
	require.Empty(t, stderr.String())

	response := new(pluginpb.CodeGeneratorResponse)
	err = protoencoding.NewWireUnmarshaler(nil).Unmarshal(stdout.Bytes(), response)
	require.NoError(t, err)

	for _, file := range response.File {
		filename := path.Join(goldenPath, file.GetName())
		want, err := os.ReadFile(filename)
		require.NoError(t, err)
		require.Equal(t, string(want), file.GetContent())
	}
}

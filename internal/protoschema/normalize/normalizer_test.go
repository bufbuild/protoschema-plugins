// Copyright 2024-2026 Buf Technologies, Inc.
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

package normalize

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

// buildThreeLevelDescriptor creates a file descriptor with 3-level nested messages:
//
//	Organization (Level 1)
//	  Department (Level 2)
//	    Team (Level 3)
func buildThreeLevelDescriptor(t *testing.T) *descriptorpb.FileDescriptorProto {
	t.Helper()
	return &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Organization"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("department"),
						Number:   int32Ptr(1),
						Type:     enumPtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".test.v1.Organization.Department"),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("Department"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     strPtr("teams"),
								Number:   int32Ptr(1),
								Type:     enumPtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
								TypeName: strPtr(".test.v1.Organization.Department.Team"),
								Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
							},
						},
						NestedType: []*descriptorpb.DescriptorProto{
							{
								Name: strPtr("Team"),
								Field: []*descriptorpb.FieldDescriptorProto{
									{
										Name:   strPtr("name"),
										Number: int32Ptr(1),
										Type:   enumPtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
										Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestFindRootAndPath_ThreeLevelNesting(t *testing.T) {
	t.Parallel()
	fd := buildThreeLevelDescriptor(t)

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}})
	require.NoError(t, err)

	fileDesc, err := files.FindFileByPath("test.proto")
	require.NoError(t, err)

	organization := fileDesc.Messages().ByName("Organization")
	require.NotNil(t, organization)

	// Retrieve the 3-level nested Team message.
	team := organization.Messages().ByName("Department").Messages().ByName("Team")
	require.NotNil(t, team)

	root, path := findRootAndPath(team)
	require.Equal(t, "Organization", string(root.Name()))

	// The path must be in top-down order: Department, Team.
	// Before the fix, this was reversed: Team, Department.
	require.Equal(t, []string{"Department", "Team"}, path)
}

func TestNormalize_ThreeLevelNesting(t *testing.T) {
	t.Parallel()
	fd := buildThreeLevelDescriptor(t)

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}})
	require.NoError(t, err)

	fileDesc, err := files.FindFileByPath("test.proto")
	require.NoError(t, err)

	organization := fileDesc.Messages().ByName("Organization")
	require.NotNil(t, organization)

	normalizer := NewNormalizer()
	result, err := normalizer.Normalize(organization)
	require.NoError(t, err)

	// The Department nested type should reference Team correctly.
	departmentType := findNestedMessage(result, "Department")
	require.NotNil(t, departmentType, "Department nested type should exist")

	var teamsField *descriptorpb.FieldDescriptorProto
	for _, f := range departmentType.GetField() {
		if f.GetName() == "teams" {
			teamsField = f
			break
		}
	}
	require.NotNil(t, teamsField, "teams field should exist")

	// The type name should be in top-down nesting order, not reversed.
	require.Equal(t, "Organization.Department.Team", teamsField.GetTypeName())
}

func strPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32 { return &i }

func enumPtr(e descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &e
}

func labelPtr(l descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &l
}

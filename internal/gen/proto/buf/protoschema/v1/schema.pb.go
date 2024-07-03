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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: buf/protoschema/v1/schema.proto

package protoschemav1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type FieldSchema struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Additional names for the field.
	Alias []string `protobuf:"bytes,1,rep,name=alias,proto3" json:"alias,omitempty"`
	// Aditional json names for the field.
	//
	// If empty, automatically generated from `alias`.
	AliasJson []string `protobuf:"bytes,2,rep,name=alias_json,json=aliasJson,proto3" json:"alias_json,omitempty"`
}

func (x *FieldSchema) Reset() {
	*x = FieldSchema{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_protoschema_v1_schema_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FieldSchema) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FieldSchema) ProtoMessage() {}

func (x *FieldSchema) ProtoReflect() protoreflect.Message {
	mi := &file_buf_protoschema_v1_schema_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FieldSchema.ProtoReflect.Descriptor instead.
func (*FieldSchema) Descriptor() ([]byte, []int) {
	return file_buf_protoschema_v1_schema_proto_rawDescGZIP(), []int{0}
}

func (x *FieldSchema) GetAlias() []string {
	if x != nil {
		return x.Alias
	}
	return nil
}

func (x *FieldSchema) GetAliasJson() []string {
	if x != nil {
		return x.AliasJson
	}
	return nil
}

var file_buf_protoschema_v1_schema_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*FieldSchema)(nil),
		Field:         1162,
		Name:          "buf.protoschema.v1.field",
		Tag:           "bytes,1162,opt,name=field",
		Filename:      "buf/protoschema/v1/schema.proto",
	},
}

// Extension fields to descriptorpb.FieldOptions.
var (
	// optional buf.protoschema.v1.FieldSchema field = 1162;
	E_Field = &file_buf_protoschema_v1_schema_proto_extTypes[0]
)

var File_buf_protoschema_v1_schema_proto protoreflect.FileDescriptor

var file_buf_protoschema_v1_schema_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d,
	0x61, 0x2f, 0x76, 0x31, 0x2f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x12, 0x62, 0x75, 0x66, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65,
	0x6d, 0x61, 0x2e, 0x76, 0x31, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f,
	0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x42, 0x0a, 0x0b, 0x46, 0x69, 0x65, 0x6c, 0x64,
	0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x12, 0x14, 0x0a, 0x05, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x12, 0x1d, 0x0a, 0x0a,
	0x61, 0x6c, 0x69, 0x61, 0x73, 0x5f, 0x6a, 0x73, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x09, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x4a, 0x73, 0x6f, 0x6e, 0x3a, 0x55, 0x0a, 0x05, 0x66,
	0x69, 0x65, 0x6c, 0x64, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x18, 0x8a, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2e, 0x76, 0x31, 0x2e,
	0x46, 0x69, 0x65, 0x6c, 0x64, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x52, 0x05, 0x66, 0x69, 0x65,
	0x6c, 0x64, 0x42, 0xec, 0x01, 0x0a, 0x16, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2e, 0x76, 0x31, 0x42, 0x0b, 0x53,
	0x63, 0x68, 0x65, 0x6d, 0x61, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x5b, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c,
	0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2d, 0x70, 0x6c,
	0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x67,
	0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2f, 0x76, 0x31, 0x3b, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x42, 0x50, 0x58, 0xaa,
	0x02, 0x12, 0x42, 0x75, 0x66, 0x2e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d,
	0x61, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x12, 0x42, 0x75, 0x66, 0x5c, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x1e, 0x42, 0x75, 0x66, 0x5c,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5c, 0x56, 0x31, 0x5c, 0x47,
	0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x14, 0x42, 0x75, 0x66,
	0x3a, 0x3a, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x3a, 0x3a, 0x56,
	0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_protoschema_v1_schema_proto_rawDescOnce sync.Once
	file_buf_protoschema_v1_schema_proto_rawDescData = file_buf_protoschema_v1_schema_proto_rawDesc
)

func file_buf_protoschema_v1_schema_proto_rawDescGZIP() []byte {
	file_buf_protoschema_v1_schema_proto_rawDescOnce.Do(func() {
		file_buf_protoschema_v1_schema_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_protoschema_v1_schema_proto_rawDescData)
	})
	return file_buf_protoschema_v1_schema_proto_rawDescData
}

var file_buf_protoschema_v1_schema_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_buf_protoschema_v1_schema_proto_goTypes = []any{
	(*FieldSchema)(nil),               // 0: buf.protoschema.v1.FieldSchema
	(*descriptorpb.FieldOptions)(nil), // 1: google.protobuf.FieldOptions
}
var file_buf_protoschema_v1_schema_proto_depIdxs = []int32{
	1, // 0: buf.protoschema.v1.field:extendee -> google.protobuf.FieldOptions
	0, // 1: buf.protoschema.v1.field:type_name -> buf.protoschema.v1.FieldSchema
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	1, // [1:2] is the sub-list for extension type_name
	0, // [0:1] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_buf_protoschema_v1_schema_proto_init() }
func file_buf_protoschema_v1_schema_proto_init() {
	if File_buf_protoschema_v1_schema_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_protoschema_v1_schema_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*FieldSchema); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_protoschema_v1_schema_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 1,
			NumServices:   0,
		},
		GoTypes:           file_buf_protoschema_v1_schema_proto_goTypes,
		DependencyIndexes: file_buf_protoschema_v1_schema_proto_depIdxs,
		MessageInfos:      file_buf_protoschema_v1_schema_proto_msgTypes,
		ExtensionInfos:    file_buf_protoschema_v1_schema_proto_extTypes,
	}.Build()
	File_buf_protoschema_v1_schema_proto = out.File
	file_buf_protoschema_v1_schema_proto_rawDesc = nil
	file_buf_protoschema_v1_schema_proto_goTypes = nil
	file_buf_protoschema_v1_schema_proto_depIdxs = nil
}

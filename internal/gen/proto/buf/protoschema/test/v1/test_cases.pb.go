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
// source: buf/protoschema/test/v1/test_cases.proto

package testv1

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	proto3 "github.com/bufbuild/protoschema-plugins/internal/gen/proto/bufext/cel/expr/conformance/proto3"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type NestedReference struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NestedMessage *proto3.TestAllTypes_NestedMessage `protobuf:"bytes,1,opt,name=nested_message,json=nestedMessage,proto3" json:"nested_message,omitempty"`
}

func (x *NestedReference) Reset() {
	*x = NestedReference{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_protoschema_test_v1_test_cases_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NestedReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NestedReference) ProtoMessage() {}

func (x *NestedReference) ProtoReflect() protoreflect.Message {
	mi := &file_buf_protoschema_test_v1_test_cases_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NestedReference.ProtoReflect.Descriptor instead.
func (*NestedReference) Descriptor() ([]byte, []int) {
	return file_buf_protoschema_test_v1_test_cases_proto_rawDescGZIP(), []int{0}
}

func (x *NestedReference) GetNestedMessage() *proto3.TestAllTypes_NestedMessage {
	if x != nil {
		return x.NestedMessage
	}
	return nil
}

type CustomOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Int32Field []int32 `protobuf:"varint,1,rep,packed,name=int32_field,json=int32Field,proto3" json:"int32_field,omitempty"`
	// Types that are assignable to Kind:
	//
	//	*CustomOptions_StringField
	Kind isCustomOptions_Kind `protobuf_oneof:"kind"`
}

func (x *CustomOptions) Reset() {
	*x = CustomOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_protoschema_test_v1_test_cases_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CustomOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CustomOptions) ProtoMessage() {}

func (x *CustomOptions) ProtoReflect() protoreflect.Message {
	mi := &file_buf_protoschema_test_v1_test_cases_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CustomOptions.ProtoReflect.Descriptor instead.
func (*CustomOptions) Descriptor() ([]byte, []int) {
	return file_buf_protoschema_test_v1_test_cases_proto_rawDescGZIP(), []int{1}
}

func (x *CustomOptions) GetInt32Field() []int32 {
	if x != nil {
		return x.Int32Field
	}
	return nil
}

func (m *CustomOptions) GetKind() isCustomOptions_Kind {
	if m != nil {
		return m.Kind
	}
	return nil
}

func (x *CustomOptions) GetStringField() string {
	if x, ok := x.GetKind().(*CustomOptions_StringField); ok {
		return x.StringField
	}
	return ""
}

type isCustomOptions_Kind interface {
	isCustomOptions_Kind()
}

type CustomOptions_StringField struct {
	StringField string `protobuf:"bytes,2,opt,name=string_field,json=stringField,proto3,oneof"`
}

func (*CustomOptions_StringField) isCustomOptions_Kind() {}

type IgnoreField struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StringField string `protobuf:"bytes,1,opt,name=string_field,json=stringField,proto3" json:"string_field,omitempty"` // jsonschema:ignore
	// jsonschema:ignore
	Int32Field int32  `protobuf:"varint,2,opt,name=int32_field,json=int32Field,proto3" json:"int32_field,omitempty"`
	BoolField  bool   `protobuf:"varint,3,opt,name=bool_field,json=boolField,proto3" json:"bool_field,omitempty"`
	BytesField []byte `protobuf:"bytes,4,opt,name=bytes_field,json=bytesField,proto3" json:"bytes_field,omitempty"` // jsonschema:hide
	// jsonschema:hide
	NestedReference *NestedReference `protobuf:"bytes,5,opt,name=nested_reference,json=nestedReference,proto3" json:"nested_reference,omitempty"`
}

func (x *IgnoreField) Reset() {
	*x = IgnoreField{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_protoschema_test_v1_test_cases_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IgnoreField) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IgnoreField) ProtoMessage() {}

func (x *IgnoreField) ProtoReflect() protoreflect.Message {
	mi := &file_buf_protoschema_test_v1_test_cases_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IgnoreField.ProtoReflect.Descriptor instead.
func (*IgnoreField) Descriptor() ([]byte, []int) {
	return file_buf_protoschema_test_v1_test_cases_proto_rawDescGZIP(), []int{2}
}

func (x *IgnoreField) GetStringField() string {
	if x != nil {
		return x.StringField
	}
	return ""
}

func (x *IgnoreField) GetInt32Field() int32 {
	if x != nil {
		return x.Int32Field
	}
	return 0
}

func (x *IgnoreField) GetBoolField() bool {
	if x != nil {
		return x.BoolField
	}
	return false
}

func (x *IgnoreField) GetBytesField() []byte {
	if x != nil {
		return x.BytesField
	}
	return nil
}

func (x *IgnoreField) GetNestedReference() *NestedReference {
	if x != nil {
		return x.NestedReference
	}
	return nil
}

var File_buf_protoschema_test_v1_test_cases_proto protoreflect.FileDescriptor

var file_buf_protoschema_test_v1_test_cases_proto_rawDesc = []byte{
	0x0a, 0x28, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d,
	0x61, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x63,
	0x61, 0x73, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x17, 0x62, 0x75, 0x66, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74,
	0x2e, 0x76, 0x31, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x37, 0x62, 0x75, 0x66, 0x65, 0x78, 0x74, 0x2f, 0x63, 0x65, 0x6c, 0x2f, 0x65, 0x78, 0x70,
	0x72, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x6e, 0x63, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x61, 0x6c, 0x6c, 0x5f, 0x74, 0x79,
	0x70, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x78, 0x0a, 0x0f, 0x4e, 0x65, 0x73,
	0x74, 0x65, 0x64, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x65, 0x0a, 0x0e,
	0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x3e, 0x2e, 0x62, 0x75, 0x66, 0x65, 0x78, 0x74, 0x2e, 0x63, 0x65,
	0x6c, 0x2e, 0x65, 0x78, 0x70, 0x72, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x6e,
	0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x41, 0x6c,
	0x6c, 0x54, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x52, 0x0d, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x22, 0xf4, 0x01, 0x0a, 0x0d, 0x43, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x4f, 0x0a, 0x0b, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x5f, 0x66,
	0x69, 0x65, 0x6c, 0x64, 0x18, 0x01, 0x20, 0x03, 0x28, 0x05, 0x42, 0x2e, 0xba, 0x48, 0x29, 0xba,
	0x01, 0x26, 0x0a, 0x0e, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x5f,
	0x69, 0x64, 0x12, 0x0c, 0x6d, 0x75, 0x73, 0x74, 0x20, 0x62, 0x65, 0x20, 0x74, 0x72, 0x75, 0x65,
	0x1a, 0x06, 0x31, 0x20, 0x3d, 0x3d, 0x20, 0x31, 0x10, 0x01, 0x52, 0x0a, 0x69, 0x6e, 0x74, 0x33,
	0x32, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x52, 0x0a, 0x0c, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x2d, 0xba, 0x48,
	0x2a, 0xba, 0x01, 0x27, 0x0a, 0x0f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x5f, 0x66, 0x69, 0x65,
	0x6c, 0x64, 0x5f, 0x69, 0x64, 0x12, 0x0c, 0x6d, 0x75, 0x73, 0x74, 0x20, 0x62, 0x65, 0x20, 0x74,
	0x72, 0x75, 0x65, 0x1a, 0x06, 0x31, 0x20, 0x3d, 0x3d, 0x20, 0x31, 0x48, 0x00, 0x52, 0x0b, 0x73,
	0x74, 0x72, 0x69, 0x6e, 0x67, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x3a, 0x2f, 0xba, 0x48, 0x2a, 0x1a,
	0x28, 0x0a, 0x10, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x5f, 0x69, 0x64, 0x12, 0x0c, 0x6d, 0x75, 0x73, 0x74, 0x20, 0x62, 0x65, 0x20, 0x74, 0x72, 0x75,
	0x65, 0x1a, 0x06, 0x31, 0x20, 0x3d, 0x3d, 0x20, 0x31, 0x10, 0x01, 0x42, 0x0d, 0x0a, 0x04, 0x6b,
	0x69, 0x6e, 0x64, 0x12, 0x05, 0xba, 0x48, 0x02, 0x08, 0x01, 0x22, 0xe6, 0x01, 0x0a, 0x0b, 0x49,
	0x67, 0x6e, 0x6f, 0x72, 0x65, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x73, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0b, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1f, 0x0a,
	0x0b, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x0a, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1d,
	0x0a, 0x0a, 0x62, 0x6f, 0x6f, 0x6c, 0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x09, 0x62, 0x6f, 0x6f, 0x6c, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1f, 0x0a,
	0x0b, 0x62, 0x79, 0x74, 0x65, 0x73, 0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x0a, 0x62, 0x79, 0x74, 0x65, 0x73, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x53,
	0x0a, 0x10, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e,
	0x63, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e,
	0x76, 0x31, 0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e,
	0x63, 0x65, 0x52, 0x0f, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65,
	0x6e, 0x63, 0x65, 0x42, 0x87, 0x02, 0x0a, 0x1b, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2e, 0x74, 0x65, 0x73, 0x74,
	0x2e, 0x76, 0x31, 0x42, 0x0e, 0x54, 0x65, 0x73, 0x74, 0x43, 0x61, 0x73, 0x65, 0x73, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2d, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d,
	0x61, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x3b, 0x74, 0x65, 0x73, 0x74, 0x76, 0x31,
	0xa2, 0x02, 0x03, 0x42, 0x50, 0x54, 0xaa, 0x02, 0x17, 0x42, 0x75, 0x66, 0x2e, 0x50, 0x72, 0x6f,
	0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x2e, 0x56, 0x31,
	0xca, 0x02, 0x17, 0x42, 0x75, 0x66, 0x5c, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65,
	0x6d, 0x61, 0x5c, 0x54, 0x65, 0x73, 0x74, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x23, 0x42, 0x75, 0x66,
	0x5c, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5c, 0x54, 0x65, 0x73,
	0x74, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0xea, 0x02, 0x1a, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x63, 0x68,
	0x65, 0x6d, 0x61, 0x3a, 0x3a, 0x54, 0x65, 0x73, 0x74, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_protoschema_test_v1_test_cases_proto_rawDescOnce sync.Once
	file_buf_protoschema_test_v1_test_cases_proto_rawDescData = file_buf_protoschema_test_v1_test_cases_proto_rawDesc
)

func file_buf_protoschema_test_v1_test_cases_proto_rawDescGZIP() []byte {
	file_buf_protoschema_test_v1_test_cases_proto_rawDescOnce.Do(func() {
		file_buf_protoschema_test_v1_test_cases_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_protoschema_test_v1_test_cases_proto_rawDescData)
	})
	return file_buf_protoschema_test_v1_test_cases_proto_rawDescData
}

var file_buf_protoschema_test_v1_test_cases_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_buf_protoschema_test_v1_test_cases_proto_goTypes = []any{
	(*NestedReference)(nil),                   // 0: buf.protoschema.test.v1.NestedReference
	(*CustomOptions)(nil),                     // 1: buf.protoschema.test.v1.CustomOptions
	(*IgnoreField)(nil),                       // 2: buf.protoschema.test.v1.IgnoreField
	(*proto3.TestAllTypes_NestedMessage)(nil), // 3: bufext.cel.expr.conformance.proto3.TestAllTypes.NestedMessage
}
var file_buf_protoschema_test_v1_test_cases_proto_depIdxs = []int32{
	3, // 0: buf.protoschema.test.v1.NestedReference.nested_message:type_name -> bufext.cel.expr.conformance.proto3.TestAllTypes.NestedMessage
	0, // 1: buf.protoschema.test.v1.IgnoreField.nested_reference:type_name -> buf.protoschema.test.v1.NestedReference
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_buf_protoschema_test_v1_test_cases_proto_init() }
func file_buf_protoschema_test_v1_test_cases_proto_init() {
	if File_buf_protoschema_test_v1_test_cases_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_protoschema_test_v1_test_cases_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*NestedReference); i {
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
		file_buf_protoschema_test_v1_test_cases_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*CustomOptions); i {
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
		file_buf_protoschema_test_v1_test_cases_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*IgnoreField); i {
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
	file_buf_protoschema_test_v1_test_cases_proto_msgTypes[1].OneofWrappers = []any{
		(*CustomOptions_StringField)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_protoschema_test_v1_test_cases_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_protoschema_test_v1_test_cases_proto_goTypes,
		DependencyIndexes: file_buf_protoschema_test_v1_test_cases_proto_depIdxs,
		MessageInfos:      file_buf_protoschema_test_v1_test_cases_proto_msgTypes,
	}.Build()
	File_buf_protoschema_test_v1_test_cases_proto = out.File
	file_buf_protoschema_test_v1_test_cases_proto_rawDesc = nil
	file_buf_protoschema_test_v1_test_cases_proto_goTypes = nil
	file_buf_protoschema_test_v1_test_cases_proto_depIdxs = nil
}

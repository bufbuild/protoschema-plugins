// Copyright 2024-2025 Buf Technologies, Inc.
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
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Normalizer is a normalizer for descriptor protos.
type Normalizer struct {
	skipTypes     []string
	rootDesc      protoreflect.MessageDescriptor
	rootPb        *descriptorpb.DescriptorProto
	nameToMangled map[string]string
	mangledToName map[string]string
	inlineMsgMap  map[string]*descriptorpb.DescriptorProto
	inlineMsgs    []*descriptorpb.DescriptorProto
}

// NewNormalizer returns a new Normalizer.
func NewNormalizer(options ...NormalizerOption) *Normalizer {
	normalizer := &Normalizer{}
	for _, option := range options {
		option(normalizer)
	}
	return normalizer
}

// Normalize returns the normalized descriptor proto for the given message descriptor.
func (n *Normalizer) Normalize(rootDesc protoreflect.MessageDescriptor) (*descriptorpb.DescriptorProto, error) {
	n.rootDesc = rootDesc
	n.nameToMangled = map[string]string{
		string(n.rootDesc.FullName()): string(n.rootDesc.Name()),
	}
	for _, name := range n.skipTypes {
		n.nameToMangled[name] = name
	}
	n.mangledToName = map[string]string{}
	n.inlineMsgMap = map[string]*descriptorpb.DescriptorProto{}

	if n.rootDesc.ParentFile() != n.rootDesc.Parent() {
		return nil, errors.New("message must be top-level")
	}
	n.rootPb = protodesc.ToDescriptorProto(n.rootDesc)
	if err := n.inlineRefs(n.rootPb, n.rootDesc); err != nil {
		return nil, err
	}
	n.inlineMsgMap[string(n.rootDesc.FullName())] = n.rootPb
	n.rootPb.NestedType = append(n.rootPb.NestedType, n.inlineMsgs...)
	return n.rootPb, nil
}

// FindDescriptorProto finds the descriptor proto for the given message descriptor.
func (n *Normalizer) FindDescriptorProto(msgDesc protoreflect.Descriptor) (*descriptorpb.DescriptorProto, error) {
	rootMsg, path := findRootAndPath(msgDesc)
	msg, ok := n.inlineMsgMap[string(rootMsg.FullName())]
	if !ok {
		return nil, fmt.Errorf("could not find message %s", rootMsg.FullName())
	}
	for _, pathPart := range path {
		msg = findNestedMessage(msg, pathPart)
	}
	return msg, nil
}

func (n *Normalizer) inlineMessage(msgDesc protoreflect.MessageDescriptor) error {
	msg := protodesc.ToDescriptorProto(msgDesc)
	msg.Name = proto.String(n.addMangledName(string(msgDesc.FullName()), ""))
	if err := n.inlineRefs(msg, msgDesc); err != nil {
		return err
	}
	n.inlineMsgMap[string(msgDesc.FullName())] = msg
	n.inlineMsgs = append(n.inlineMsgs, msg)
	return nil
}

func (n *Normalizer) inlineEnum(enumDesc protoreflect.EnumDescriptor) error {
	enum := protodesc.ToEnumDescriptorProto(enumDesc)
	// Create a message to hold the enum.
	msg := &descriptorpb.DescriptorProto{
		Name: proto.String(n.addMangledName(string(enumDesc.FullName()), string(enumDesc.Name()))),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			enum,
		},
	}
	n.inlineMsgMap[string(enumDesc.FullName())] = msg
	n.inlineMsgs = append(n.inlineMsgs, msg)
	return nil
}

func (n *Normalizer) inlineRefs(msgDescPb *descriptorpb.DescriptorProto, msgDesc protoreflect.MessageDescriptor) error {
	// Remap types in nested any nested messages.
	for _, nestedMsg := range msgDescPb.GetNestedType() {
		nestedDesc := msgDesc.Messages().ByName(protoreflect.Name(nestedMsg.GetName()))
		if err := n.inlineRefs(nestedMsg, nestedDesc); err != nil {
			return err
		}
	}

	// Strip any custom options.
	stripExtensionsAndUnknown(msgDescPb.GetOptions())
	for _, oneOf := range msgDescPb.GetOneofDecl() {
		stripExtensionsAndUnknown(oneOf.GetOptions())
	}

	// Remap types in fields.
	syntheticOneofs := map[int32]struct{}{}
	for _, field := range msgDescPb.GetField() {
		stripExtensionsAndUnknown(field.GetOptions())
		if field.GetProto3Optional() {
			// Since we currently normalize to proto2, we need to
			// Remove the weird proto3-specific synthetic oneof for
			// explicit presence fields.
			// TODO: use editions to normalize and add relevant
			//       field presence feature to field options here
			field.Proto3Optional = nil
			syntheticOneofs[field.GetOneofIndex()] = struct{}{}
			field.OneofIndex = nil
		}
		if err := n.inlineFieldRefs(msgDescPb, msgDesc, field); err != nil {
			return err
		}
	}

	if len(syntheticOneofs) > 0 {
		n.removeOneofs(msgDescPb, syntheticOneofs)
	}

	return nil
}

func (n *Normalizer) inlineFieldRefs(
	msgDescPb *descriptorpb.DescriptorProto, msgDesc protoreflect.MessageDescriptor, field *descriptorpb.FieldDescriptorProto) error {
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		fieldDesc := msgDesc.Fields().ByName(protoreflect.Name(field.GetName()))
		if !fieldDesc.Enum().IsClosed() {
			// Convert to int32.
			field.Type = descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()
			field.TypeName = nil
			return nil
		}
		refDesc, path := findRootAndPath(fieldDesc.Enum())
		if refDesc == nil {
			if err := n.updateEnumType(fieldDesc.Enum(), field); err != nil {
				return err
			}
		} else {
			if err := n.updateMessageType(refDesc, field, path); err != nil {
				return err
			}
		}
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		fieldDesc := msgDesc.Fields().ByName(protoreflect.Name(field.GetName()))
		if fieldDesc.IsMap() {
			field.TypeName = proto.String(msgDescPb.GetName() + "." + string(fieldDesc.Message().Name()))
		} else {
			refDesc, path := findRootAndPath(fieldDesc.Message())
			if err := n.updateMessageType(refDesc, field, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Normalizer) removeOneofs(msgDescPb *descriptorpb.DescriptorProto, emptyOneofs map[int32]struct{}) {
	oneofRemap := map[int32]int32{}
	newOneofDecl := []*descriptorpb.OneofDescriptorProto{}
	for idx, oneOf := range msgDescPb.GetOneofDecl() {
		idx := int32(idx) //nolint:gosec
		if _, ok := emptyOneofs[idx]; ok {
			continue
		}
		stripExtensionsAndUnknown(oneOf.GetOptions())
		oneofRemap[idx] = int32(len(newOneofDecl)) //nolint:gosec
		newOneofDecl = append(newOneofDecl, oneOf)
	}
	for _, field := range msgDescPb.GetField() {
		if field.OneofIndex != nil {
			oneofIdx := oneofRemap[field.GetOneofIndex()]
			field.OneofIndex = &oneofIdx
		}
	}
	msgDescPb.OneofDecl = newOneofDecl
}

func (n *Normalizer) updateMessageType(refDesc protoreflect.MessageDescriptor, field *descriptorpb.FieldDescriptorProto, path []string) error {
	ref := string(refDesc.FullName())
	newRef, ok := n.nameToMangled[ref]
	if !ok {
		if err := n.inlineMessage(refDesc); err != nil {
			return err
		}
		newRef, ok = n.nameToMangled[ref]
		if !ok {
			return fmt.Errorf("could not find mangled name for %s", ref)
		}
	}
	newType := newRef
	for _, pathPart := range path {
		newType += "." + pathPart
	}
	field.TypeName = proto.String(newType)
	return nil
}

func (n *Normalizer) updateEnumType(refDesc protoreflect.EnumDescriptor, field *descriptorpb.FieldDescriptorProto) error {
	ref := string(refDesc.FullName())
	newRef, ok := n.nameToMangled[ref]
	if !ok {
		if err := n.inlineEnum(refDesc); err != nil {
			return err
		}
		newRef, ok = n.nameToMangled[ref]
		if !ok {
			return fmt.Errorf("could not find mangled name for %s", ref)
		}
	}
	field.TypeName = proto.String(newRef)
	return nil
}

func (n *Normalizer) mangleName(name string) string {
	return "Inline_" + strings.ReplaceAll(name, ".", "_")
}

func (n *Normalizer) addMangledName(name string, subName string) string {
	mangled := n.mangleName(name)
	base := mangled
	for i := 0; ; i++ {
		if _, ok := n.mangledToName[mangled]; !ok {
			break
		}
		mangled = fmt.Sprintf("%s_%d", base, i)
	}
	n.mangledToName[mangled] = name
	if subName != "" {
		n.nameToMangled[name] = fmt.Sprintf("%s.%s.%s", n.rootDesc.Name(), mangled, subName)
	} else {
		n.nameToMangled[name] = fmt.Sprintf("%s.%s", n.rootDesc.Name(), mangled)
	}
	return mangled
}

func findNestedMessage(msg *descriptorpb.DescriptorProto, name string) *descriptorpb.DescriptorProto {
	for _, nestedMsg := range msg.GetNestedType() {
		if nestedMsg.GetName() == name {
			return nestedMsg
		}
	}
	return nil
}

func findRootAndPath(target protoreflect.Descriptor) (protoreflect.MessageDescriptor, []string) {
	var path []string
	for {
		parent := target.Parent()
		parentMsg, ok := parent.(protoreflect.MessageDescriptor)
		if !ok {
			break
		}
		path = append(path, string(target.Name()))
		target = parentMsg
	}
	msg, _ := target.(protoreflect.MessageDescriptor)
	return msg, path
}

func stripExtensionsAndUnknown(options protoreflect.ProtoMessage) {
	msg := options.ProtoReflect()
	if !msg.IsValid() {
		return
	}
	msg.Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if field.IsExtension() {
			msg.Clear(field)
		}
		return true
	})
	msg.SetUnknown(nil)
}

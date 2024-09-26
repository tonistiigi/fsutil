// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.11.4
// source: github.com/tonistiigi/fsutil/types/wire.proto

package types

import (
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

type Packet_PacketType int32

const (
	Packet_PACKET_STAT Packet_PacketType = 0
	Packet_PACKET_REQ  Packet_PacketType = 1
	Packet_PACKET_DATA Packet_PacketType = 2
	Packet_PACKET_FIN  Packet_PacketType = 3
	Packet_PACKET_ERR  Packet_PacketType = 4
)

// Enum value maps for Packet_PacketType.
var (
	Packet_PacketType_name = map[int32]string{
		0: "PACKET_STAT",
		1: "PACKET_REQ",
		2: "PACKET_DATA",
		3: "PACKET_FIN",
		4: "PACKET_ERR",
	}
	Packet_PacketType_value = map[string]int32{
		"PACKET_STAT": 0,
		"PACKET_REQ":  1,
		"PACKET_DATA": 2,
		"PACKET_FIN":  3,
		"PACKET_ERR":  4,
	}
)

func (x Packet_PacketType) Enum() *Packet_PacketType {
	p := new(Packet_PacketType)
	*p = x
	return p
}

func (x Packet_PacketType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Packet_PacketType) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_tonistiigi_fsutil_types_wire_proto_enumTypes[0].Descriptor()
}

func (Packet_PacketType) Type() protoreflect.EnumType {
	return &file_github_com_tonistiigi_fsutil_types_wire_proto_enumTypes[0]
}

func (x Packet_PacketType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Packet_PacketType.Descriptor instead.
func (Packet_PacketType) EnumDescriptor() ([]byte, []int) {
	return file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescGZIP(), []int{0, 0}
}

type Packet struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type Packet_PacketType `protobuf:"varint,1,opt,name=type,proto3,enum=fsutil.types.Packet_PacketType" json:"type,omitempty"`
	Stat *Stat             `protobuf:"bytes,2,opt,name=stat,proto3" json:"stat,omitempty"`
	ID   uint32            `protobuf:"varint,3,opt,name=ID,proto3" json:"ID,omitempty"`
	Data []byte            `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Packet) Reset() {
	*x = Packet{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_tonistiigi_fsutil_types_wire_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Packet) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Packet) ProtoMessage() {}

func (x *Packet) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_tonistiigi_fsutil_types_wire_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Packet.ProtoReflect.Descriptor instead.
func (*Packet) Descriptor() ([]byte, []int) {
	return file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescGZIP(), []int{0}
}

func (x *Packet) GetType() Packet_PacketType {
	if x != nil {
		return x.Type
	}
	return Packet_PACKET_STAT
}

func (x *Packet) GetStat() *Stat {
	if x != nil {
		return x.Stat
	}
	return nil
}

func (x *Packet) GetID() uint32 {
	if x != nil {
		return x.ID
	}
	return 0
}

func (x *Packet) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_github_com_tonistiigi_fsutil_types_wire_proto protoreflect.FileDescriptor

var file_github_com_tonistiigi_fsutil_types_wire_proto_rawDesc = []byte{
	0x0a, 0x2d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x6f, 0x6e,
	0x69, 0x73, 0x74, 0x69, 0x69, 0x67, 0x69, 0x2f, 0x66, 0x73, 0x75, 0x74, 0x69, 0x6c, 0x2f, 0x74,
	0x79, 0x70, 0x65, 0x73, 0x2f, 0x77, 0x69, 0x72, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0c, 0x66, 0x73, 0x75, 0x74, 0x69, 0x6c, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x1a, 0x2d, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x6f, 0x6e, 0x69, 0x73, 0x74,
	0x69, 0x69, 0x67, 0x69, 0x2f, 0x66, 0x73, 0x75, 0x74, 0x69, 0x6c, 0x2f, 0x74, 0x79, 0x70, 0x65,
	0x73, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xe9, 0x01, 0x0a,
	0x06, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x12, 0x33, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1f, 0x2e, 0x66, 0x73, 0x75, 0x74, 0x69, 0x6c, 0x2e, 0x74,
	0x79, 0x70, 0x65, 0x73, 0x2e, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x2e, 0x50, 0x61, 0x63, 0x6b,
	0x65, 0x74, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x26, 0x0a, 0x04,
	0x73, 0x74, 0x61, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x66, 0x73, 0x75,
	0x74, 0x69, 0x6c, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x52, 0x04,
	0x73, 0x74, 0x61, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x02, 0x49, 0x44, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x5e, 0x0a, 0x0a, 0x50, 0x61, 0x63, 0x6b,
	0x65, 0x74, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0f, 0x0a, 0x0b, 0x50, 0x41, 0x43, 0x4b, 0x45, 0x54,
	0x5f, 0x53, 0x54, 0x41, 0x54, 0x10, 0x00, 0x12, 0x0e, 0x0a, 0x0a, 0x50, 0x41, 0x43, 0x4b, 0x45,
	0x54, 0x5f, 0x52, 0x45, 0x51, 0x10, 0x01, 0x12, 0x0f, 0x0a, 0x0b, 0x50, 0x41, 0x43, 0x4b, 0x45,
	0x54, 0x5f, 0x44, 0x41, 0x54, 0x41, 0x10, 0x02, 0x12, 0x0e, 0x0a, 0x0a, 0x50, 0x41, 0x43, 0x4b,
	0x45, 0x54, 0x5f, 0x46, 0x49, 0x4e, 0x10, 0x03, 0x12, 0x0e, 0x0a, 0x0a, 0x50, 0x41, 0x43, 0x4b,
	0x45, 0x54, 0x5f, 0x45, 0x52, 0x52, 0x10, 0x04, 0x42, 0x24, 0x5a, 0x22, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x6f, 0x6e, 0x69, 0x73, 0x74, 0x69, 0x69, 0x67,
	0x69, 0x2f, 0x66, 0x73, 0x75, 0x74, 0x69, 0x6c, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescOnce sync.Once
	file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescData = file_github_com_tonistiigi_fsutil_types_wire_proto_rawDesc
)

func file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescGZIP() []byte {
	file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescOnce.Do(func() {
		file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescData)
	})
	return file_github_com_tonistiigi_fsutil_types_wire_proto_rawDescData
}

var file_github_com_tonistiigi_fsutil_types_wire_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_tonistiigi_fsutil_types_wire_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_tonistiigi_fsutil_types_wire_proto_goTypes = []interface{}{
	(Packet_PacketType)(0), // 0: fsutil.types.Packet.PacketType
	(*Packet)(nil),         // 1: fsutil.types.Packet
	(*Stat)(nil),           // 2: fsutil.types.Stat
}
var file_github_com_tonistiigi_fsutil_types_wire_proto_depIdxs = []int32{
	0, // 0: fsutil.types.Packet.type:type_name -> fsutil.types.Packet.PacketType
	2, // 1: fsutil.types.Packet.stat:type_name -> fsutil.types.Stat
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_github_com_tonistiigi_fsutil_types_wire_proto_init() }
func file_github_com_tonistiigi_fsutil_types_wire_proto_init() {
	if File_github_com_tonistiigi_fsutil_types_wire_proto != nil {
		return
	}
	file_github_com_tonistiigi_fsutil_types_stat_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_github_com_tonistiigi_fsutil_types_wire_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Packet); i {
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
			RawDescriptor: file_github_com_tonistiigi_fsutil_types_wire_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_tonistiigi_fsutil_types_wire_proto_goTypes,
		DependencyIndexes: file_github_com_tonistiigi_fsutil_types_wire_proto_depIdxs,
		EnumInfos:         file_github_com_tonistiigi_fsutil_types_wire_proto_enumTypes,
		MessageInfos:      file_github_com_tonistiigi_fsutil_types_wire_proto_msgTypes,
	}.Build()
	File_github_com_tonistiigi_fsutil_types_wire_proto = out.File
	file_github_com_tonistiigi_fsutil_types_wire_proto_rawDesc = nil
	file_github_com_tonistiigi_fsutil_types_wire_proto_goTypes = nil
	file_github_com_tonistiigi_fsutil_types_wire_proto_depIdxs = nil
}

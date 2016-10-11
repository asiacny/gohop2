// Code generated by protoc-gen-go.
// source: fin.proto
// DO NOT EDIT!

package protodef

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Fin struct {
	Header *PacketHeader `protobuf:"bytes,1,opt,name=header" json:"header,omitempty"`
}

func (m *Fin) Reset()                    { *m = Fin{} }
func (m *Fin) String() string            { return proto.CompactTextString(m) }
func (*Fin) ProtoMessage()               {}
func (*Fin) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *Fin) GetHeader() *PacketHeader {
	if m != nil {
		return m.Header
	}
	return nil
}

type FinAck struct {
	Header *PacketHeader `protobuf:"bytes,1,opt,name=header" json:"header,omitempty"`
}

func (m *FinAck) Reset()                    { *m = FinAck{} }
func (m *FinAck) String() string            { return proto.CompactTextString(m) }
func (*FinAck) ProtoMessage()               {}
func (*FinAck) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (m *FinAck) GetHeader() *PacketHeader {
	if m != nil {
		return m.Header
	}
	return nil
}

func init() {
	proto.RegisterType((*Fin)(nil), "protodef.Fin")
	proto.RegisterType((*FinAck)(nil), "protodef.FinAck")
}

func init() { proto.RegisterFile("fin.proto", fileDescriptor1) }

var fileDescriptor1 = []byte{
	// 106 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0x4c, 0xcb, 0xcc, 0xd3,
	0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x00, 0x53, 0x29, 0xa9, 0x69, 0x52, 0x3c, 0x19, 0xa9,
	0x89, 0x29, 0xa9, 0x45, 0x10, 0x71, 0x25, 0x53, 0x2e, 0x66, 0xb7, 0xcc, 0x3c, 0x21, 0x3d, 0x2e,
	0x36, 0x88, 0xb0, 0x04, 0xa3, 0x02, 0xa3, 0x06, 0xb7, 0x91, 0x98, 0x1e, 0x4c, 0xbd, 0x5e, 0x40,
	0x62, 0x72, 0x76, 0x6a, 0x89, 0x07, 0x58, 0x36, 0x08, 0xaa, 0x4a, 0xc9, 0x82, 0x8b, 0xcd, 0x2d,
	0x33, 0xcf, 0x31, 0x39, 0x9b, 0x54, 0x9d, 0x49, 0x6c, 0x60, 0x69, 0x63, 0x40, 0x00, 0x00, 0x00,
	0xff, 0xff, 0x61, 0xde, 0xf6, 0x81, 0x9c, 0x00, 0x00, 0x00,
}
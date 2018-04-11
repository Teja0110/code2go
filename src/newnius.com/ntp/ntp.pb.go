// Code generated by protoc-gen-go. DO NOT EDIT.
// source: ntp.proto

/*
Package ntp is a generated protocol buffer package.

It is generated from these files:
	ntp.proto

It has these top-level messages:
	NtpRequest
	NtpReply
*/
package ntp

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type NtpRequest struct {
}

func (m *NtpRequest) Reset()                    { *m = NtpRequest{} }
func (m *NtpRequest) String() string            { return proto.CompactTextString(m) }
func (*NtpRequest) ProtoMessage()               {}
func (*NtpRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type NtpReply struct {
	// string message = 1;
	Message int64 `protobuf:"varint,1,opt,name=message" json:"message,omitempty"`
}

func (m *NtpReply) Reset()                    { *m = NtpReply{} }
func (m *NtpReply) String() string            { return proto.CompactTextString(m) }
func (*NtpReply) ProtoMessage()               {}
func (*NtpReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *NtpReply) GetMessage() int64 {
	if m != nil {
		return m.Message
	}
	return 0
}

func init() {
	proto.RegisterType((*NtpRequest)(nil), "ntp.NtpRequest")
	proto.RegisterType((*NtpReply)(nil), "ntp.NtpReply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Ntp service

type NtpClient interface {
	Query(ctx context.Context, in *NtpRequest, opts ...grpc.CallOption) (*NtpReply, error)
}

type ntpClient struct {
	cc *grpc.ClientConn
}

func NewNtpClient(cc *grpc.ClientConn) NtpClient {
	return &ntpClient{cc}
}

func (c *ntpClient) Query(ctx context.Context, in *NtpRequest, opts ...grpc.CallOption) (*NtpReply, error) {
	out := new(NtpReply)
	err := grpc.Invoke(ctx, "/ntp.Ntp/Query", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Ntp service

type NtpServer interface {
	Query(context.Context, *NtpRequest) (*NtpReply, error)
}

func RegisterNtpServer(s *grpc.Server, srv NtpServer) {
	s.RegisterService(&_Ntp_serviceDesc, srv)
}

func _Ntp_Query_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NtpRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NtpServer).Query(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ntp.Ntp/Query",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NtpServer).Query(ctx, req.(*NtpRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Ntp_serviceDesc = grpc.ServiceDesc{
	ServiceName: "ntp.Ntp",
	HandlerType: (*NtpServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Query",
			Handler:    _Ntp_Query_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ntp.proto",
}

func init() { proto.RegisterFile("ntp.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 142 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xcc, 0x2b, 0x29, 0xd0,
	0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0xce, 0x2b, 0x29, 0x50, 0xe2, 0xe1, 0xe2, 0xf2, 0x2b,
	0x29, 0x08, 0x4a, 0x2d, 0x2c, 0x4d, 0x2d, 0x2e, 0x51, 0x52, 0xe1, 0xe2, 0x00, 0xf3, 0x0a, 0x72,
	0x2a, 0x85, 0x24, 0xb8, 0xd8, 0x73, 0x53, 0x8b, 0x8b, 0x13, 0xd3, 0x53, 0x25, 0x18, 0x15, 0x18,
	0x35, 0x98, 0x83, 0x60, 0x5c, 0x23, 0x03, 0x2e, 0x66, 0xbf, 0x92, 0x02, 0x21, 0x4d, 0x2e, 0xd6,
	0xc0, 0xd2, 0xd4, 0xa2, 0x4a, 0x21, 0x7e, 0x3d, 0x90, 0xa1, 0x08, 0x63, 0xa4, 0x78, 0x11, 0x02,
	0x05, 0x39, 0x95, 0x4a, 0x0c, 0x4e, 0x22, 0x5c, 0x42, 0xc9, 0xf9, 0xb9, 0x7a, 0xc9, 0xf9, 0x39,
	0xf9, 0x49, 0xa5, 0x7a, 0x45, 0x05, 0xc9, 0x25, 0xa9, 0xc5, 0x25, 0x49, 0x6c, 0x60, 0x77, 0x18,
	0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x7d, 0x39, 0x97, 0x07, 0x94, 0x00, 0x00, 0x00,
}

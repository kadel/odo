// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v0/resources/customer_client.proto

package resources // import "google.golang.org/genproto/googleapis/ads/googleads/v0/resources"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import wrappers "github.com/golang/protobuf/ptypes/wrappers"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// A link between the given customer and a client customer. CustomerClients only
// exist for manager customers. All direct and indirect client customers are
// included, as well as the manager itself.
type CustomerClient struct {
	// The resource name of the customer client.
	// CustomerClient resource names have the form:
	// `customers/{customer_id}/customerClients/{client_customer_id}`
	ResourceName string `protobuf:"bytes,1,opt,name=resource_name,json=resourceName,proto3" json:"resource_name,omitempty"`
	// The resource name of the client-customer which is linked to
	// the given customer. Read only.
	ClientCustomer *wrappers.StringValue `protobuf:"bytes,3,opt,name=client_customer,json=clientCustomer,proto3" json:"client_customer,omitempty"`
	// Specifies whether this is a hidden account. Learn more about hidden
	// accounts
	// <a href="https://support.google.com/google-ads/answer/7519830">here</a>.
	// Read only.
	Hidden *wrappers.BoolValue `protobuf:"bytes,4,opt,name=hidden,proto3" json:"hidden,omitempty"`
	// Distance between given customer and client. For self link, the level value
	// will be 0. Read only.
	Level                *wrappers.Int64Value `protobuf:"bytes,5,opt,name=level,proto3" json:"level,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *CustomerClient) Reset()         { *m = CustomerClient{} }
func (m *CustomerClient) String() string { return proto.CompactTextString(m) }
func (*CustomerClient) ProtoMessage()    {}
func (*CustomerClient) Descriptor() ([]byte, []int) {
	return fileDescriptor_customer_client_b474fc80928b30c9, []int{0}
}
func (m *CustomerClient) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CustomerClient.Unmarshal(m, b)
}
func (m *CustomerClient) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CustomerClient.Marshal(b, m, deterministic)
}
func (dst *CustomerClient) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CustomerClient.Merge(dst, src)
}
func (m *CustomerClient) XXX_Size() int {
	return xxx_messageInfo_CustomerClient.Size(m)
}
func (m *CustomerClient) XXX_DiscardUnknown() {
	xxx_messageInfo_CustomerClient.DiscardUnknown(m)
}

var xxx_messageInfo_CustomerClient proto.InternalMessageInfo

func (m *CustomerClient) GetResourceName() string {
	if m != nil {
		return m.ResourceName
	}
	return ""
}

func (m *CustomerClient) GetClientCustomer() *wrappers.StringValue {
	if m != nil {
		return m.ClientCustomer
	}
	return nil
}

func (m *CustomerClient) GetHidden() *wrappers.BoolValue {
	if m != nil {
		return m.Hidden
	}
	return nil
}

func (m *CustomerClient) GetLevel() *wrappers.Int64Value {
	if m != nil {
		return m.Level
	}
	return nil
}

func init() {
	proto.RegisterType((*CustomerClient)(nil), "google.ads.googleads.v0.resources.CustomerClient")
}

func init() {
	proto.RegisterFile("google/ads/googleads/v0/resources/customer_client.proto", fileDescriptor_customer_client_b474fc80928b30c9)
}

var fileDescriptor_customer_client_b474fc80928b30c9 = []byte{
	// 351 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x91, 0x41, 0x4b, 0xf3, 0x30,
	0x18, 0xc7, 0x69, 0xf7, 0x6e, 0xf0, 0x56, 0x9d, 0x50, 0x2f, 0x65, 0x8a, 0x6c, 0xca, 0x60, 0xa7,
	0xb4, 0x4e, 0x51, 0x88, 0xa7, 0x6e, 0xc8, 0xd0, 0x83, 0x8c, 0x09, 0x3d, 0x48, 0x61, 0x74, 0x4d,
	0x8c, 0x85, 0x34, 0x29, 0x49, 0x3b, 0xaf, 0x7e, 0x16, 0x8f, 0x7e, 0x14, 0xbf, 0x86, 0x37, 0x3f,
	0x85, 0xac, 0x69, 0x02, 0x32, 0xd0, 0xdb, 0x9f, 0xf6, 0xf7, 0xfb, 0x3f, 0x0f, 0x79, 0x9c, 0x2b,
	0xc2, 0x39, 0xa1, 0xd8, 0x4f, 0x90, 0xf4, 0x55, 0xdc, 0xa4, 0x75, 0xe0, 0x0b, 0x2c, 0x79, 0x25,
	0x52, 0x2c, 0xfd, 0xb4, 0x92, 0x25, 0xcf, 0xb1, 0x58, 0xa6, 0x34, 0xc3, 0xac, 0x04, 0x85, 0xe0,
	0x25, 0x77, 0x07, 0x8a, 0x06, 0x09, 0x92, 0xc0, 0x88, 0x60, 0x1d, 0x00, 0x23, 0xf6, 0x8e, 0x9b,
	0xee, 0x5a, 0x58, 0x55, 0x4f, 0xfe, 0x8b, 0x48, 0x8a, 0x02, 0x0b, 0xa9, 0x2a, 0x4e, 0x3e, 0x2d,
	0xa7, 0x3b, 0x6d, 0xca, 0xa7, 0x75, 0xb7, 0x7b, 0xea, 0xec, 0x69, 0x7f, 0xc9, 0x92, 0x1c, 0x7b,
	0x56, 0xdf, 0x1a, 0xfd, 0x5f, 0xec, 0xea, 0x8f, 0xf7, 0x49, 0x8e, 0xdd, 0x1b, 0x67, 0x5f, 0xad,
	0xb2, 0xd4, 0xab, 0x79, 0xad, 0xbe, 0x35, 0xda, 0x19, 0x1f, 0x35, 0x9b, 0x00, 0x3d, 0x11, 0x3c,
	0x94, 0x22, 0x63, 0x24, 0x4a, 0x68, 0x85, 0x17, 0x5d, 0x25, 0xe9, 0x89, 0xee, 0xd8, 0xe9, 0x3c,
	0x67, 0x08, 0x61, 0xe6, 0xfd, 0xab, 0xed, 0xde, 0x96, 0x3d, 0xe1, 0x9c, 0x2a, 0xb7, 0x21, 0xdd,
	0x33, 0xa7, 0x4d, 0xf1, 0x1a, 0x53, 0xaf, 0x5d, 0x2b, 0x87, 0x5b, 0xca, 0x2d, 0x2b, 0x2f, 0x2f,
	0x94, 0xa3, 0xc8, 0xc9, 0xab, 0xed, 0x0c, 0x53, 0x9e, 0x83, 0x3f, 0xdf, 0x6b, 0x72, 0xf0, 0xf3,
	0x31, 0xe6, 0x9b, 0xce, 0xb9, 0xf5, 0x78, 0xd7, 0x98, 0x84, 0xd3, 0x84, 0x11, 0xc0, 0x05, 0xf1,
	0x09, 0x66, 0xf5, 0x44, 0x7d, 0xb2, 0x22, 0x93, 0xbf, 0x5c, 0xf0, 0xda, 0xa4, 0x37, 0xbb, 0x35,
	0x0b, 0xc3, 0x77, 0x7b, 0x30, 0x53, 0x95, 0x21, 0x92, 0x40, 0xc5, 0x4d, 0x8a, 0x02, 0xb0, 0xd0,
	0xe4, 0x87, 0x66, 0xe2, 0x10, 0xc9, 0xd8, 0x30, 0x71, 0x14, 0xc4, 0x86, 0xf9, 0xb2, 0x87, 0xea,
	0x07, 0x84, 0x21, 0x92, 0x10, 0x1a, 0x0a, 0xc2, 0x28, 0x80, 0xd0, 0x70, 0xab, 0x4e, 0xbd, 0xec,
	0xf9, 0x77, 0x00, 0x00, 0x00, 0xff, 0xff, 0x16, 0x7a, 0xb0, 0x1a, 0x6d, 0x02, 0x00, 0x00,
}

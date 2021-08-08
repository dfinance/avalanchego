// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: dfinance/dvm/compiler.proto

package dvm

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

// Compilation unit.
type CompilationUnit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Text string `protobuf:"bytes,1,opt,name=text,proto3" json:"text,omitempty"` // utf8 encoded source code with libra/bech32 addresses
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"` // name of the unit.
}

func (x *CompilationUnit) Reset() {
	*x = CompilationUnit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dfinance_dvm_compiler_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CompilationUnit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CompilationUnit) ProtoMessage() {}

func (x *CompilationUnit) ProtoReflect() protoreflect.Message {
	mi := &file_dfinance_dvm_compiler_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CompilationUnit.ProtoReflect.Descriptor instead.
func (*CompilationUnit) Descriptor() ([]byte, []int) {
	return file_dfinance_dvm_compiler_proto_rawDescGZIP(), []int{0}
}

func (x *CompilationUnit) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

func (x *CompilationUnit) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

// Compiler API
type SourceFiles struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Units   []*CompilationUnit `protobuf:"bytes,1,rep,name=units,proto3" json:"units,omitempty"`     // Compilation units.
	Address []byte             `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"` // address of the sender, in bech32 form
}

func (x *SourceFiles) Reset() {
	*x = SourceFiles{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dfinance_dvm_compiler_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SourceFiles) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SourceFiles) ProtoMessage() {}

func (x *SourceFiles) ProtoReflect() protoreflect.Message {
	mi := &file_dfinance_dvm_compiler_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SourceFiles.ProtoReflect.Descriptor instead.
func (*SourceFiles) Descriptor() ([]byte, []int) {
	return file_dfinance_dvm_compiler_proto_rawDescGZIP(), []int{1}
}

func (x *SourceFiles) GetUnits() []*CompilationUnit {
	if x != nil {
		return x.Units
	}
	return nil
}

func (x *SourceFiles) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

// Compiled source.
type CompiledUnit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name     string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`         // name of the module/script.
	Bytecode []byte `protobuf:"bytes,2,opt,name=bytecode,proto3" json:"bytecode,omitempty"` // bytecode of the compiled module/script
}

func (x *CompiledUnit) Reset() {
	*x = CompiledUnit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dfinance_dvm_compiler_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CompiledUnit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CompiledUnit) ProtoMessage() {}

func (x *CompiledUnit) ProtoReflect() protoreflect.Message {
	mi := &file_dfinance_dvm_compiler_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CompiledUnit.ProtoReflect.Descriptor instead.
func (*CompiledUnit) Descriptor() ([]byte, []int) {
	return file_dfinance_dvm_compiler_proto_rawDescGZIP(), []int{2}
}

func (x *CompiledUnit) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *CompiledUnit) GetBytecode() []byte {
	if x != nil {
		return x.Bytecode
	}
	return nil
}

type CompilationResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Units  []*CompiledUnit `protobuf:"bytes,1,rep,name=units,proto3" json:"units,omitempty"`
	Errors []string        `protobuf:"bytes,2,rep,name=errors,proto3" json:"errors,omitempty"` // list of error messages, empty if successful
}

func (x *CompilationResult) Reset() {
	*x = CompilationResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dfinance_dvm_compiler_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CompilationResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CompilationResult) ProtoMessage() {}

func (x *CompilationResult) ProtoReflect() protoreflect.Message {
	mi := &file_dfinance_dvm_compiler_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CompilationResult.ProtoReflect.Descriptor instead.
func (*CompilationResult) Descriptor() ([]byte, []int) {
	return file_dfinance_dvm_compiler_proto_rawDescGZIP(), []int{3}
}

func (x *CompilationResult) GetUnits() []*CompiledUnit {
	if x != nil {
		return x.Units
	}
	return nil
}

func (x *CompilationResult) GetErrors() []string {
	if x != nil {
		return x.Errors
	}
	return nil
}

var File_dfinance_dvm_compiler_proto protoreflect.FileDescriptor

var file_dfinance_dvm_compiler_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x64, 0x66, 0x69, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2f, 0x64, 0x76, 0x6d, 0x2f, 0x63,
	0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c, 0x64,
	0x66, 0x69, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x64, 0x76, 0x6d, 0x22, 0x39, 0x0a, 0x0f, 0x43,
	0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x55, 0x6e, 0x69, 0x74, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x65, 0x78, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65,
	0x78, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x5c, 0x0a, 0x0b, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x46, 0x69, 0x6c, 0x65, 0x73, 0x12, 0x33, 0x0a, 0x05, 0x75, 0x6e, 0x69, 0x74, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x64, 0x66, 0x69, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e,
	0x64, 0x76, 0x6d, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x55,
	0x6e, 0x69, 0x74, 0x52, 0x05, 0x75, 0x6e, 0x69, 0x74, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64,
	0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x22, 0x3e, 0x0a, 0x0c, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x64,
	0x55, 0x6e, 0x69, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x62, 0x79, 0x74, 0x65,
	0x63, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x08, 0x62, 0x79, 0x74, 0x65,
	0x63, 0x6f, 0x64, 0x65, 0x22, 0x5d, 0x0a, 0x11, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x30, 0x0a, 0x05, 0x75, 0x6e, 0x69,
	0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x64, 0x66, 0x69, 0x6e, 0x61,
	0x6e, 0x63, 0x65, 0x2e, 0x64, 0x76, 0x6d, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x64,
	0x55, 0x6e, 0x69, 0x74, 0x52, 0x05, 0x75, 0x6e, 0x69, 0x74, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x65, 0x72, 0x72,
	0x6f, 0x72, 0x73, 0x32, 0x56, 0x0a, 0x0b, 0x44, 0x76, 0x6d, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c,
	0x65, 0x72, 0x12, 0x47, 0x0a, 0x07, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x12, 0x19, 0x2e,
	0x64, 0x66, 0x69, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x2e, 0x64, 0x76, 0x6d, 0x2e, 0x53, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x73, 0x1a, 0x1f, 0x2e, 0x64, 0x66, 0x69, 0x6e, 0x61,
	0x6e, 0x63, 0x65, 0x2e, 0x64, 0x76, 0x6d, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x22, 0x00, 0x42, 0x0a, 0x5a, 0x08, 0x6d,
	0x76, 0x6d, 0x2f, 0x64, 0x76, 0x6d, 0x2f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_dfinance_dvm_compiler_proto_rawDescOnce sync.Once
	file_dfinance_dvm_compiler_proto_rawDescData = file_dfinance_dvm_compiler_proto_rawDesc
)

func file_dfinance_dvm_compiler_proto_rawDescGZIP() []byte {
	file_dfinance_dvm_compiler_proto_rawDescOnce.Do(func() {
		file_dfinance_dvm_compiler_proto_rawDescData = protoimpl.X.CompressGZIP(file_dfinance_dvm_compiler_proto_rawDescData)
	})
	return file_dfinance_dvm_compiler_proto_rawDescData
}

var file_dfinance_dvm_compiler_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_dfinance_dvm_compiler_proto_goTypes = []interface{}{
	(*CompilationUnit)(nil),   // 0: dfinance.dvm.CompilationUnit
	(*SourceFiles)(nil),       // 1: dfinance.dvm.SourceFiles
	(*CompiledUnit)(nil),      // 2: dfinance.dvm.CompiledUnit
	(*CompilationResult)(nil), // 3: dfinance.dvm.CompilationResult
}
var file_dfinance_dvm_compiler_proto_depIdxs = []int32{
	0, // 0: dfinance.dvm.SourceFiles.units:type_name -> dfinance.dvm.CompilationUnit
	2, // 1: dfinance.dvm.CompilationResult.units:type_name -> dfinance.dvm.CompiledUnit
	1, // 2: dfinance.dvm.DvmCompiler.Compile:input_type -> dfinance.dvm.SourceFiles
	3, // 3: dfinance.dvm.DvmCompiler.Compile:output_type -> dfinance.dvm.CompilationResult
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_dfinance_dvm_compiler_proto_init() }
func file_dfinance_dvm_compiler_proto_init() {
	if File_dfinance_dvm_compiler_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_dfinance_dvm_compiler_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CompilationUnit); i {
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
		file_dfinance_dvm_compiler_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SourceFiles); i {
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
		file_dfinance_dvm_compiler_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CompiledUnit); i {
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
		file_dfinance_dvm_compiler_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CompilationResult); i {
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
			RawDescriptor: file_dfinance_dvm_compiler_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_dfinance_dvm_compiler_proto_goTypes,
		DependencyIndexes: file_dfinance_dvm_compiler_proto_depIdxs,
		MessageInfos:      file_dfinance_dvm_compiler_proto_msgTypes,
	}.Build()
	File_dfinance_dvm_compiler_proto = out.File
	file_dfinance_dvm_compiler_proto_rawDesc = nil
	file_dfinance_dvm_compiler_proto_goTypes = nil
	file_dfinance_dvm_compiler_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// DvmCompilerClient is the client API for DvmCompiler service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type DvmCompilerClient interface {
	Compile(ctx context.Context, in *SourceFiles, opts ...grpc.CallOption) (*CompilationResult, error)
}

type dvmCompilerClient struct {
	cc grpc.ClientConnInterface
}

func NewDvmCompilerClient(cc grpc.ClientConnInterface) DvmCompilerClient {
	return &dvmCompilerClient{cc}
}

func (c *dvmCompilerClient) Compile(ctx context.Context, in *SourceFiles, opts ...grpc.CallOption) (*CompilationResult, error) {
	out := new(CompilationResult)
	err := c.cc.Invoke(ctx, "/dfinance.dvm.DvmCompiler/Compile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DvmCompilerServer is the server API for DvmCompiler service.
type DvmCompilerServer interface {
	Compile(context.Context, *SourceFiles) (*CompilationResult, error)
}

// UnimplementedDvmCompilerServer can be embedded to have forward compatible implementations.
type UnimplementedDvmCompilerServer struct {
}

func (*UnimplementedDvmCompilerServer) Compile(context.Context, *SourceFiles) (*CompilationResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Compile not implemented")
}

func RegisterDvmCompilerServer(s *grpc.Server, srv DvmCompilerServer) {
	s.RegisterService(&_DvmCompiler_serviceDesc, srv)
}

func _DvmCompiler_Compile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SourceFiles)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DvmCompilerServer).Compile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/dfinance.dvm.DvmCompiler/Compile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DvmCompilerServer).Compile(ctx, req.(*SourceFiles))
	}
	return interceptor(ctx, in, info, handler)
}

var _DvmCompiler_serviceDesc = grpc.ServiceDesc{
	ServiceName: "dfinance.dvm.DvmCompiler",
	HandlerType: (*DvmCompilerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Compile",
			Handler:    _DvmCompiler_Compile_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "dfinance/dvm/compiler.proto",
}

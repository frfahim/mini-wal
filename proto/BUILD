load("@rules_proto//proto:defs.bzl", "proto_library")
load("@rules_go//proto:def.bzl", "go_proto_library")
load("@rules_go//go:def.bzl", "go_library")


proto_library(
    name = "wal_proto",
    srcs = ["wal.proto"],
    visibility = ["//visibility:public"],
)

# Build the Go proto library:
go_proto_library(
    name = "wal_go_proto",
    importpath = "wal/proto",
    proto = ":wal_proto",
    visibility = ["//visibility:public"],
)

# Build the full Go library:
go_library(
    name = "wal",
	srcs = ["wal.pb.go"],
    importpath = "wal/proto",
    visibility = ["//visibility:public"],
	deps = [
        "@org_golang_google_protobuf//reflect/protoreflect",
        "@org_golang_google_protobuf//runtime/protoimpl",
    ],
)

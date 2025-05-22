load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "wal_lib",
    srcs = ["wal.go", "types.go"],
    importpath = "wal/internal",
    visibility = ["//:__subpackages__"],
    deps = [
        "//proto:wal",
        "@org_golang_google_protobuf//proto"
    ]
)

go_test(
    name = "wal_test",
    srcs = ["wal_test.go"],
    embed = [":wal_lib"],
)

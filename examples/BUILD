load("@rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "test_wal",
    srcs = ["main.go"],
    deps = [
        "//internal:wal_lib"
    ],
    visibility = ["//visibility:public"]
)

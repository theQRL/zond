load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "a8d6b1b354d371a646d2f7927319974e0f9e52f73a2452d2b3877118169eb6bb",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.3/rules_go-v0.23.3.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.3/rules_go-v0.23.3.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@io_bazel_rules_go//extras:embed_data_deps.bzl", "go_embed_data_dependencies")

go_embed_data_dependencies()

http_archive(
    name = "bazel_gazelle",
    sha256 = "cdb02a887a7187ea4d5a27452311a75ed8637379a1287d8eeb952138ea485f7d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
    ],
)

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "com_google_protobuf",
    commit = "09745575a923640154bcf307fba8aedff47f240a",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1558721209 -0700",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable",
    importpath = "google.golang.org/grpc",
    sum = "h1:SfXqXS5hkufcdZ/mHtYCh53P2b+92WQq/DZcKLgsFRs=",
    version = "v1.31.1",
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    sum = "h1:vGXIOMxbNfDTk/aXCmfdLgkrSV+Z2tcbze+pEc3v5W4=",
    version = "v0.0.0-20200625001655-4c5254603344",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    sum = "h1:tW2bmiBqwgJj/UpqtC8EpXEZVYOwU0yG4iWbprSVAcs=",
    version = "v0.3.2",
)

load("//:deps.bzl", "zond_deps")

zond_deps()

go_repository(
    name = "co_honnef_go_tools",
    importpath = "honnef.co/go/tools",
    sum = "h1:3JgtbtFHMiCmsznwGVTUWbgGov+pVqnlf1dEJTNAXeM=",
    version = "v0.0.1-2019.2.3",
)

go_repository(
    name = "com_github_beevik_ntp",
    importpath = "github.com/beevik/ntp",
    sum = "h1:xzVrPrE4ziasFXgBVBZJDP0Wg/KpMwk2KHJ4Ba8GrDw=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
    version = "v0.3.1",
)

go_repository(
    name = "com_github_census_instrumentation_opencensus_proto",
    importpath = "github.com/census-instrumentation/opencensus-proto",
    sum = "h1:glEXhBS5PSLLv4IXzLA5yPRVX4bilULVyxxbrfOtDAk=",
    version = "v0.2.1",
)

go_repository(
    name = "com_github_client9_misspell",
    importpath = "github.com/client9/misspell",
    sum = "h1:ta993UF76GwbvJcIo3Y68y/M3WxlpEHPWIGDkJYwzJI=",
    version = "v0.3.4",
)

go_repository(
    name = "com_github_cpuguy83_go_md2man_v2",
    importpath = "github.com/cpuguy83/go-md2man/v2",
    sum = "h1:U+s90UTSYgptZMwQh2aRr3LuazLJIa+Pg3Kc1ylSYVY=",
    version = "v2.0.0-20190314233015-f79a8a8ca69d",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    importpath = "github.com/davecgh/go-spew",
    sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_deckarep_golang_set",
    importpath = "github.com/deckarep/golang-set",
    sum = "h1:SCQV0S6gTtp6itiFrTqI+pfmJ4LN85S1YzhDf9rTHJQ=",
    version = "v1.7.1",
)

go_repository(
    name = "com_github_envoyproxy_go_control_plane",
    importpath = "github.com/envoyproxy/go-control-plane",
    sum = "h1:rEvIZUSZ3fx39WIi3JkQqQBitGwpELBIYWeBVh6wn+E=",
    version = "v0.9.4",
)

go_repository(
    name = "com_github_envoyproxy_protoc_gen_validate",
    importpath = "github.com/envoyproxy/protoc-gen-validate",
    sum = "h1:EQciDnbrYxy13PgWoY8AqoxGiPrpgBZ1R8UNe3ddc+A=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_ghodss_yaml",
    importpath = "github.com/ghodss/yaml",
    sum = "h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_golang_glog",
    importpath = "github.com/golang/glog",
    sum = "h1:VKtxabqXZkF25pY9ekfRL6a582T4P37/31XEstQ5p58=",
    version = "v0.0.0-20160126235308-23def4e6c14b",
)

go_repository(
    name = "com_github_golang_mock",
    importpath = "github.com/golang/mock",
    sum = "h1:G5FRp8JnTd7RQH5kemVNlMeyXQAztQ3mOWV95KxsXH8=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_golang_protobuf",
    importpath = "github.com/golang/protobuf",
    sum = "h1:+Z5KGCizgyZCbGh1KZqA0fcLLkwbsjIzS4aV2v7wJX0=",
    version = "v1.4.2",
)

go_repository(
    name = "com_github_google_go_cmp",
    importpath = "github.com/google/go-cmp",
    sum = "h1:/QaMHBdZ26BB3SSst0Iwl10Epc+xhTquomWX0oZEB6w=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_gorilla_mux",
    importpath = "github.com/gorilla/mux",
    sum = "h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_konsorten_go_windows_terminal_sequences",
    importpath = "github.com/konsorten/go-windows-terminal-sequences",
    sum = "h1:CE8S1cTafDpPvMhIxNJKvHsGVBgn1xWYf1NbHQhywc8=",
    version = "v1.0.3",
)

go_repository(
    name = "com_github_pmezard_go_difflib",
    importpath = "github.com/pmezard/go-difflib",
    sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_prometheus_client_model",
    importpath = "github.com/prometheus/client_model",
    sum = "h1:gQz4mCbXsO+nc9n1hCxHcGA3Zx3Eo+UHZoInFGUIXNM=",
    version = "v0.0.0-20190812154241-14fe0d1b01d4",
)

go_repository(
    name = "com_github_rs_cors",
    importpath = "github.com/rs/cors",
    sum = "h1:+88SsELBHx5r+hZ8TCkggzSstaWNbDvThkVK8H6f9ik=",
    version = "v1.7.0",
)

go_repository(
    name = "com_github_russross_blackfriday_v2",
    importpath = "github.com/russross/blackfriday/v2",
    sum = "h1:lPqVAte+HuHNfhJ/0LC98ESWRz8afy9tM/0RK8m9o+Q=",
    version = "v2.0.1",
)

go_repository(
    name = "com_github_shurcool_sanitized_anchor_name",
    importpath = "github.com/shurcooL/sanitized_anchor_name",
    sum = "h1:PdmoCO6wvbs+7yrJyMORt4/BmY5IYyJwS/kOiWx8mHo=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    importpath = "github.com/sirupsen/logrus",
    sum = "h1:UBcNElsrwanuuMsnGSlYmtmgbb23qDR5dG+6X6Oo89I=",
    version = "v1.6.0",
)

go_repository(
    name = "com_github_spaolacci_murmur3",
    importpath = "github.com/spaolacci/murmur3",
    sum = "h1:7c1g84S4BPRrfL5Xrdp6fOJ206sU9y293DDHaoy0bLI=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_stretchr_testify",
    importpath = "github.com/stretchr/testify",
    sum = "h1:hDPOHmpOpP40lSULcqw7IrRb/u7w6RpDC9399XyoNd0=",
    version = "v1.6.1",
)

git_repository(
    name = "com_github_theqrl_qrllib",
    remote = "https://github.com/theQRL/qrllib",
    commit = "99bf7f1427a0cf026b868627fbbe077de0d28522",
    recursive_init_submodules = True,
    patch_cmds = [
        "cmake -DBUILD_GO=ON",
        "make",
        "rm goqrllib/goqrllib/*.cxx",
        "rm goqrllib/dilithium/*.cxx",
        "rm goqrllib/kyber/*.cxx",
        "bazel run //:gazelle",
        "rm tests/golang/misc/*.bazel",
        "rm tests/golang/tests/*.bazel",
    ],
)

go_repository(
    name = "com_github_urfave_cli_v2",
    importpath = "github.com/urfave/cli/v2",
    sum = "h1:JTTnM6wKzdA0Jqodd966MVj4vWbbquZykeX1sKbe2C4=",
    version = "v2.2.0",
)

go_repository(
    name = "com_github_willf_bitset",
    importpath = "github.com/willf/bitset",
    sum = "h1:N7Z7E9UvjW+sGsEl7k/SJrvY2reP1A07MrGuCjIOjRE=",
    version = "v1.1.11",
)

go_repository(
    name = "com_github_willf_bloom",
    importpath = "github.com/willf/bloom",
    sum = "h1:QDacWdqcAUI1MPOwIQZRy9kOR7yxfyEmxX8Wdm2/JPA=",
    version = "v2.0.3+incompatible",
)

go_repository(
    name = "com_google_cloud_go",
    importpath = "cloud.google.com/go",
    sum = "h1:e0WKqKTd5BnrG8aKH3J3h+QvEIQtSUcf2n5UZ5ZgLtQ=",
    version = "v0.26.0",
)

go_repository(
    name = "in_gopkg_check_v1",
    importpath = "gopkg.in/check.v1",
    sum = "h1:YR8cESwS4TdDjEe65xsg0ogRM/Nc3DYOhEAlW+xobZo=",
    version = "v1.0.0-20190902080502-41f04d3bba15",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    sum = "h1:clyUAQHOM3G0M3f5vQj7LuJrETvjVot3Z5el9nffUtU=",
    version = "v2.3.0",
)

go_repository(
    name = "io_etcd_go_bbolt",
    importpath = "go.etcd.io/bbolt",
    sum = "h1:XAzx9gjCb0Rxj7EoqcClPD1d5ZBxZJk0jbuoPHenBt0=",
    version = "v1.3.5",
)

go_repository(
    name = "org_golang_google_appengine",
    importpath = "google.golang.org/appengine",
    sum = "h1:/wp5JvzpHIxhs/dumFmF7BXTf3Z+dd4uXta4kVyO508=",
    version = "v1.4.0",
)

go_repository(
    name = "org_golang_google_genproto",
    importpath = "google.golang.org/genproto",
    sum = "h1:+kGHl1aib/qcwaRi1CbqBZ1rk19r85MNUf8HaBghugY=",
    version = "v0.0.0-20200526211855-cb27e3aa2013",
)

go_repository(
    name = "org_golang_google_protobuf",
    importpath = "google.golang.org/protobuf",
    sum = "h1:Ejskq+SyPohKW+1uil0JJMtmHCgJPJ/qWTxr8qp+R4c=",
    version = "v1.25.0",
)

go_repository(
    name = "org_golang_x_crypto",
    importpath = "golang.org/x/crypto",
    sum = "h1:psW17arqaxU48Z5kZ0CQnkZWQJsqcURM6tKiBApRjXI=",
    version = "v0.0.0-20200622213623-75b288015ac9",
)

go_repository(
    name = "org_golang_x_exp",
    importpath = "golang.org/x/exp",
    sum = "h1:c2HOrn5iMezYjSlGPncknSEr/8x5LELb/ilJbXi9DEA=",
    version = "v0.0.0-20190121172915-509febef88a4",
)

go_repository(
    name = "org_golang_x_lint",
    importpath = "golang.org/x/lint",
    sum = "h1:5hukYrvBGR8/eNkX5mdUezrA6JiaEZDtJb9Ei+1LlBs=",
    version = "v0.0.0-20190930215403-16217165b5de",
)

go_repository(
    name = "org_golang_x_oauth2",
    importpath = "golang.org/x/oauth2",
    sum = "h1:vEDujvNQGv4jgYKudGeI/+DAX4Jffq6hpD55MmoEvKs=",
    version = "v0.0.0-20180821212333-d2e6202438be",
)

go_repository(
    name = "org_golang_x_sync",
    importpath = "golang.org/x/sync",
    sum = "h1:WXEvlFVvvGxCJLG6REjsT03iWnKLEWinaScsxF2Vm2o=",
    version = "v0.0.0-20200317015054-43a5402ce75a",
)

go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
    sum = "h1:Ih9Yo4hSPImZOpfGuA4bR/ORKTAbhZo2AbWNRCnevdo=",
    version = "v0.0.0-20200625212154-ddb9806d33ae",
)

go_repository(
    name = "org_golang_x_time",
    importpath = "golang.org/x/time",
    sum = "h1:EHBhcS0mlXEAVwNyO2dLfjToGsyY4j24pTs2ScHnX7s=",
    version = "v0.0.0-20200630173020-3af7569d3a1e",
)

go_repository(
    name = "org_golang_x_tools",
    importpath = "golang.org/x/tools",
    sum = "h1:VvQyQJN0tSuecqgcIxMWnnfG5kSmgy9KZR9sW3W5QeA=",
    version = "v0.0.0-20191216052735-49a3e744a425",
)

go_repository(
    name = "org_golang_x_xerrors",
    importpath = "golang.org/x/xerrors",
    sum = "h1:E7g+9GITq07hpfrRu66IVDexMakfv52eLZ2CXBWiKr4=",
    version = "v0.0.0-20191204190536-9bdfabe68543",
)

go_repository(
    name = "com_github_fsnotify_fsnotify",
    importpath = "github.com/fsnotify/fsnotify",
    sum = "h1:IXs+QLmnXW2CcXuY+8Mzv/fWEsPGWxqefPtCP5CnV9I=",
    version = "v1.4.7",
)

go_repository(
    name = "com_github_hpcloud_tail",
    importpath = "github.com/hpcloud/tail",
    sum = "h1:nfCOvKYfkgYP8hkirhJocXT2+zOD8yUNjXaWfTlyFKI=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_kr_pretty",
    importpath = "github.com/kr/pretty",
    sum = "h1:s5hAObm+yFO5uHYt5dYjxi2rXrsnmRpJx4OYvIWUaQs=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_kr_pty",
    importpath = "github.com/kr/pty",
    sum = "h1:VkoXIwSboBpnk99O/KFauAEILuNHv5DVFKZMBN/gUgw=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_kr_text",
    importpath = "github.com/kr/text",
    sum = "h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    importpath = "github.com/mattn/go-colorable",
    sum = "h1:G1f5SKeVxmagw/IyvzvtZE4Gybcc4Tr1tf7I8z0XgOg=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    importpath = "github.com/mattn/go-isatty",
    sum = "h1:tHXDdz1cpzGaovsTB+TVB8q90WEokoVmfMqoVcrLUgw=",
    version = "v0.0.5",
)

go_repository(
    name = "com_github_mgutz_ansi",
    importpath = "github.com/mgutz/ansi",
    sum = "h1:5PJl274Y63IEHC+7izoQE9x6ikvDFZS2mDVS3drnohI=",
    version = "v0.0.0-20200706080929-d51e80ef957d",
)

go_repository(
    name = "com_github_onsi_ginkgo",
    importpath = "github.com/onsi/ginkgo",
    sum = "h1:mFwc4LvZ0xpSvDZ3E+k8Yte0hLOMxXUlP+yXtJqkYfQ=",
    version = "v1.12.1",
)

go_repository(
    name = "com_github_onsi_gomega",
    importpath = "github.com/onsi/gomega",
    sum = "h1:R1uwffexN6Pr340GtYRIdZmAiN4J+iw6WG4wog1DUXg=",
    version = "v1.9.0",
)

go_repository(
    name = "com_github_stretchr_objx",
    importpath = "github.com/stretchr/objx",
    sum = "h1:2vfRuCMp5sSVIDSqO8oNnWJq7mPa6KVP3iPIwFBuy8A=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_x_cray_logrus_prefixed_formatter",
    importpath = "github.com/x-cray/logrus-prefixed-formatter",
    sum = "h1:00txxvfBM9muc0jiLIEAkAcIMJzfthRT6usrui8uGmg=",
    version = "v0.5.2",
)

go_repository(
    name = "in_gopkg_fsnotify_v1",
    importpath = "gopkg.in/fsnotify.v1",
    sum = "h1:xOHLXZwVvI9hhs+cLKq5+I5onOuwQLhQwiu63xxlHs4=",
    version = "v1.4.7",
)

go_repository(
    name = "in_gopkg_natefinch_lumberjack_v2",
    importpath = "gopkg.in/natefinch/lumberjack.v2",
    sum = "h1:1Lc07Kr7qY4U2YPouBjpCLxpiyxIVoxqXgkXLknAOE8=",
    version = "v2.0.0",
)

go_repository(
    name = "in_gopkg_tomb_v1",
    importpath = "gopkg.in/tomb.v1",
    sum = "h1:uRGJdciOHaEIrze2W8Q3AKkepLTh2hOroT7a+7czfdQ=",
    version = "v1.0.0-20141024135613-dd632973f1e7",
)

go_repository(
    name = "com_github_theqrl_go_qrllib_crypto",
    importpath = "github.com/theQRL/go-qrllib-crypto",
    sum = "h1:MzWCvHs74DJ+Tp3AKaWklNUv+5uUZrn7rHjVVqV/QK4=",
    version = "v0.0.0-20201216132758-851d39cc0d43",
)

go_repository(
    name = "in_gopkg_yaml_v3",
    importpath = "gopkg.in/yaml.v3",
    sum = "h1:dUUwHk2QECo/6vqA44rthZ8ie2QXMNeKRTHCNY2nXvo=",
    version = "v3.0.0-20200313102051-9f266ea9e77c",
)

go_repository(
    name = "com_github_aead_siphash",
    importpath = "github.com/aead/siphash",
    sum = "h1:FwHfE/T45KPKYuuSAKyyvE+oPWcaQ+CUmFW0bPlM+kg=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_andreasbriese_bbloom",
    importpath = "github.com/AndreasBriese/bbloom",
    sum = "h1:HD8gA2tkByhMAwYaFAX9w2l7vxvBQ5NMoxDrkhqhtn4=",
    version = "v0.0.0-20190306092124-e2d15f34fcf9",
)

go_repository(
    name = "com_github_armon_consul_api",
    importpath = "github.com/armon/consul-api",
    sum = "h1:G1bPvciwNyF7IUmKXNt9Ak3m6u9DE1rF+RmtIkBpVdA=",
    version = "v0.0.0-20180202201655-eb2c6b5be1b6",
)

go_repository(
    name = "com_github_btcsuite_btcd",
    importpath = "github.com/btcsuite/btcd",
    sum = "h1:Ik4hyJqN8Jfyv3S4AGBOmyouMsYE3EdYODkMbQjwPGw=",
    version = "v0.20.1-beta",
)

go_repository(
    name = "com_github_btcsuite_btclog",
    importpath = "github.com/btcsuite/btclog",
    sum = "h1:bAs4lUbRJpnnkd9VhRV3jjAVU7DJVjMaK+IsvSeZvFo=",
    version = "v0.0.0-20170628155309-84c8d2346e9f",
)

go_repository(
    name = "com_github_btcsuite_btcutil",
    importpath = "github.com/btcsuite/btcutil",
    sum = "h1:yJzD/yFppdVCf6ApMkVy8cUxV0XrxdP9rVf6D87/Mng=",
    version = "v0.0.0-20190425235716-9e5f4b9a998d",
)

go_repository(
    name = "com_github_btcsuite_go_socks",
    importpath = "github.com/btcsuite/go-socks",
    sum = "h1:R/opQEbFEy9JGkIguV40SvRY1uliPX8ifOvi6ICsFCw=",
    version = "v0.0.0-20170105172521-4720035b7bfd",
)

go_repository(
    name = "com_github_btcsuite_goleveldb",
    importpath = "github.com/btcsuite/goleveldb",
    sum = "h1:qdGvebPBDuYDPGi1WCPjy1tGyMpmDK8IEapSsszn7HE=",
    version = "v0.0.0-20160330041536-7834afc9e8cd",
)

go_repository(
    name = "com_github_btcsuite_snappy_go",
    importpath = "github.com/btcsuite/snappy-go",
    sum = "h1:ZA/jbKoGcVAnER6pCHPEkGdZOV7U1oLUedErBHCUMs0=",
    version = "v0.0.0-20151229074030-0bdef8d06723",
)

go_repository(
    name = "com_github_btcsuite_websocket",
    importpath = "github.com/btcsuite/websocket",
    sum = "h1:R8vQdOQdZ9Y3SkEwmHoWBmX1DNXhXZqlTpq6s4tyJGc=",
    version = "v0.0.0-20150119174127-31079b680792",
)

go_repository(
    name = "com_github_btcsuite_winsvc",
    importpath = "github.com/btcsuite/winsvc",
    sum = "h1:J9B4L7e3oqhXOcm+2IuNApwzQec85lE+QaikUcCs+dk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_cespare_xxhash",
    importpath = "github.com/cespare/xxhash",
    sum = "h1:a6HrQnmkObjyL+Gs60czilIUGqrzKutQD6XZog3p+ko=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_cncf_udpa_go",
    importpath = "github.com/cncf/udpa/go",
    sum = "h1:WBZRG4aNOuI15bLRrCgN8fCq8E5Xuty6jGbmSNEvSsU=",
    version = "v0.0.0-20191209042840-269d4d468f6f",
)

go_repository(
    name = "com_github_coreos_etcd",
    importpath = "github.com/coreos/etcd",
    sum = "h1:jFneRYjIvLMLhDLCzuTuU4rSJUjRplcJQ7pD7MnhC04=",
    version = "v3.3.10+incompatible",
)

go_repository(
    name = "com_github_coreos_go_etcd",
    importpath = "github.com/coreos/go-etcd",
    sum = "h1:bXhRBIXoTm9BYHS3gE0TtQuyNZyeEMux2sDi4oo5YOo=",
    version = "v2.0.0+incompatible",
)

go_repository(
    name = "com_github_coreos_go_semver",
    importpath = "github.com/coreos/go-semver",
    sum = "h1:wkHLiw0WNATZnSG7epLsujiMCgPAc9xhjJ4tgnAxmfM=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_cpuguy83_go_md2man",
    importpath = "github.com/cpuguy83/go-md2man",
    sum = "h1:BSKMNlYxDvnunlTymqtgONjNnaRV1sTpcovwwjF22jk=",
    version = "v1.0.10",
)

go_repository(
    name = "com_github_davidlazar_go_crypto",
    importpath = "github.com/davidlazar/go-crypto",
    sum = "h1:6xT9KW8zLC5IlbaIF5Q7JNieBoACT7iW0YTxQHR0in0=",
    version = "v0.0.0-20170701192655-dcfb0a7ac018",
)

go_repository(
    name = "com_github_dgraph_io_badger",
    importpath = "github.com/dgraph-io/badger",
    sum = "h1:w9pSFNSdq/JPM1N12Fz/F/bzo993Is1W+Q7HjPzi7yg=",
    version = "v1.6.1",
)

go_repository(
    name = "com_github_dgraph_io_ristretto",
    importpath = "github.com/dgraph-io/ristretto",
    sum = "h1:a5WaUrDa0qm0YrAAS1tUykT5El3kt62KNZZeMxQn3po=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_dgryski_go_farm",
    importpath = "github.com/dgryski/go-farm",
    sum = "h1:tdlZCpZ/P9DhczCTSixgIKmwPv6+wP5DGjqLYw5SUiA=",
    version = "v0.0.0-20190423205320-6a90982ecee2",
)

go_repository(
    name = "com_github_dustin_go_humanize",
    importpath = "github.com/dustin/go-humanize",
    sum = "h1:VSnTsYCnlFHaM2/igO1h6X3HA71jcobQuxemgkq4zYo=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_flynn_noise",
    importpath = "github.com/flynn/noise",
    sum = "h1:u/UEqS66A5ckRmS4yNpjmVH56sVtS/RfclBAYocb4as=",
    version = "v0.0.0-20180327030543-2492fe189ae6",
)

go_repository(
    name = "com_github_go_check_check",
    importpath = "github.com/go-check/check",
    sum = "h1:0gkP6mzaMqkmpcJYCFOLkIBwI7xFExG03bbkOkCvUPI=",
    version = "v0.0.0-20180628173108-788fd7840127",
)

go_repository(
    name = "com_github_gogo_protobuf",
    importpath = "github.com/gogo/protobuf",
    sum = "h1:DqDEcV5aeaTmdFBePNpYsp3FlcVH/2ISVVM9Qf8PSls=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_golang_groupcache",
    importpath = "github.com/golang/groupcache",
    sum = "h1:ZgQEtGgCBiWRM39fZuwSd1LwSqqSW0hOdXCYYDX0R3I=",
    version = "v0.0.0-20190702054246-869f871628b6",
)

go_repository(
    name = "com_github_golang_snappy",
    importpath = "github.com/golang/snappy",
    sum = "h1:woRePGFeVFfLKN/pOkfl+p/TAqKOfFu+7KPlMVpok/w=",
    version = "v0.0.0-20180518054509-2e65f85255db",
)

go_repository(
    name = "com_github_google_gopacket",
    importpath = "github.com/google/gopacket",
    sum = "h1:rMrlX2ZY2UbvT+sdz3+6J+pp2z+msCq9MxTU6ymxbBY=",
    version = "v1.1.17",
)

go_repository(
    name = "com_github_google_renameio",
    importpath = "github.com/google/renameio",
    sum = "h1:GOZbcHa3HfsPKPlmyPyN2KEohoMXOhdMbHrvbpl2QaA=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_google_uuid",
    importpath = "github.com/google/uuid",
    sum = "h1:Gkbcsh/GbpXz7lPftLA3P6TYMwjCLYm83jiFQZF/3gY=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_gorilla_websocket",
    importpath = "github.com/gorilla/websocket",
    sum = "h1:+/TMaTYc4QFitKJxsQ7Yye35DkWvkdLcvGKqM+x0Ufc=",
    version = "v1.4.2",
)

go_repository(
    name = "com_github_gxed_hashland_keccakpg",
    importpath = "github.com/gxed/hashland/keccakpg",
    sum = "h1:wrk3uMNaMxbXiHibbPO4S0ymqJMm41WiudyFSs7UnsU=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_gxed_hashland_murmur3",
    importpath = "github.com/gxed/hashland/murmur3",
    sum = "h1:SheiaIt0sda5K+8FLz952/1iWS9zrnKsEJaOJu4ZbSc=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    importpath = "github.com/hashicorp/golang-lru",
    sum = "h1:YDjusn29QI/Das2iO9M0BHnIbxPeyuCHsjMW+lJfyTc=",
    version = "v0.5.4",
)

go_repository(
    name = "com_github_hashicorp_hcl",
    importpath = "github.com/hashicorp/hcl",
    sum = "h1:0Anlzjpi4vEasTeNFn2mLJgTSwt0+6sfsiTG8qcWGx4=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_huin_goupnp",
    importpath = "github.com/huin/goupnp",
    sum = "h1:wg75sLpL6DZqwHQN6E1Cfk6mtfzS45z8OV+ic+DtHRo=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_huin_goutil",
    importpath = "github.com/huin/goutil",
    sum = "h1:vlNjIqmUZ9CMAWsbURYl3a6wZbw7q5RHVvlXTNS/Bs8=",
    version = "v0.0.0-20170803182201-1ca381bf3150",
)

go_repository(
    name = "com_github_inconshreveable_mousetrap",
    importpath = "github.com/inconshreveable/mousetrap",
    sum = "h1:Z8tu5sraLXCXIcARxBp/8cbvlwVa7Z1NHg9XEKhtSvM=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    importpath = "github.com/ipfs/go-cid",
    sum = "h1:ysQJVJA3fNDF1qigJbsSQOdjhVLsOEoPdh0+R97k3jY=",
    version = "v0.0.7",
)

go_repository(
    name = "com_github_ipfs_go_datastore",
    importpath = "github.com/ipfs/go-datastore",
    sum = "h1:rjvQ9+muFaJ+QZ7dN5B1MSDNQ0JVZKkkES/rMZmA8X8=",
    version = "v0.4.4",
)

go_repository(
    name = "com_github_ipfs_go_detect_race",
    importpath = "github.com/ipfs/go-detect-race",
    sum = "h1:qX/xay2W3E4Q1U7d9lNs1sU9nvguX0a7319XbyQ6cOk=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_ipfs_go_ds_badger",
    importpath = "github.com/ipfs/go-ds-badger",
    sum = "h1:J27YvAcpuA5IvZUbeBxOcQgqnYHUPxoygc6QxxkodZ4=",
    version = "v0.2.3",
)

go_repository(
    name = "com_github_ipfs_go_ds_leveldb",
    importpath = "github.com/ipfs/go-ds-leveldb",
    sum = "h1:QmQoAJ9WkPMUfBLnu1sBVy0xWWlJPg0m4kRAiJL9iaw=",
    version = "v0.4.2",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_delay",
    importpath = "github.com/ipfs/go-ipfs-delay",
    sum = "h1:NAviDvJ0WXgD+yiL2Rj35AmnfgI11+pHXbdciD917U0=",
    version = "v0.0.0-20181109222059-70721b86a9a8",
)

go_repository(
    name = "com_github_ipfs_go_ipfs_util",
    importpath = "github.com/ipfs/go-ipfs-util",
    sum = "h1:59Sswnk1MFaiq+VcaknX7aYEyGyGDAA73ilhEK2POp8=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_ipfs_go_log",
    importpath = "github.com/ipfs/go-log",
    sum = "h1:6nLQdX4W8P9yZZFH7mO+X/PzjN8Laozm/lMJ6esdgzY=",
    version = "v1.0.4",
)

go_repository(
    name = "com_github_ipfs_go_log_v2",
    importpath = "github.com/ipfs/go-log/v2",
    sum = "h1:G4TtqN+V9y9HY9TA6BwbCVyyBZ2B9MbCjR2MtGx8FR0=",
    version = "v2.1.1",
)

go_repository(
    name = "com_github_jackpal_gateway",
    importpath = "github.com/jackpal/gateway",
    sum = "h1:qzXWUJfuMdlLMtt0a3Dgt+xkWQiA5itDEITVJtuSwMc=",
    version = "v1.0.5",
)

go_repository(
    name = "com_github_jackpal_go_nat_pmp",
    importpath = "github.com/jackpal/go-nat-pmp",
    sum = "h1:KzKSgb7qkJvOUTqYl9/Hg/me3pWgBmERKrTGD7BdWus=",
    version = "v1.0.2",
)

go_repository(
    name = "com_github_jbenet_go_cienv",
    importpath = "github.com/jbenet/go-cienv",
    sum = "h1:Vc/s0QbQtoxX8MwwSLWWh+xNNZvM3Lw7NsTcHrvvhMc=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_jbenet_go_temp_err_catcher",
    importpath = "github.com/jbenet/go-temp-err-catcher",
    sum = "h1:zpb3ZH6wIE8Shj2sKS+khgRvf7T7RABoLk/+KKHggpk=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_jbenet_goprocess",
    importpath = "github.com/jbenet/goprocess",
    sum = "h1:DRGOFReOMqqDNXwW70QkacFW0YN9QnwLV0Vqk+3oU0o=",
    version = "v0.1.4",
)

go_repository(
    name = "com_github_jessevdk_go_flags",
    importpath = "github.com/jessevdk/go-flags",
    sum = "h1:4IU2WS7AumrZ/40jfhf4QVDMsQwqA7VEHozFRrGARJA=",
    version = "v1.4.0",
)

go_repository(
    name = "com_github_jrick_logrotate",
    importpath = "github.com/jrick/logrotate",
    sum = "h1:lQ1bL/n9mBNeIXoTUoYRlK4dHuNJVofX9oWqBtPnSzI=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_kami_zh_go_capturer",
    importpath = "github.com/kami-zh/go-capturer",
    sum = "h1:cVtBfNW5XTHiKQe7jDaDBSh/EVM4XLPutLAGboIXuM0=",
    version = "v0.0.0-20171211120116-e492ea43421d",
)

go_repository(
    name = "com_github_kisielk_errcheck",
    importpath = "github.com/kisielk/errcheck",
    sum = "h1:reN85Pxc5larApoH1keMBiu2GWtPqXQ1nc9gx+jOU+E=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_kisielk_gotool",
    importpath = "github.com/kisielk/gotool",
    sum = "h1:AV2c/EiW3KqPNT9ZKl07ehoAGi4C5/01Cfbblndcapg=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_kkdai_bstream",
    importpath = "github.com/kkdai/bstream",
    sum = "h1:FOOIBWrEkLgmlgGfMuZT83xIwfPDxEI2OHu6xUmJMFE=",
    version = "v0.0.0-20161212061736-f391b8402d23",
)

go_repository(
    name = "com_github_koron_go_ssdp",
    importpath = "github.com/koron/go-ssdp",
    sum = "h1:68u9r4wEvL3gYg2jvAOgROwZ3H+Y3hIDk4tbbmIjcYQ=",
    version = "v0.0.0-20191105050749-2e1c40ed0b5d",
)

go_repository(
    name = "com_github_kubuxu_go_os_helper",
    importpath = "github.com/Kubuxu/go-os-helper",
    sum = "h1:EJiD2VUQyh5A9hWJLmc6iWg6yIcJ7jpBcwC8GMGXfDk=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_libp2p_go_addr_util",
    importpath = "github.com/libp2p/go-addr-util",
    sum = "h1:7cWK5cdA5x72jX0g8iLrQWm5TRJZ6CzGdPEhWj7plWU=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_libp2p_go_buffer_pool",
    importpath = "github.com/libp2p/go-buffer-pool",
    sum = "h1:QNK2iAFa8gjAe1SPz6mHSMuCcjs+X1wlHzeOSqcmlfs=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_libp2p_go_conn_security_multistream",
    importpath = "github.com/libp2p/go-conn-security-multistream",
    sum = "h1:uNiDjS58vrvJTg9jO6bySd1rMKejieG7v45ekqHbZ1M=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_eventbus",
    importpath = "github.com/libp2p/go-eventbus",
    sum = "h1:VanAdErQnpTioN2TowqNcOijf6YwhuODe4pPKSDpxGc=",
    version = "v0.2.1",
)

go_repository(
    name = "com_github_libp2p_go_flow_metrics",
    importpath = "github.com/libp2p/go-flow-metrics",
    sum = "h1:8tAs/hSdNvUiLgtlSy3mxwxWP4I9y/jlkPFT7epKdeM=",
    version = "v0.0.3",
)

go_repository(
    name = "com_github_libp2p_go_libp2p",
    importpath = "github.com/libp2p/go-libp2p",
    sum = "h1:tDdrXARSghmusdm0nf1U/4M8aj8Rr0V2IzQOXmbzQ3s=",
    version = "v0.13.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_autonat",
    importpath = "github.com/libp2p/go-libp2p-autonat",
    sum = "h1:3y8XQbpr+ssX8QfZUHekjHCYK64sj6/4hnf/awD4+Ug=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_blankhost",
    importpath = "github.com/libp2p/go-libp2p-blankhost",
    sum = "h1:3EsGAi0CBGcZ33GwRuXEYJLLPoVWyXJ1bcJzAJjINkk=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_circuit",
    importpath = "github.com/libp2p/go-libp2p-circuit",
    sum = "h1:eqQ3sEYkGTtybWgr6JLqJY6QLtPWRErvFjFDfAOO1wc=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_core",
    importpath = "github.com/libp2p/go-libp2p-core",
    sum = "h1:5K3mT+64qDTKbV3yTdbMCzJ7O6wbNsavAEb8iqBvBcI=",
    version = "v0.8.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_crypto",
    importpath = "github.com/libp2p/go-libp2p-crypto",
    sum = "h1:k9MFy+o2zGDNGsaoZl0MA3iZ75qXxr9OOoAZF+sD5OQ=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_discovery",
    importpath = "github.com/libp2p/go-libp2p-discovery",
    sum = "h1:Qfl+e5+lfDgwdrXdu4YNCWyEo3fWuP+WgN9mN0iWviQ=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_loggables",
    importpath = "github.com/libp2p/go-libp2p-loggables",
    sum = "h1:h3w8QFfCt2UJl/0/NW4K829HX/0S4KD31PQ7m8UXXO8=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_mplex",
    importpath = "github.com/libp2p/go-libp2p-mplex",
    sum = "h1:/pyhkP1nLwjG3OM+VuaNJkQT/Pqq73WzB3aDN3Fx1sc=",
    version = "v0.4.1",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_nat",
    importpath = "github.com/libp2p/go-libp2p-nat",
    sum = "h1:wMWis3kYynCbHoyKLPBEMu4YRLltbm8Mk08HGSfvTkU=",
    version = "v0.0.6",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_netutil",
    importpath = "github.com/libp2p/go-libp2p-netutil",
    sum = "h1:zscYDNVEcGxyUpMd0JReUZTrpMfia8PmLKcKF72EAMQ=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_noise",
    importpath = "github.com/libp2p/go-libp2p-noise",
    sum = "h1:vqYQWvnIcHpIoWJKC7Al4D6Hgj0H012TuXRhPwSMGpQ=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peer",
    importpath = "github.com/libp2p/go-libp2p-peer",
    sum = "h1:EQ8kMjaCUwt/Y5uLgjT8iY2qg0mGUT0N1zUjer50DsY=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_peerstore",
    importpath = "github.com/libp2p/go-libp2p-peerstore",
    sum = "h1:2ACefBX23iMdJU9Ke+dcXt3w86MIryes9v7In4+Qq3U=",
    version = "v0.2.6",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_pnet",
    importpath = "github.com/libp2p/go-libp2p-pnet",
    sum = "h1:J6htxttBipJujEjz1y0a5+eYoiPcFHhSYHH6na5f0/k=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_secio",
    importpath = "github.com/libp2p/go-libp2p-secio",
    sum = "h1:rLLPvShPQAcY6eNurKNZq3eZjPWfU9kXF2eI9jIYdrg=",
    version = "v0.2.2",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_swarm",
    importpath = "github.com/libp2p/go-libp2p-swarm",
    sum = "h1:hahq/ijRoeH6dgROOM8x7SeaKK5VgjjIr96vdrT+NUA=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_testing",
    importpath = "github.com/libp2p/go-libp2p-testing",
    sum = "h1:PrwHRi0IGqOwVQWR3xzgigSlhlLfxgfXgkHxr77EghQ=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_tls",
    importpath = "github.com/libp2p/go-libp2p-tls",
    sum = "h1:twKMhMu44jQO+HgQK9X8NHO5HkeJu2QbhLzLJpa8oNM=",
    version = "v0.1.3",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_transport_upgrader",
    importpath = "github.com/libp2p/go-libp2p-transport-upgrader",
    sum = "h1:xwj4h3hJdBrxqMOyMUjwscjoVst0AASTsKtZiTChoHI=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_yamux",
    importpath = "github.com/libp2p/go-libp2p-yamux",
    sum = "h1:sX4WQPHMhRxJE5UZTfjEuBvlQWXB5Bo3A2JK9ZJ9EM0=",
    version = "v0.5.1",
)

go_repository(
    name = "com_github_libp2p_go_maddr_filter",
    importpath = "github.com/libp2p/go-maddr-filter",
    sum = "h1:4ACqZKw8AqiuJfwFGq1CYDFugfXTOos+qQ3DETkhtCE=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_libp2p_go_mplex",
    importpath = "github.com/libp2p/go-mplex",
    sum = "h1:U1T+vmCYJaEoDJPV1aq31N56hS+lJgb397GsylNSgrU=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_libp2p_go_msgio",
    importpath = "github.com/libp2p/go-msgio",
    sum = "h1:lQ7Uc0kS1wb1EfRxO2Eir/RJoHkHn7t6o+EiwsYIKJA=",
    version = "v0.0.6",
)

go_repository(
    name = "com_github_libp2p_go_nat",
    importpath = "github.com/libp2p/go-nat",
    sum = "h1:qxnwkco8RLKqVh1NmjQ+tJ8p8khNLFxuElYG/TwqW4Q=",
    version = "v0.0.5",
)

go_repository(
    name = "com_github_libp2p_go_netroute",
    importpath = "github.com/libp2p/go-netroute",
    sum = "h1:1ngWRx61us/EpaKkdqkMjKk/ufr/JlIFYQAxV2XX8Ig=",
    version = "v0.1.3",
)

go_repository(
    name = "com_github_libp2p_go_openssl",
    importpath = "github.com/libp2p/go-openssl",
    sum = "h1:eCAzdLejcNVBzP/iZM9vqHnQm+XyCEbSSIheIPRGNsw=",
    version = "v0.0.7",
)

go_repository(
    name = "com_github_libp2p_go_reuseport",
    importpath = "github.com/libp2p/go-reuseport",
    sum = "h1:XSG94b1FJfGA01BUrT82imejHQyTxO4jEWqheyCXYvU=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_libp2p_go_reuseport_transport",
    importpath = "github.com/libp2p/go-reuseport-transport",
    sum = "h1:OZGz0RB620QDGpv300n1zaOcKGGAoGVf8h9txtt/1uM=",
    version = "v0.0.4",
)

go_repository(
    name = "com_github_libp2p_go_sockaddr",
    importpath = "github.com/libp2p/go-sockaddr",
    sum = "h1:tCuXfpA9rq7llM/v834RKc/Xvovy/AqM9kHvTV/jY/Q=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_libp2p_go_stream_muxer",
    importpath = "github.com/libp2p/go-stream-muxer",
    sum = "h1:Ce6e2Pyu+b5MC1k3eeFtAax0pW4gc6MosYSLV05UeLw=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_libp2p_go_stream_muxer_multistream",
    importpath = "github.com/libp2p/go-stream-muxer-multistream",
    sum = "h1:TqnSHPJEIqDEO7h1wZZ0p3DXdvDSiLHQidKKUGZtiOY=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_libp2p_go_tcp_transport",
    importpath = "github.com/libp2p/go-tcp-transport",
    sum = "h1:ExZiVQV+h+qL16fzCWtd1HSzPsqWottJ8KXwWaVi8Ns=",
    version = "v0.2.1",
)

go_repository(
    name = "com_github_libp2p_go_ws_transport",
    importpath = "github.com/libp2p/go-ws-transport",
    sum = "h1:9tvtQ9xbws6cA5LvqdE6Ne3vcmGB4f1z9SByggk4s0k=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_libp2p_go_yamux",
    importpath = "github.com/libp2p/go-yamux",
    sum = "h1:P1Fe9vF4th5JOxxgQvfbOHkrGqIZniTLf+ddhZp8YTI=",
    version = "v1.4.1",
)

go_repository(
    name = "com_github_libp2p_go_yamux_v2",
    importpath = "github.com/libp2p/go-yamux/v2",
    sum = "h1:vSGhAy5u6iHBq11ZDcyHH4Blcf9xlBhT4WQDoOE90LU=",
    version = "v2.0.0",
)

go_repository(
    name = "com_github_magiconair_properties",
    importpath = "github.com/magiconair/properties",
    sum = "h1:LLgXmsheXeRoUOBOjtwPQCWIYqM/LU1ayDtDePerRcY=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_mailru_easyjson",
    importpath = "github.com/mailru/easyjson",
    sum = "h1:2gxZ0XQIU/5z3Z3bUBu+FXuk2pFbkN6tcwi/pjyaDic=",
    version = "v0.0.0-20180823135443-60711f1a8329",
)

go_repository(
    name = "com_github_miekg_dns",
    importpath = "github.com/miekg/dns",
    sum = "h1:sJFOl9BgwbYAWOGEwr61FU28pqsBNdpRBnhGXtO06Oo=",
    version = "v1.1.31",
)

go_repository(
    name = "com_github_minio_blake2b_simd",
    importpath = "github.com/minio/blake2b-simd",
    sum = "h1:lYpkrQH5ajf0OXOcUbGjvZxxijuBwbbmlSxLiuofa+g=",
    version = "v0.0.0-20160723061019-3f5f724cb5b1",
)

go_repository(
    name = "com_github_minio_sha256_simd",
    importpath = "github.com/minio/sha256-simd",
    sum = "h1:5QHSlgo3nt5yKOJrC7W8w7X+NFl8cMPZm96iu8kKUJU=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_mitchellh_go_homedir",
    importpath = "github.com/mitchellh/go-homedir",
    sum = "h1:lukF9ziXFxDFPkA1vsr5zpc1XuPDn/wFntq5mG+4E0Y=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_mitchellh_mapstructure",
    importpath = "github.com/mitchellh/mapstructure",
    sum = "h1:fmNYVwqnSfB9mZU6OS2O6GsXM+wcskZDuKQzvN1EDeE=",
    version = "v1.1.2",
)

go_repository(
    name = "com_github_mr_tron_base58",
    importpath = "github.com/mr-tron/base58",
    sum = "h1:T/HDJBh4ZCPbU39/+c3rRvE0uKBQlU27+QI8LJ4t64o=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_multiformats_go_base32",
    importpath = "github.com/multiformats/go-base32",
    sum = "h1:tw5+NhuwaOjJCC5Pp82QuXbrmLzWg7uxlMFp8Nq/kkI=",
    version = "v0.0.3",
)

go_repository(
    name = "com_github_multiformats_go_base36",
    importpath = "github.com/multiformats/go-base36",
    sum = "h1:JR6TyF7JjGd3m6FbLU2cOxhC0Li8z8dLNGQ89tUg4F4=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr",
    importpath = "github.com/multiformats/go-multiaddr",
    sum = "h1:1bxa+W7j9wZKTZREySx1vPMs2TqrYWjVZ7zE6/XLG1I=",
    version = "v0.3.1",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    importpath = "github.com/multiformats/go-multiaddr-dns",
    sum = "h1:YWJoIDwLePniH7OU5hBnDZV6SWuvJqJ0YtN6pLeH9zA=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_fmt",
    importpath = "github.com/multiformats/go-multiaddr-fmt",
    sum = "h1:WLEFClPycPkp4fnIzoFoV9FVd49/eQsuaL3/CWe167E=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_net",
    importpath = "github.com/multiformats/go-multiaddr-net",
    sum = "h1:MSXRGN0mFymt6B1yo/6BPnIRpLPEnKgQNvVfCX5VDJk=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_multiformats_go_multibase",
    importpath = "github.com/multiformats/go-multibase",
    sum = "h1:l/B6bJDQjvQ5G52jw4QGSYeOTZoAwIO77RblWplfIqk=",
    version = "v0.0.3",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    importpath = "github.com/multiformats/go-multihash",
    sum = "h1:QoBceQYQQtNUuf6s7wHxnE2c8bhbMqhfGzNI032se/I=",
    version = "v0.0.14",
)

go_repository(
    name = "com_github_multiformats_go_multistream",
    importpath = "github.com/multiformats/go-multistream",
    sum = "h1:6AuNmQVKUkRnddw2YiDjt5Elit40SFxMJkVnhmETXtU=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_multiformats_go_varint",
    importpath = "github.com/multiformats/go-varint",
    sum = "h1:gk85QWKxh3TazbLxED/NlDVv8+q+ReFJk7Y2W/KhfNY=",
    version = "v0.0.6",
)

go_repository(
    name = "com_github_nxadm_tail",
    importpath = "github.com/nxadm/tail",
    sum = "h1:DQuhQpB1tVlglWS2hLQ5OV6B5r8aGxSrPc5Qo6uTN78=",
    version = "v1.4.4",
)

go_repository(
    name = "com_github_oneofone_xxhash",
    importpath = "github.com/OneOfOne/xxhash",
    sum = "h1:KMrpdQIwFcEqXDklaen+P1axHaj9BSKzvpUUfnHldSE=",
    version = "v1.2.2",
)

go_repository(
    name = "com_github_opentracing_opentracing_go",
    importpath = "github.com/opentracing/opentracing-go",
    sum = "h1:uEJPy/1a5RIPAJ0Ov+OIO8OxWu77jEv+1B0VhjKrZUs=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_pelletier_go_toml",
    importpath = "github.com/pelletier/go-toml",
    sum = "h1:T5zMGML61Wp+FlcbWjRDT7yAxhJNAiPPLOFECq181zc=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
    sum = "h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt4=",
    version = "v0.9.1",
)

go_repository(
    name = "com_github_rogpeppe_go_internal",
    importpath = "github.com/rogpeppe/go-internal",
    sum = "h1:RR9dF3JtopPvtkroDZuVD7qquD0bnHlKSqaQhgwt8yk=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_russross_blackfriday",
    importpath = "github.com/russross/blackfriday",
    sum = "h1:HyvC0ARfnZBqnXwABFeSZHpKvJHJJfPz81GNueLj0oo=",
    version = "v1.5.2",
)

go_repository(
    name = "com_github_smola_gocompat",
    importpath = "github.com/smola/gocompat",
    sum = "h1:6b1oIMlUXIpz//VKEDzPVBK8KG7beVwmHIUEBIs/Pns=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_spacemonkeygo_openssl",
    importpath = "github.com/spacemonkeygo/openssl",
    sum = "h1:/eS3yfGjQKG+9kayBkj0ip1BGhq6zJ3eaVksphxAaek=",
    version = "v0.0.0-20181017203307-c2dcc5cca94a",
)

go_repository(
    name = "com_github_spacemonkeygo_spacelog",
    importpath = "github.com/spacemonkeygo/spacelog",
    sum = "h1:RC6RW7j+1+HkWaX/Yh71Ee5ZHaHYt7ZP4sQgUrm6cDU=",
    version = "v0.0.0-20180420211403-2296661a0572",
)

go_repository(
    name = "com_github_spf13_afero",
    importpath = "github.com/spf13/afero",
    sum = "h1:m8/z1t7/fwjysjQRYbP0RD+bUIF/8tJwPdEZsI83ACI=",
    version = "v1.1.2",
)

go_repository(
    name = "com_github_spf13_cast",
    importpath = "github.com/spf13/cast",
    sum = "h1:oget//CVOEoFewqQxwr0Ej5yjygnqGkvggSE/gB35Q8=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_spf13_cobra",
    importpath = "github.com/spf13/cobra",
    sum = "h1:f0B+LkLX6DtmRH1isoNA9VTtNUK9K8xYd28JNNfOv/s=",
    version = "v0.0.5",
)

go_repository(
    name = "com_github_spf13_jwalterweatherman",
    importpath = "github.com/spf13/jwalterweatherman",
    sum = "h1:XHEdyB+EcvlqZamSM4ZOMGlc93t6AcsBEu9Gc1vn7yk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_spf13_pflag",
    importpath = "github.com/spf13/pflag",
    sum = "h1:zPAT6CGy6wXeQ7NtTnaTerfKOsV6V6F8agHXFiazDkg=",
    version = "v1.0.3",
)

go_repository(
    name = "com_github_spf13_viper",
    importpath = "github.com/spf13/viper",
    sum = "h1:VUFqw5KcqRf7i70GOzW7N+Q7+gxVBkSSqiXB12+JQ4M=",
    version = "v1.3.2",
)

go_repository(
    name = "com_github_src_d_envconfig",
    importpath = "github.com/src-d/envconfig",
    sum = "h1:/AJi6DtjFhZKNx3OB2qMsq7y4yT5//AeSZIe7rk+PX8=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_syndtr_goleveldb",
    importpath = "github.com/syndtr/goleveldb",
    sum = "h1:fBdIW9lB4Iz0n9khmH8w27SJ3QEJ7+IgjPEwGSZiFdE=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_theqrl_go_libp2p_qrl",
    importpath = "github.com/theQRL/go-libp2p-qrl",
    sum = "h1:9CCrC48WCkqZO7dsX4jTvQwXwWNy6ANnP/Uk6NrpStY=",
    version = "v0.0.0-20201218151637-0bc463da5c16",
)

go_repository(
    name = "com_github_ugorji_go_codec",
    importpath = "github.com/ugorji/go/codec",
    sum = "h1:3SVOIvH7Ae1KRYyQWRjXWJEA9sS/c/pjvH++55Gr648=",
    version = "v0.0.0-20181204163529-d75b2dcb6bc8",
)

go_repository(
    name = "com_github_whyrusleeping_go_keyspace",
    importpath = "github.com/whyrusleeping/go-keyspace",
    sum = "h1:EKhdznlJHPMoKr0XTrX+IlJs1LH3lyx2nfr1dOlZ79k=",
    version = "v0.0.0-20160322163242-5b898ac5add1",
)

go_repository(
    name = "com_github_whyrusleeping_go_logging",
    importpath = "github.com/whyrusleeping/go-logging",
    sum = "h1:fwpzlmT0kRC/Fmd0MdmGgJG/CXIZ6gFq46FQZjprUcc=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_whyrusleeping_mafmt",
    importpath = "github.com/whyrusleeping/mafmt",
    sum = "h1:TCghSl5kkwEE0j+sU/gudyhVMRlpBin8fMBBHg59EbA=",
    version = "v1.2.8",
)

go_repository(
    name = "com_github_whyrusleeping_mdns",
    importpath = "github.com/whyrusleeping/mdns",
    sum = "h1:Y1/FEOpaCpD21WxrmfeIYCFPuVPRCY2XZTWzTNHGw30=",
    version = "v0.0.0-20190826153040-b9b60ed33aa9",
)

go_repository(
    name = "com_github_whyrusleeping_multiaddr_filter",
    importpath = "github.com/whyrusleeping/multiaddr-filter",
    sum = "h1:E9S12nwJwEOXe2d6gT6qxdvqMnNq+VnSsKPgm2ZZNds=",
    version = "v0.0.0-20160516205228-e903e4adabd7",
)

go_repository(
    name = "com_github_xordataexchange_crypt",
    importpath = "github.com/xordataexchange/crypt",
    sum = "h1:ESFSdwYZvkeru3RtdrYueztKhOBCSAAzS4Gf+k0tEow=",
    version = "v0.0.3-0.20170626215501-b2862e3d0a77",
)

go_repository(
    name = "in_gopkg_errgo_v2",
    importpath = "gopkg.in/errgo.v2",
    sum = "h1:0vLT13EuvQ0hNvakwLuFZ/jYrLp5F3kcWHXdRggjCE8=",
    version = "v2.1.0",
)

go_repository(
    name = "in_gopkg_src_d_go_cli_v0",
    importpath = "gopkg.in/src-d/go-cli.v0",
    sum = "h1:mXa4inJUuWOoA4uEROxtJ3VMELMlVkIxIfcR0HBekAM=",
    version = "v0.0.0-20181105080154-d492247bbc0d",
)

go_repository(
    name = "in_gopkg_src_d_go_log_v1",
    importpath = "gopkg.in/src-d/go-log.v1",
    sum = "h1:heWvX7J6qbGWbeFS/aRmiy1eYaT+QMV6wNvHDyMjQV4=",
    version = "v1.0.1",
)

go_repository(
    name = "io_opencensus_go",
    importpath = "go.opencensus.io",
    sum = "h1:LYy1Hy3MJdrCdMwwzxA/dRok4ejH+RwNGbuoD9fCjto=",
    version = "v0.22.4",
)

go_repository(
    name = "org_golang_x_mod",
    importpath = "golang.org/x/mod",
    sum = "h1:WG0RUwxtNT4qqaXX3DPA8zHFNm/D9xaBpxzHt1WcA/E=",
    version = "v0.1.1-0.20191105210325-c90efee705ee",
)

go_repository(
    name = "org_uber_go_atomic",
    importpath = "go.uber.org/atomic",
    sum = "h1:Ezj3JGmsOnG1MoRWQkPBsKLe9DwWD9QeXzTRzzldNVk=",
    version = "v1.6.0",
)

go_repository(
    name = "org_uber_go_goleak",
    importpath = "go.uber.org/goleak",
    sum = "h1:qsup4IcBdlmsnGfqyLl4Ntn3C2XCCuKAE7DwHpScyUo=",
    version = "v1.0.0",
)

go_repository(
    name = "org_uber_go_multierr",
    importpath = "go.uber.org/multierr",
    sum = "h1:KCa4XfM8CWFCpxXRGok+Q0SS/0XBhMDbHHGABQLvD2A=",
    version = "v1.5.0",
)

go_repository(
    name = "org_uber_go_tools",
    importpath = "go.uber.org/tools",
    sum = "h1:0mgffUl7nfd+FpvXMVz4IDEaUSmT1ysygQC7qYo7sG4=",
    version = "v0.0.0-20190618225709-2cfd321de3ee",
)

go_repository(
    name = "org_uber_go_zap",
    importpath = "go.uber.org/zap",
    sum = "h1:ZZCA22JRF2gQE5FoNmhmrf7jeJJ2uhqDUNRYKm8dvmM=",
    version = "v1.15.0",
)

[![PkgGoDev](https://img.shields.io/badge/go.dev-docs-007d9c?style=flat-square&logo=go&logoColor=white)](https://pkg.go.dev/github.com/tonistiigi/fsutil)
[![CI Status](https://img.shields.io/github/actions/workflow/status/tonistiigi/fsutil/ci.yml?label=ci&logo=github&style=flat-square)](https://github.com/tonistiigi/fsutil/actions?query=workflow%3Aci)
[![Go Report Card](https://goreportcard.com/badge/github.com/tonistiigi/fsutil?style=flat-square)](https://goreportcard.com/report/github.com/tonistiigi/fsutil)
[![Codecov](https://img.shields.io/codecov/c/github/tonistiigi/fsutil?logo=codecov&style=flat-square)](https://codecov.io/gh/tonistiigi/fsutil)

Incremental file directory sync tools in golang.

```
BENCH_FILE_SIZE=10000 docker buildx bake bench-root
...
#20 0.276 + CGO_ENABLED=0 xx-go test -benchmem '-bench=.' '-run=^$' .
#20 0.276 + tee /tmp/fsutil.log
#20 3.783 BenchmarkWalker/depth_1_target-32                30003             39975 ns/op            9233 B/op        174 allocs/op
#20 5.399 BenchmarkWalker/depth_1_doublestar_target-32             29533             40131 ns/op            9280 B/op        175 allocs/op
#20 7.058 BenchmarkWalker/depth_2_star_target-32                    1250           942068 ns/op          200675 B/op       3971 allocs/op
#20 8.484 BenchmarkWalker/depth_2_doublestar_target-32              1263           950370 ns/op          195964 B/op       3908 allocs/op
#20 11.52 BenchmarkWalker/depth_3_star_star_target-32                 39          27486182 ns/op         5218849 B/op     100914 allocs/op
#20 15.38 BenchmarkWalker/depth_3_doublestar_target-32                40          26643263 ns/op         5202168 B/op     100827 allocs/op
#20 19.48 BenchmarkWalker/depth_4_star_star_star_target-32            22          47317811 ns/op         6568429 B/op     119420 allocs/op
#20 23.72 BenchmarkWalker/depth_4_doublestar_target-32                26          43800871 ns/op         6549295 B/op     119314 allocs/op
#20 29.48 BenchmarkWalker/depth_5_star_star_star_star_target-32                       52          20775183 ns/op         2474331 B/op      42818 allocs/op
#20 31.60 BenchmarkWalker/depth_5_doublestar_target-32                                51          20152359 ns/op         2460404 B/op      42698 allocs/op
#20 34.01 BenchmarkWalker/depth_6_star_star_star_star_star_target-32                  30          36303006 ns/op         3995022 B/op      67770 allocs/op
#20 36.72 BenchmarkWalker/depth_6_doublestar_target-32                                31          36020767 ns/op         3981605 B/op      67634 allocs/op
#20 39.47 BenchmarkWalker/depth_6_doublestar_exclude_star_star_doublestar-32                3175            390627 ns/op           46942 B/op       1006 allocs/op
#20 42.40 + cd bench
#20 42.40 + CGO_ENABLED=0 xx-go test -benchmem '-bench=.' '-run=^$' .
#20 42.40 + tee /tmp/bench.log
#20 43.06 BenchmarkCopyWithTar10-32                          298           4009415 ns/op          836739 B/op        723 allocs/op
#20 46.32 BenchmarkCopyWithTar50-32                           54          21041722 ns/op         4762559 B/op       3881 allocs/op
#20 50.19 BenchmarkCopyWithTar200-32                          19          59134695 ns/op        17963038 B/op      13265 allocs/op
#20 52.53 BenchmarkCopyWithTar1000-32                          5         209966591 ns/op        69498342 B/op      47683 allocs/op
#20 55.76 BenchmarkCPA10-32                                  366           3206154 ns/op            7102 B/op         77 allocs/op
#20 59.05 BenchmarkCPA50-32                                   82          14457219 ns/op            7102 B/op         77 allocs/op
#20 61.63 BenchmarkCPA200-32                                  28          43178227 ns/op            7101 B/op         77 allocs/op
#20 64.44 BenchmarkCPA1000-32                                  8         126501459 ns/op            7102 B/op         77 allocs/op
#20 66.60 BenchmarkDiffCopy10-32                             458           2623652 ns/op          218457 B/op       1123 allocs/op
#20 70.09 BenchmarkDiffCopy50-32                              98          12777867 ns/op         1243712 B/op       5456 allocs/op
#20 72.97 BenchmarkDiffCopy200-32                             32          37305271 ns/op         4682522 B/op      18644 allocs/op
#20 75.94 BenchmarkDiffCopy1000-32                             9         113655143 ns/op        17239066 B/op      67655 allocs/op
#20 78.28 BenchmarkDiffCopyProto10-32                        457           2684875 ns/op          232619 B/op       1143 allocs/op
#20 81.84 BenchmarkDiffCopyProto50-32                         96          12785543 ns/op         1279922 B/op       5569 allocs/op
#20 84.67 BenchmarkDiffCopyProto200-32                        33          36892044 ns/op         4736327 B/op      18997 allocs/op
#20 87.66 BenchmarkDiffCopyProto1000-32                        9         113899974 ns/op        17408194 B/op      68896 allocs/op
#20 90.00 BenchmarkIncrementalDiffCopy10-32                 1525            745277 ns/op          119711 B/op       1015 allocs/op
#20 92.37 BenchmarkIncrementalDiffCopy50-32                  823           1551876 ns/op          455277 B/op       4287 allocs/op
#20 94.60 BenchmarkIncrementalDiffCopy200-32                 267           4842579 ns/op         1335725 B/op      13606 allocs/op
#20 97.07 BenchmarkIncrementalDiffCopy1000-32                 80          15097221 ns/op         4551946 B/op      46855 allocs/op
#20 100.1 BenchmarkIncrementalDiffCopy5000-32                 14          73468188 ns/op        24508496 B/op     266767 allocs/op
#20 105.2 BenchmarkIncrementalDiffCopy10000-32                 9         160205175 ns/op        43698322 B/op     472975 allocs/op
#20 109.4 BenchmarkIncrementalCopyWithTar10-32               507           2284446 ns/op          841826 B/op        702 allocs/op
#20 111.4 BenchmarkIncrementalCopyWithTar50-32                94          12455430 ns/op         4782267 B/op       3803 allocs/op
#20 112.8 BenchmarkIncrementalCopyWithTar200-32               25          48493228 ns/op        18070606 B/op      13101 allocs/op
#20 114.7 BenchmarkIncrementalCopyWithTar1000-32               6         176957907 ns/op        69691893 B/op      47325 allocs/op
#20 116.2 BenchmarkIncrementalRsync10-32                      26          43711169 ns/op            6600 B/op         69 allocs/op
#20 117.6 BenchmarkIncrementalRsync50-32                      25          46322066 ns/op            6608 B/op         69 allocs/op
#20 119.0 BenchmarkIncrementalRsync200-32                     22          51190363 ns/op            6608 B/op         69 allocs/op
#20 120.7 BenchmarkIncrementalRsync1000-32                    16          65351370 ns/op            6600 B/op         69 allocs/op
#20 124.0 BenchmarkIncrementalRsync5000-32                     8         132279028 ns/op            6608 B/op         69 allocs/op
#20 129.6 BenchmarkIncrementalRsync10000-32                    6         192954744 ns/op            6608 B/op         69 allocs/op
#20 133.6 BenchmarkRsync10-32                                 25          47151091 ns/op            6605 B/op         69 allocs/op
#20 135.0 BenchmarkRsync50-32                                 19          59415573 ns/op            6606 B/op         69 allocs/op
#20 136.5 BenchmarkRsync200-32                                13          90882993 ns/op            6606 B/op         69 allocs/op
#20 138.7 BenchmarkRsync1000-32                                6         202851848 ns/op            6606 B/op         69 allocs/op
#20 140.8 BenchmarkGnuTar10-32                               278           4317587 ns/op           14191 B/op        151 allocs/op
#20 143.9 BenchmarkGnuTar50-32                                63          19306289 ns/op           14192 B/op        151 allocs/op
#20 146.3 BenchmarkGnuTar200-32                               20          56949381 ns/op           14192 B/op        151 allocs/op
#20 148.7 BenchmarkGnuTar1000-32                               6         187783406 ns/op           14192 B/op        151 allocs/op
```

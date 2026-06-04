[![PkgGoDev](https://img.shields.io/badge/go.dev-docs-007d9c?style=flat-square&logo=go&logoColor=white)](https://pkg.go.dev/github.com/tonistiigi/fsutil)
[![CI Status](https://img.shields.io/github/actions/workflow/status/tonistiigi/fsutil/ci.yml?label=ci&logo=github&style=flat-square)](https://github.com/tonistiigi/fsutil/actions?query=workflow%3Aci)
[![Go Report Card](https://goreportcard.com/badge/github.com/tonistiigi/fsutil?style=flat-square)](https://goreportcard.com/report/github.com/tonistiigi/fsutil)
[![Codecov](https://img.shields.io/codecov/c/github/tonistiigi/fsutil?logo=codecov&style=flat-square)](https://codecov.io/gh/tonistiigi/fsutil)

Incremental file directory sync tools in golang.

```
BENCH_FILE_SIZE=10000 docker buildx bake bench-root
...
#15 0.257 + CGO_ENABLED=0 xx-go test -benchmem '-bench=.' '-run=^$' ./...
#15 0.693 BenchmarkWalker/[1]-target-32                    29266             40683 ns/op            9233 B/op        174 allocs/op
#15 2.310 BenchmarkWalker/[1]-**/target-32                 29121             40738 ns/op            9281 B/op        175 allocs/op
#15 3.967 BenchmarkWalker/[2]-*/target-32                   1250            964012 ns/op          200812 B/op       3971 allocs/op
#15 5.413 BenchmarkWalker/[2]-**/target-32                  1232            991293 ns/op          195884 B/op       3908 allocs/op
#15 8.137 BenchmarkWalker/[3]-*/*/target-32                   38          28249040 ns/op         5219115 B/op     100914 allocs/op
#15 11.80 BenchmarkWalker/[3]-**/target-32                    39          26951802 ns/op         5203848 B/op     100827 allocs/op
#15 15.61 BenchmarkWalker/[4]-*/*/*/target-32                 25          45280808 ns/op         6568953 B/op     119420 allocs/op
#15 19.60 BenchmarkWalker/[4]-**/target-32                    22          46757187 ns/op         6548530 B/op     119314 allocs/op
#15 22.66 BenchmarkWalker/[5]-*/*/*/*/target-32               51          21749543 ns/op         2472273 B/op      42817 allocs/op
#15 24.74 BenchmarkWalker/[5]-**/target-32                    57          21094871 ns/op         2461442 B/op      42699 allocs/op
#15 28.60 BenchmarkWalker/[6]-*/*/*/*/*/target-32                     28          43522799 ns/op         3996910 B/op      67770 allocs/op
#15 31.46 BenchmarkWalker/[6]-**/target-32                            32          36312221 ns/op         3979274 B/op      67633 allocs/op
#15 35.86 BenchmarkWalker/[6]-**-!*/*/**-32                         3040            378412 ns/op           47470 B/op       1006 allocs/op
#15 38.58 PASS
#15 38.58 ok    github.com/tonistiigi/fsutil    37.898s
#15 38.58 ?     github.com/tonistiigi/fsutil/cmd/receive        [no test files]
#15 38.58 ?     github.com/tonistiigi/fsutil/cmd/send   [no test files]
#15 38.58 ?     github.com/tonistiigi/fsutil/cmd/walk   [no test files]
#15 38.59 PASS
#15 38.59 ok    github.com/tonistiigi/fsutil/copy       0.005s
#15 38.59 ?     github.com/tonistiigi/fsutil/types      [no test files]
#15 38.59 ?     github.com/tonistiigi/fsutil/util       [no test files]
#15 38.61 + cd bench
#15 38.61 + CGO_ENABLED=0 xx-go test -benchmem '-bench=.' '-run=^$' ./...
#15 38.97 BenchmarkCopyWithTar10-32                          326           3764572 ns/op          906061 B/op        843 allocs/op
#15 42.10 BenchmarkCopyWithTar50-32                           55          20591865 ns/op         5015270 B/op       4516 allocs/op
#15 44.18 BenchmarkCopyWithTar200-32                          16          64855277 ns/op        18768999 B/op      15353 allocs/op
#15 46.28 BenchmarkCopyWithTar1000-32                          5         215755521 ns/op        72467841 B/op      55045 allocs/op
#15 49.48 BenchmarkCPA10-32                                  387           3117276 ns/op            7102 B/op         77 allocs/op
#15 52.84 BenchmarkCPA50-32                                   74          14216530 ns/op            7102 B/op         77 allocs/op
#15 55.16 BenchmarkCPA200-32                                  28          41337870 ns/op            7103 B/op         77 allocs/op
#15 57.90 BenchmarkCPA1000-32                                  9         120160486 ns/op            7104 B/op         77 allocs/op
#15 62.34 BenchmarkDiffCopy10-32                             472           2532670 ns/op          219677 B/op       1124 allocs/op
#15 65.81 BenchmarkDiffCopy50-32                              93          12163321 ns/op         1258443 B/op       5460 allocs/op
#15 68.43 BenchmarkDiffCopy200-32                             33          35695300 ns/op         4736101 B/op      18654 allocs/op
#15 71.35 BenchmarkDiffCopy1000-32                            10         107299221 ns/op        17376938 B/op      67675 allocs/op
#15 73.89 BenchmarkDiffCopyProto10-32                        450           2548261 ns/op          235518 B/op       1144 allocs/op
#15 77.28 BenchmarkDiffCopyProto50-32                         94          12235131 ns/op         1308093 B/op       5570 allocs/op
#15 79.95 BenchmarkDiffCopyProto200-32                        31          35108771 ns/op         4753186 B/op      19001 allocs/op
#15 82.66 BenchmarkDiffCopyProto1000-32                       10         103423551 ns/op        17448403 B/op      68910 allocs/op
#15 85.09 BenchmarkIncrementalDiffCopy10-32                 1645            738769 ns/op          119215 B/op       1014 allocs/op
#15 87.62 BenchmarkIncrementalDiffCopy50-32                  742           1483920 ns/op          440246 B/op       4278 allocs/op
#15 89.53 BenchmarkIncrementalDiffCopy200-32                 282           4371728 ns/op         1335620 B/op      13606 allocs/op
#15 91.86 BenchmarkIncrementalDiffCopy1000-32                 82          13720766 ns/op         4543527 B/op      46847 allocs/op
#15 94.69 BenchmarkIncrementalDiffCopy5000-32                 15          75319643 ns/op        24488733 B/op     266774 allocs/op
#15 99.61 BenchmarkIncrementalDiffCopy10000-32                 9         132956476 ns/op        43688770 B/op     472977 allocs/op
#15 103.3 BenchmarkIncrementalCopyWithTar10-32               441           2789884 ns/op          925828 B/op        829 allocs/op
#15 105.2 BenchmarkIncrementalCopyWithTar50-32                73          16177382 ns/op         5047137 B/op       4463 allocs/op
#15 106.7 BenchmarkIncrementalCopyWithTar200-32               18          56786225 ns/op        18940856 B/op      15255 allocs/op
#15 108.4 BenchmarkIncrementalCopyWithTar1000-32               5         223768989 ns/op        73159873 B/op      54913 allocs/op
#15 111.2 BenchmarkIncrementalRsync10-32                      26          43708755 ns/op            6608 B/op         69 allocs/op
#15 112.5 BenchmarkIncrementalRsync50-32                      25          45821410 ns/op            6608 B/op         69 allocs/op
#15 114.0 BenchmarkIncrementalRsync200-32                     22          50456838 ns/op            6608 B/op         69 allocs/op
#15 115.6 BenchmarkIncrementalRsync1000-32                    18          63784533 ns/op            6600 B/op         69 allocs/op
#15 118.8 BenchmarkIncrementalRsync5000-32                     8         135204923 ns/op            6608 B/op         69 allocs/op
#15 124.1 BenchmarkIncrementalRsync10000-32                    6         193377650 ns/op            6600 B/op         69 allocs/op
#15 127.9 BenchmarkRsync10-32                                 25          46024966 ns/op            6606 B/op         69 allocs/op
#15 129.3 BenchmarkRsync50-32                                 20          57464062 ns/op            6606 B/op         69 allocs/op
#15 130.9 BenchmarkRsync200-32                                13          86092507 ns/op            6604 B/op         69 allocs/op
#15 132.9 BenchmarkRsync1000-32                                6         180127658 ns/op            6606 B/op         69 allocs/op
#15 134.8 BenchmarkGnuTar10-32                               286           4216965 ns/op           14191 B/op        151 allocs/op
#15 137.9 BenchmarkGnuTar50-32                                61          18779459 ns/op           14192 B/op        151 allocs/op
#15 140.1 BenchmarkGnuTar200-32                               20          56019558 ns/op           14192 B/op        151 allocs/op
#15 142.4 BenchmarkGnuTar1000-32                               6         183059208 ns/op           14189 B/op        151 allocs/op
#15 144.4 PASS
#15 144.4 ok    github.com/tonistiigi/fsutil/bench      105.439s
```

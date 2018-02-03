# hashservice

A coding exercise that implements an http service that hashes strings with SHA-512 and returns base64 encoded result.

## Instructions

Download and test:

```
~/go/src/github.com/bbawn$ git clone https://github.com/bbawn/hashservice.git
Cloning into 'hashservice'...
remote: Counting objects: 52, done.
remote: Compressing objects: 100% (35/35), done.
remote: Total 52 (delta 19), reused 46 (delta 16), pack-reused 0 
Unpacking objects: 100% (52/52), done.

~/go/src/github.com/bbawn$ cd hashservice

~/go/src/github.com/bbawn/hashservice$ go test -bench .
2018/02/03 10:44:37 BenchmarkSimple b.N 1 stats total 0 average 0 ms
goos: linux
goarch: amd64
pkg: github.com/bbawn/hashservice
BenchmarkSimple-4       2018/02/03 10:44:38 BenchmarkSimple b.N 100 stats total 1 average 0.024255 ms
2018/02/03 10:44:38 BenchmarkSimple b.N 2000 stats total 1892 average 0.1634888710359408 ms
2018/02/03 10:44:40 BenchmarkSimple b.N 5000 stats total 4900 average 0.22087256428571428 ms
    5000            316458 ns/op
PASS
ok      github.com/bbawn/hashservice    2.336s
```

Install and run:
```
~/go/src/github.com/bbawn/hashservice$ hashservice &
[1] 22938

~/go/src/github.com/bbawn/hashservice$ echo $(curl -s --data "password=angryMonkey" http://localhost:8080/hash)
1

~/go/src/github.com/bbawn/hashservice$ echo $(curl -s http://localhost:8080/hash/1)
ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q==

~/go/src/github.com/bbawn/hashservice$ echo $(curl -s http://localhost:8080/stats)
{"total":1,"average":0.067323}

~/go/src/github.com/bbawn/hashservice$ echo $(curl -s http://localhost:8080/shutdown)

[1]+  Done                    hashservice

```

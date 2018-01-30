# hashservice

A coding exercise that implements an http service that hashes strings with SHA-512 and returns base64 encoded result.

## Usage

Server command-line usage:

```
$ hashservice --help
Usage of ./hashservice:
  -addr string
          http service address (default ":8080")
```

Start a server:
```
$ hashservice
```

Request a hashing/encoding operation:
```
$ curl --data "password=angryMonkey" http://localhost:8080/hash
ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q==
```

## Benchmarks

## Technical details

## Issues

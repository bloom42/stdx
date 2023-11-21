# zign


## Installation

```shell
$ go install github.com/bloom42/stdx/tools/zign@latest
```


## Usage

```shell
$ zign init
$ ls
zign.private
zign.public
```

```shell
$ zign sign -o myproject_1.4.2.zign.json zign.private file1 file2 file2...
```


```shell
$ zign verify "publicKey" myproject_1.4.2.zign.json
```

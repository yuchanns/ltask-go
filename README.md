# ltask-go

A showcase of how to build a lua library with [go.yuchanns.xyz/lua](https://github.com/yuchanns/lua) the lua go-binding.

## Instructions

### Clone the repository
```bash
git clone https://github.com/yuchanns/ltask-go

cd ltask-go

git submodule update --init --recursive
```

### Prepare the dynamic artifacts
```bash
cd lua && make lua54 && cd -
```

### Run the tests
```bash
go test -v ./...
```

## Credits

- [ltask](https://github.com/cloudwu/ltask) is a lua task library that implements an n:m scheduler, so that you can run M lua VMs on N OS threads.

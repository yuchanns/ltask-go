# ltask-go

A showcase of how to build a lua library with [go.yuchanns.xyz/lua](https://github.com/yuchanns/lua) the lua go-binding.

## Devlopment

We use Go Workspace to dev, so you can use the following commands to run the tests.

1. **Clone the repository**
```bash
git clone --recurse-submodules  https://github.com/yuchanns/ltask-go

cd ltask-go
```

2. **Prepare the dynamic artifacts**
```bash
cd lua && make lua54 && cd -
```

3. **Run the tests**
```bash
cd ltask && go test -v ./...
```


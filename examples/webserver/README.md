# Example: Webserver

All sources (lua scripts and shared objects included) are compiled into a single binary.

```bash
git submodule update --init --recursive

cd examples/webserver

luamake

cd -

go build ./examples/webserver

./webserver
```

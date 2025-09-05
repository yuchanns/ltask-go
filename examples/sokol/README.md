# Example: Integration with Sokol App

Handle sokol app callbacks from Lua by forwarding external messages via `require("ltask.bootstrap").external_sender`.

```bash
git submodule update --init --recursive

cd examples/sokol

luamake

go run . ./test.lua
```

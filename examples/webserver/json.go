package main

import (
	"encoding/json"
	"fmt"

	"go.yuchanns.xyz/lua"
)

func luaOpenJSON(L *lua.State) int {
	l := []*lua.Reg{
		{"encode", lua.NewCallback(luaJsonEncode, L.Lib())},
		{"decode", lua.NewCallback(luaJsonDecode, L.Lib())},
	}
	L.NewLib(l)
	return 1
}

// luaJsonEncode encodes a Lua table to JSON format.
func luaJsonEncode(L *lua.State) int {
	if L.GetTop() != 1 {
		L.PushNil()
		L.PushString("json.encode: wrong number of arguments")
		return 2
	}
	if L.Type(1) != lua.LUA_TTABLE {
		L.PushNil()
		L.PushString("json.encode: argument must be a table")
		return 2
	}

	L.PushValue(1)
	data, err := luaToGo(L, -1)
	L.Pop(1)

	if err != nil {
		L.PushNil()
		L.PushString("json.encode: " + err.Error())
		return 2
	}

	jsonStr, err := json.Marshal(data)
	if err != nil {
		L.PushNil()
		L.PushString("json.encode: " + err.Error())
		return 2
	}

	L.PushString(string(jsonStr))
	return 1
}

// luaToGo converts a Lua table at idx into Go types: map[string]interface{} or []interface{}
// Supports nested tables.
func luaToGo(L *lua.State, idx int) (interface{}, error) {
	// Absolute index
	if idx < 0 {
		idx = L.GetTop() + idx + 1
	}
	if L.Type(idx) != lua.LUA_TTABLE {
		return nil, fmt.Errorf("expected a table at index %d, got %s", idx, L.TypeName(L.Type(idx)))
	}

	// Decide if it's an array (all keys are integers and start from 1) or map
	size := int(L.RawLen(idx))
	isArray := true
	L.PushNil()
	for L.Next(idx) {
		if !L.IsNumber(-2) {
			isArray = false
			L.Pop(1)
			break
		}
		L.Pop(1)
	}

	L.PushNil()
	if isArray && size > 0 {
		arr := make([]interface{}, size)
		for i := 1; i <= size; i++ {
			L.RawGetI(idx, int64(i))
			val, err := luaValueToGo(L, -1)
			if err != nil {
				return nil, err
			}
			arr[i-1] = val
			L.Pop(1)
		}
		return arr, nil
	} else {
		mp := make(map[string]interface{})
		for L.Next(idx) {
			// Stack: key (-2), value (-1)
			key := L.ToString(-2)
			val, err := luaValueToGo(L, -1)
			if err != nil {
				return nil, err
			}
			mp[key] = val
			L.Pop(1)
		}
		return mp, nil
	}
}

// luaValueToGo converts a Lua value at idx to Go value.
func luaValueToGo(L *lua.State, idx int) (interface{}, error) {
	switch L.Type(idx) {
	case lua.LUA_TNIL:
		return nil, nil
	case lua.LUA_TBOOLEAN:
		return L.ToBoolean(idx), nil
	case lua.LUA_TNUMBER:
		return L.ToNumber(idx), nil
	case lua.LUA_TSTRING:
		return L.ToString(idx), nil
	case lua.LUA_TTABLE:
		return luaToGo(L, idx)
	default:
		return nil, fmt.Errorf("unsupported value type: %s", L.TypeName(L.Type(idx)))
	}
}

// jsonDecode decodes a JSON string into a Lua table.
func luaJsonDecode(L *lua.State) int {
	if L.GetTop() != 1 {
		L.PushNil()
		L.PushString("json.decode: wrong number of arguments")
		return 2
	}
	if !L.IsString(1) {
		L.PushNil()
		L.PushString("json.decode: argument must be a string")
		return 2
	}
	jsonStr := L.ToString(1)
	L.SetTop(0)
	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		L.PushNil()
		L.PushString("json.decode: " + err.Error())
		return 2
	}
	err = jsonDecode(L, data)
	if err != nil {
		L.PushNil()
		L.PushString("json.decode: " + err.Error())
		return 2
	}
	return 1
}

func jsonDecode(L *lua.State, data interface{}) (err error) {
	switch v := data.(type) {
	case map[string]interface{}:
		L.NewTable()
		for key, value := range v {
			L.PushString(key)
			switch val := value.(type) {
			case string:
				L.PushString(val)
			case float64:
				L.PushNumber(val)
			case bool:
				L.PushBoolean(val)
			case nil:
				L.PushNil()
			default:
				// Handle nested tables or arrays
				if err = jsonDecode(L, val); err != nil {
					return err
				}
			}
			L.SetTable(-3)
		}
	case []interface{}:
		L.NewTable()
		for i, value := range v {
			i := int64(i)
			switch val := value.(type) {
			case string:
				L.PushString(val)
			case float64:
				L.PushNumber(val)
			case bool:
				L.PushBoolean(val)
			case nil:
				L.PushNil()
			default:
				// Handle nested tables or arrays
				if err = jsonDecode(L, val); err != nil {
					return err
				}
			}
			L.RawSetI(-2, i+1)
		}
	default:
		return fmt.Errorf("unsupported type %T", data)
	}
	return nil
}

package ltask

import (
	"bytes"
	"encoding/binary"
	"unsafe"

	"github.com/smasher164/mem"
	"go.yuchanns.xyz/lua"
)

const (
	blockSize    = 128
	maxDepth     = 31
	maxReference = 32
)

const (
	serdeTypeBoolean      = 0
	serdeTypeBooleanNil   = 0
	serdeTypeBooleanFalse = 1
	serdeTypeBooleanTrue  = 2

	serdeTypeNumber      = 1
	serdeTypeNumberZero  = 0
	serdeTypeNumberByte  = 1
	serdeTypeNumberWord  = 2
	serdeTypeNumberDword = 4
	serdeTypeNumberQword = 6
	serdeTypeNumberReal  = 8

	serdeTypeShortString = 3
	serdeTypeLongString  = 4

	serdeTypeTable = 5
)

func combineType(t, v uint8) uint8 {
	return t | (v << 3)
}

type stack struct {
	depth    int
	objectID int
	ancestor []int // stack of Lua stack indices
}

type writeBlock struct {
	buf       *bytes.Buffer
	stack     stack
	reference map[unsafe.Pointer]int // object pointer -> id
}

func newWriteBlock() *writeBlock {
	return &writeBlock{
		buf:       new(bytes.Buffer),
		stack:     stack{ancestor: make([]int, 0, maxDepth)},
		reference: make(map[unsafe.Pointer]int),
	}
}

func (wb *writeBlock) writeByte(b byte) {
	wb.buf.WriteByte(b)
}
func (wb *writeBlock) writeUint16(x uint16) {
	_ = binary.Write(wb.buf, binary.LittleEndian, x)
}
func (wb *writeBlock) writeUint32(x uint32) {
	_ = binary.Write(wb.buf, binary.LittleEndian, x)
}
func (wb *writeBlock) writeInt32(x int32) {
	_ = binary.Write(wb.buf, binary.LittleEndian, x)
}
func (wb *writeBlock) writeInt64(x int64) {
	_ = binary.Write(wb.buf, binary.LittleEndian, x)
}
func (wb *writeBlock) writeFloat64(x float64) {
	_ = binary.Write(wb.buf, binary.LittleEndian, x)
}
func (wb *writeBlock) writeBytes(b []byte) {
	wb.buf.Write(b)
}

func (wb *writeBlock) packOne(L *lua.State, index int) {
	typ := L.Type(index)
	switch typ {
	case lua.LUA_TNIL:
		tag := combineType(serdeTypeBoolean, serdeTypeBooleanNil)
		wb.writeByte(tag)
	case lua.LUA_TNUMBER:
		if L.IsInteger(index) {
			x := L.ToInteger(index)
			if x == 0 {
				tag := combineType(serdeTypeNumber, serdeTypeNumberZero)
				wb.writeByte(tag)
			} else if x < 0 || x > 0xFFFFFFFF {
				tag := combineType(serdeTypeNumber, serdeTypeNumberQword)
				wb.writeByte(tag)
				wb.writeInt64(x)
			} else if x < 0x100 {
				tag := combineType(serdeTypeNumber, serdeTypeNumberByte)
				wb.writeByte(tag)
				wb.writeByte(byte(x))
			} else if x < 0x10000 {
				tag := combineType(serdeTypeNumber, serdeTypeNumberWord)
				wb.writeByte(tag)
				wb.writeUint16(uint16(x))
			} else {
				tag := combineType(serdeTypeNumber, serdeTypeNumberDword)
				wb.writeByte(tag)
				wb.writeUint32(uint32(x))
			}
		} else {
			n := L.ToNumber(index)
			tag := combineType(serdeTypeNumber, serdeTypeNumberReal)
			wb.writeByte(tag)
			wb.writeFloat64(n)
		}
	case lua.LUA_TBOOLEAN:
		b := L.ToBoolean(index)
		var subtype uint8
		if b {
			subtype = serdeTypeBooleanTrue
		} else {
			subtype = serdeTypeBooleanFalse
		}
		tag := combineType(serdeTypeBoolean, subtype)
		wb.writeByte(tag)
	case lua.LUA_TSTRING:
		str := L.ToString(index)
		length := len(str)
		if length < 32 {
			tag := combineType(serdeTypeShortString, uint8(length))
			wb.writeByte(tag)
			if length > 0 {
				wb.writeBytes([]byte(str))
			}
		} else if length < 0x10000 {
			tag := combineType(serdeTypeLongString, 2)
			wb.writeByte(tag)
			wb.writeUint16(uint16(length))
			wb.writeBytes([]byte(str))
		} else {
			tag := combineType(serdeTypeLongString, 4)
			wb.writeByte(tag)
			wb.writeUint32(uint32(length))
			wb.writeBytes([]byte(str))
		}
	case lua.LUA_TLIGHTUSERDATA:
		udPtr := L.ToUserData(index)
		tag := combineType(7 /* TYPE_USERDATA */, 0 /* POINTER */)
		wb.writeByte(tag)
		ptrBytes := (*[unsafe.Sizeof(udPtr)]byte)(unsafe.Pointer(&udPtr))[:]
		wb.writeBytes(ptrBytes)
	case lua.LUA_TFUNCTION:
		fnPtr := L.ToCFunction(index)
		tag := combineType(7 /* TYPE_USERDATA */, 1 /* CFUNCTION */)
		wb.writeByte(tag)
		fnBytes := (*[unsafe.Sizeof(fnPtr)]byte)(unsafe.Pointer(&fnPtr))[:]
		wb.writeBytes(fnBytes)
	case lua.LUA_TTABLE:
		if index < 0 {
			index = L.GetTop() + index + 1
		}
		objPtr := L.ToPointer(index)
		if id, ok := wb.reference[objPtr]; ok {
			tag := combineType(7 /* TYPE_REF */, 31 /* EXTEND_NUMBER */)
			wb.writeByte(tag)
			wb.writeInt32(int32(id))
			return
		}
		wb.stack.objectID++
		wb.reference[objPtr] = wb.stack.objectID
		if wb.stack.depth < maxDepth {
			wb.stack.ancestor = append(wb.stack.ancestor, index)
		}
		wb.stack.depth++
		wb.packTable(L, index)
		wb.stack.depth--
		if wb.stack.depth < len(wb.stack.ancestor) {
			wb.stack.ancestor = wb.stack.ancestor[:wb.stack.depth]
		}
	default:
		L.Errorf("Unsupported type %s to serialize", L.TypeName(typ))
	}
}

func (wb *writeBlock) packTable(L *lua.State, index int) {
	arraySize := L.RawLen(index)
	var tag uint8
	if arraySize >= 31 {
		tag = combineType(serdeTypeTable, 31)
		wb.writeByte(tag)
		wb.writeInt32(int32(arraySize))
	} else {
		tag = combineType(serdeTypeTable, uint8(arraySize))
		wb.writeByte(tag)
	}
	for i := int64(1); i <= int64(arraySize); i++ {
		L.RawGetI(index, i)
		wb.packOne(L, -1)
		L.Pop(1)
	}
	L.PushNil()
	for L.Next(index) {
		if L.Type(-2) == lua.LUA_TNUMBER && L.IsInteger(-2) {
			x := L.ToInteger(-2)
			if x > 0 && x <= int64(arraySize) {
				L.Pop(1)
				continue
			}
		}
		wb.packOne(L, -2)
		wb.packOne(L, -1)
		L.Pop(1)
	}
	wb.writeByte(combineType(serdeTypeBoolean, serdeTypeBooleanNil))
}

func (wb *writeBlock) packFrom(L *lua.State, from int) {
	top := L.GetTop()
	n := top - from
	L.PushNil()
	L.PushNil()
	for i := 1; i <= n; i++ {
		wb.packOne(L, from+i)
	}
}

func luaSerdePack(L *lua.State) int {
	buf := serdePack(L, 0)
	sz := len(buf)
	var b byte
	ptr := mem.Alloc(uint(sz) * uint(unsafe.Sizeof(b)))
	buffer := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), sz)
	copy(buffer, buf)
	L.PushLightUserData(ptr)
	L.PushInteger(int64(sz))
	return 2
}

func serdePack(L *lua.State, from int) (buf []byte) {
	wb := newWriteBlock()
	wb.packFrom(L, from)
	return wb.buf.Bytes()
}

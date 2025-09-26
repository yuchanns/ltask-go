package ltask

import (
	"bytes"
	"encoding/binary"
	"math"
	"unsafe"

	"go.yuchanns.xyz/lua"
)

// TODO: this module should be speparated to a standalone library once it is stable enough.

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

	serdeTypeUserData        = 2
	serdeTypeUserDataPointer = 0
	serdeTypeUserDataCFunc   = 1

	serdeTypeShortString = 3
	serdeTypeLongString  = 4

	serdeTypeTable     = 5
	serdeTypeTableMark = 6
	serdeTypeRef       = 7

	maxCookie    = 32
	extendNumber = maxCookie - 1
)

func combineType(t, v uint8) uint8 {
	return t | (v << 3)
}

type reference struct {
	object  unsafe.Pointer
	address *uint8
}

type stack struct {
	depth    int
	objectID int
	refIndex int
	ancestor []int
}

type writeBlock struct {
	buf       *bytes.Buffer
	stack     stack
	reference []reference
	refMap    map[unsafe.Pointer]int
}

func newWriteBlock() *writeBlock {
	return &writeBlock{
		buf:       new(bytes.Buffer),
		stack:     stack{ancestor: make([]int, 0, maxDepth)},
		reference: make([]reference, 0, maxReference),
		refMap:    make(map[unsafe.Pointer]int),
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

func (wb *writeBlock) writeUint64(x uint64) {
	_ = binary.Write(wb.buf, binary.LittleEndian, x)
}

func (wb *writeBlock) writeFloat64(x float64) {
	_ = binary.Write(wb.buf, binary.LittleEndian, x)
}

func (wb *writeBlock) writeBytes(b []byte) {
	wb.buf.Write(b)
}

func (wb *writeBlock) getAddress() *uint8 {
	data := wb.buf.Bytes()
	if len(data) == 0 {
		return nil
	}
	return &data[len(data)-1]
}

func (wb *writeBlock) writeInteger(v int64) {
	if v == 0 {
		tag := combineType(serdeTypeNumber, serdeTypeNumberZero)
		wb.writeByte(tag)
	} else if v != int64(int32(v)) {
		tag := combineType(serdeTypeNumber, serdeTypeNumberQword)
		wb.writeByte(tag)
		wb.writeInt64(v)
	} else if v < 0 {
		tag := combineType(serdeTypeNumber, serdeTypeNumberDword)
		wb.writeByte(tag)
		wb.writeInt32(int32(v))
	} else if v < 0x100 {
		tag := combineType(serdeTypeNumber, serdeTypeNumberByte)
		wb.writeByte(tag)
		wb.writeByte(uint8(v))
	} else if v < 0x10000 {
		tag := combineType(serdeTypeNumber, serdeTypeNumberWord)
		wb.writeByte(tag)
		wb.writeUint16(uint16(v))
	} else {
		tag := combineType(serdeTypeNumber, serdeTypeNumberDword)
		wb.writeByte(tag)
		wb.writeUint32(uint32(v))
	}
}

func (wb *writeBlock) writeString(str string) {
	length := len(str)
	if length < maxCookie {
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
}

func (wb *writeBlock) writePointer(ptr unsafe.Pointer, subtype uint8) {
	tag := combineType(serdeTypeUserData, subtype)
	wb.writeByte(tag)
	wb.writeUint64(uint64(uintptr(ptr)))
}

func (wb *writeBlock) writeNil() {
	tag := combineType(serdeTypeBoolean, serdeTypeBooleanNil)
	wb.writeByte(tag)
}

func (wb *writeBlock) writeBoolean(b bool) {
	var subtype uint8
	if b {
		subtype = serdeTypeBooleanTrue
	} else {
		subtype = serdeTypeBooleanFalse
	}
	tag := combineType(serdeTypeBoolean, subtype)
	wb.writeByte(tag)
}

func (wb *writeBlock) writeReal(v float64) {
	tag := combineType(serdeTypeNumber, serdeTypeNumberReal)
	wb.writeByte(tag)
	wb.writeFloat64(v)
}

func (wb *writeBlock) markTable(L *lua.State, index int) {
	obj := L.ToPointer(index)
	s := &wb.stack
	id := s.objectID
	s.objectID++

	addr := wb.getAddress()

	if id == maxReference {
		L.CreateTable(0, maxReference+1)
		L.Replace(s.refIndex)
		L.CreateTable(maxReference+1, 0)
		L.Replace(s.refIndex + 1)

		for i := range maxReference {
			L.PushInteger(int64(i + 1))
			L.RawSetP(s.refIndex, wb.reference[i].object)
			if wb.reference[i].address != nil {
				L.PushLightUserData(unsafe.Pointer(wb.reference[i].address))
				L.RawSetI(s.refIndex+1, int64(i+1))
			}
		}
	}

	if id < maxReference {
		if len(wb.reference) <= id {
			wb.reference = append(wb.reference, reference{})
		}
		wb.reference[id].object = obj
		wb.reference[id].address = addr
	} else {
		id++
		L.PushInteger(int64(id))
		L.RawSetP(s.refIndex, obj)
		L.PushLightUserData(addr)
		L.RawSetI(s.refIndex+1, int64(id))
	}
}

func (wb *writeBlock) changeMark(addr *uint8) {
	*addr = combineType(serdeTypeTableMark, *addr>>3)
}

func (wb *writeBlock) lookupRef(L *lua.State, obj unsafe.Pointer) int {
	s := &wb.stack
	if s.objectID <= maxReference {
		for i := range s.objectID {
			if wb.reference[i].object == obj {
				if wb.reference[i].address != nil {
					wb.changeMark(wb.reference[i].address)
					wb.reference[i].address = nil
				}
				return i + 1
			}
		}
		return 0
	} else {
		if L.RawGetP(s.refIndex, obj) != lua.LUA_TNUMBER {
			L.Pop(1)
			return 0
		}
		id := int(L.ToInteger(-1))
		L.Pop(1)

		if L.RawGetI(s.refIndex+1, int64(id)) == lua.LUA_TLIGHTUSERDATA {
			tag := (*uint8)(L.ToUserData(-1))
			L.Pop(1)
			wb.changeMark(tag)
			L.PushNil()
			L.RawSetI(s.refIndex+1, int64(id))
		} else {
			L.Pop(1)
		}
		return id
	}
}

func (wb *writeBlock) refAncestor(L *lua.State, index int) bool {
	if wb.stack.depth == 0 || wb.stack.depth >= maxDepth {
		return false
	}

	obj := L.ToPointer(index)
	for i := wb.stack.depth - 1; i >= 0; i-- {
		ancestor := L.ToPointer(wb.stack.ancestor[i])
		if ancestor == obj {
			tag := combineType(serdeTypeRef, uint8(i))
			wb.writeByte(tag)
			return true
		}
	}
	return false
}

func (wb *writeBlock) refObject(L *lua.State, index int) bool {
	obj := L.ToPointer(index)
	id := wb.lookupRef(L, obj)
	if id > 0 {
		tag := combineType(serdeTypeRef, extendNumber)
		wb.writeByte(tag)
		wb.writeInteger(int64(id))
		return true
	}
	return false
}

func (wb *writeBlock) packTableArray(L *lua.State, index int) int {
	arraySize := int(L.RawLen(index))
	if arraySize >= extendNumber {
		tag := combineType(serdeTypeTable, extendNumber)
		wb.writeByte(tag)
		wb.writeInteger(int64(arraySize))
	} else {
		tag := combineType(serdeTypeTable, uint8(arraySize))
		wb.writeByte(tag)
	}

	for i := 1; i <= arraySize; i++ {
		L.RawGetI(index, int64(i))
		wb.packOne(L, -1)
		L.Pop(1)
	}

	return arraySize
}

func (wb *writeBlock) packTableHash(L *lua.State, index int, arraySize int) {
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
	wb.writeNil()
}

func (wb *writeBlock) packTableMetaPairs(L *lua.State, index int) {
	tag := combineType(serdeTypeTable, 0)
	wb.writeByte(tag)
	L.PushValue(index)
	L.Call(1, 3)

	for {
		L.PushValue(-2)
		L.PushValue(-2)
		L.Copy(-5, -3)
		L.Call(2, 2)
		if L.Type(-2) == lua.LUA_TNIL {
			L.Pop(4)
			break
		}
		wb.packOne(L, -2)
		wb.packOne(L, -1)
		L.Pop(1)
	}
	wb.writeNil()
}

func (wb *writeBlock) packTable(L *lua.State, index int) {
	L.CheckStack(lua.LUA_MINSTACK)
	if index < 0 {
		index = L.GetTop() + index + 1
	}

	wb.markTable(L, index)

	if L.GetMetaField(index, "__pairs") != lua.LUA_TNIL {
		wb.packTableMetaPairs(L, index)
	} else {
		arraySize := wb.packTableArray(L, index)
		wb.packTableHash(L, index, arraySize)
	}
}

func (wb *writeBlock) packOne(L *lua.State, index int) {
	typ := L.Type(index)
	switch L.Type(index) {
	case lua.LUA_TNIL:
		wb.writeNil()
	case lua.LUA_TNUMBER:
		if L.IsInteger(index) {
			x := L.ToInteger(index)
			wb.writeInteger(x)
		} else {
			n := L.ToNumber(index)
			wb.writeReal(n)
		}
	case lua.LUA_TBOOLEAN:
		b := L.ToBoolean(index)
		wb.writeBoolean(b)
	case lua.LUA_TSTRING:
		str := L.ToString(index)
		wb.writeString(str)
	case lua.LUA_TLIGHTUSERDATA:
		udPtr := L.ToUserData(index)
		wb.writePointer(udPtr, serdeTypeUserDataPointer)
	case lua.LUA_TFUNCTION:
		fn := L.ToCFunction(index)
		if fn == nil || L.GetUpValue(index, 1) != "" {
			L.Errorf("Only light C function can be serialized")
		}
		wb.writePointer(fn, serdeTypeUserDataCFunc)
	case lua.LUA_TTABLE:
		if index < 0 {
			index = L.GetTop() + index + 1
		}
		if wb.refAncestor(L, index) {
			break
		}
		if wb.refObject(L, index) {
			break
		}
		if wb.stack.depth < maxDepth {
			wb.stack.ancestor = append(wb.stack.ancestor, index)
		}
		wb.stack.depth++
		wb.packTable(L, index)
		wb.stack.depth--
	default:
		L.Errorf("Unsupported type %s to serialize", L.TypeName(typ))
	}
}

func (wb *writeBlock) Bytes() []byte {
	data := wb.buf.Bytes()
	result := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(result[:4], uint32(len(data)))
	copy(result[4:], data)
	return result
}

func (wb *writeBlock) packFrom(L *lua.State, from int) {
	top := L.GetTop()
	n := top - from

	L.PushNil() // slot for table ref lookup
	L.PushNil() // slot for table refs array
	wb.stack.refIndex = top + 1

	for i := 1; i <= n; i++ {
		wb.packOne(L, from+i)
	}
}

type readBlock struct {
	buffer []byte
	len    int
	ptr    int
	stack  stack
}

func newReadBlock(buffer []byte) *readBlock {
	return &readBlock{
		buffer: buffer,
		len:    len(buffer),
		ptr:    0,
		stack:  stack{ancestor: make([]int, 0, maxDepth)},
	}
}

func (rb *readBlock) read(sz int) []byte {
	if rb.len < sz {
		return nil
	}

	ptr := rb.ptr
	rb.ptr += sz
	rb.len -= sz
	return rb.buffer[ptr : ptr+sz]
}

func (rb *readBlock) readByte() (byte, bool) {
	data := rb.read(1)
	if data == nil {
		return 0, false
	}
	return data[0], true
}

func (rb *readBlock) readUint16() (uint16, bool) {
	data := rb.read(2)
	if data == nil {
		return 0, false
	}
	return binary.LittleEndian.Uint16(data), true
}

func (rb *readBlock) readUint32() (uint32, bool) {
	data := rb.read(4)
	if data == nil {
		return 0, false
	}
	return binary.LittleEndian.Uint32(data), true
}

func (rb *readBlock) readUint64() (uint64, bool) {
	data := rb.read(8)
	if data == nil {
		return 0, false
	}
	return binary.LittleEndian.Uint64(data), true
}

func (rb *readBlock) readInt32() (int32, bool) {
	val, ok := rb.readUint32()
	return int32(val), ok
}

func (rb *readBlock) readInt64() (int64, bool) {
	data := rb.read(8)
	if data == nil {
		return 0, false
	}
	return int64(binary.LittleEndian.Uint64(data)), true
}

func (rb *readBlock) readFloat64() (float64, bool) {
	data := rb.read(8)
	if data == nil {
		return 0, false
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(data)), true
}

func (rb *readBlock) readPointer() (unsafe.Pointer, bool) {
	data, ok := rb.readUint64()
	if !ok {
		return nil, false
	}
	// Supress the "possible misuse of unsafe.Pointer" warning
	return *(*unsafe.Pointer)(unsafe.Pointer(&data)), true
}

func (rb *readBlock) getInteger(L *lua.State, cookie uint8) int64 {
	switch cookie {
	case serdeTypeNumberZero:
		return 0
	case serdeTypeNumberByte:
		if b, ok := rb.readByte(); ok {
			return int64(b)
		}
	case serdeTypeNumberWord:
		if w, ok := rb.readUint16(); ok {
			return int64(w)
		}
	case serdeTypeNumberDword:
		if d, ok := rb.readInt32(); ok {
			return int64(d)
		}
	case serdeTypeNumberQword:
		if q, ok := rb.readInt64(); ok {
			return q
		}
	}
	L.Errorf("Invalid serialize stream")
	return 0
}

func (rb *readBlock) getReal(L *lua.State) float64 {
	if f, ok := rb.readFloat64(); ok {
		return f
	}
	L.Errorf("Invalid serialize stream")
	return 0
}

func (rb *readBlock) getPointer(L *lua.State) unsafe.Pointer {
	if ptr, ok := rb.readPointer(); ok {
		return ptr
	}
	L.Errorf("Invalid serialize stream")
	return nil
}

func (rb *readBlock) getBuffer(L *lua.State, length int) string {
	data := rb.read(length)
	if data == nil {
		L.Errorf("Invalid serialize stream")
	}
	return string(data)
}

func (rb *readBlock) getExtendInteger(L *lua.State) int {
	typ, ok := rb.readByte()
	if !ok {
		L.Errorf("Invalid serialize stream")
	}

	cookie := typ >> 3
	if (typ&7) != serdeTypeNumber || cookie == serdeTypeNumberReal {
		L.Errorf("Invalid serialize stream")
	}

	return int(rb.getInteger(L, cookie))
}

func (rb *readBlock) unpackTable(L *lua.State, arraySize int, tableType uint8) {
	if arraySize == extendNumber {
		arraySize = rb.getExtendInteger(L)
	}

	rb.stack.objectID++
	id := rb.stack.objectID

	L.CheckStack(lua.LUA_MINSTACK)
	L.CreateTable(arraySize, 0)

	if tableType == serdeTypeTableMark {
		L.PushValue(-1)
		if L.Type(rb.stack.refIndex) == lua.LUA_TNIL {
			L.NewTable()
			L.Replace(rb.stack.refIndex)
		}
		L.RawSetI(rb.stack.refIndex, int64(id))
	}

	if rb.stack.depth < maxDepth {
		rb.stack.ancestor = append(rb.stack.ancestor, L.GetTop())
	}
	rb.stack.depth++

	for i := 1; i <= arraySize; i++ {
		rb.unpackOne(L)
		L.RawSetI(-2, int64(i))
	}

	rb.stack.depth--
	if rb.stack.depth < len(rb.stack.ancestor) {
		rb.stack.ancestor = rb.stack.ancestor[:rb.stack.depth]
	}

	for {
		rb.unpackOne(L)
		if L.IsNil(-1) {
			L.Pop(1)
			return
		}
		rb.stack.depth++
		rb.unpackOne(L)
		rb.stack.depth--
		L.RawSet(-3)
	}
}

func (rb *readBlock) unpackRef(L *lua.State, ref uint8) {
	if ref == extendNumber {
		id := rb.getExtendInteger(L)
		if L.Type(rb.stack.refIndex) != lua.LUA_TTABLE || L.RawGetI(rb.stack.refIndex, int64(id)) != lua.LUA_TTABLE {
			L.Errorf("Invalid ref object id %d", id)
		}
	} else {
		if int(ref) >= rb.stack.depth {
			L.Errorf("Invalid ref object %d/%d", ref, rb.stack.depth)
		}
		L.PushValue(rb.stack.ancestor[ref])
	}
}

func (rb *readBlock) pushValue(L *lua.State, typ, cookie uint8) {
	switch typ {
	case serdeTypeBoolean:
		switch cookie {
		case serdeTypeBooleanNil:
			L.PushNil()
		case serdeTypeBooleanFalse:
			L.PushBoolean(false)
		case serdeTypeBooleanTrue:
			L.PushBoolean(true)
		default:
			L.Errorf("Invalid boolean subtype %d", cookie)
		}
	case serdeTypeNumber:
		if cookie == serdeTypeNumberReal {
			L.PushNumber(rb.getReal(L))
		} else {
			L.PushInteger(rb.getInteger(L, cookie))
		}
	case serdeTypeUserData:
		switch cookie {
		case serdeTypeUserDataPointer:
			L.PushLightUserData(rb.getPointer(L))
		case serdeTypeUserDataCFunc:
			fn := rb.getPointer(L)
			L.PushCFunction(uintptr(fn))
		default:
			L.Errorf("Invalid userdata")
		}
	case serdeTypeShortString:
		str := rb.getBuffer(L, int(cookie))
		L.PushString(str)
	case serdeTypeLongString:
		var length int
		switch cookie {
		case 2:
			if l, ok := rb.readUint16(); ok {
				length = int(l)
			} else {
				L.Errorf("Invalid serialize stream")
			}
		case 4:
			if l, ok := rb.readUint32(); ok {
				length = int(l)
			} else {
				L.Errorf("Invalid serialize stream")
			}
		default:
			L.Errorf("Invalid serialize stream")
		}
		str := rb.getBuffer(L, length)
		L.PushString(str)
	case serdeTypeTable, serdeTypeTableMark:
		rb.unpackTable(L, int(cookie), typ)
	case serdeTypeRef:
		rb.unpackRef(L, cookie)
	default:
		L.Errorf("Invalid serialize stream")
	}
}

func (rb *readBlock) unpackOne(L *lua.State) {
	typ, ok := rb.readByte()
	if !ok {
		L.Errorf("Invalid serialize stream")
	}
	rb.pushValue(L, typ&0x7, typ>>3)
}

func serdePackString(content string, p unsafe.Pointer) []byte {
	wb := newWriteBlock()
	wb.writeString(content)
	if p != nil {
		wb.writePointer(p, serdeTypeUserDataPointer)
	}
	return wb.Bytes()
}

func serdeUnpack(L *lua.State, buffer []byte) int {
	top := L.GetTop()

	rb := newReadBlock(buffer)
	L.PushNil()
	rb.stack.refIndex = top + 1

	for i := 0; ; i++ {
		if i%8 == 0 {
			L.CheckStack(lua.LUA_MINSTACK)
		}

		typ, ok := rb.readByte()
		if !ok {
			break
		}
		rb.pushValue(L, typ&0x7, typ>>3)
	}

	return L.GetTop() - 1 - top
}

func serdePack(L *lua.State, from int) []byte {
	wb := newWriteBlock()
	wb.packFrom(L, from)

	return wb.Bytes()
}

func alignUp(n, align int) int {
	return (n + align - 1) &^ (align - 1)
}

func mallocFromBuffer(buf []byte) (ptr unsafe.Pointer, alignedSz int) {
	sz := len(buf)

	alignedSz = alignUp(sz, 8)
	ptr = malloc.Alloc(uint(alignedSz))
	buffer := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), alignedSz)
	copy(buffer, buf)
	return
}

var luaSerdePack = lua.NewCallback(LuaSerdePack)

func LuaSerdePack(L *lua.State) int {
	buf := serdePack(L, 0)
	ptr, alignedSz := mallocFromBuffer(buf)

	L.PushLightUserData(ptr)
	L.PushInteger(int64(alignedSz))
	return 2
}

var luaSerdeUnpack = lua.NewCallback(LuaSerdeUnpack)

func LuaSerdeUnpack(L *lua.State) int {
	if L.IsNoneOrNil(1) {
		return 0
	}
	L.SetTop(1)

	buffer := L.ToUserData(1)
	L.SetTop(0)

	length := binary.LittleEndian.Uint32(unsafe.Slice((*byte)(unsafe.Pointer(buffer)), 4))
	data := unsafe.Slice((*byte)(unsafe.Add(buffer, 4)), int(length))

	return serdeUnpack(L, data)
}

var luaSerdeUnpackRemove = lua.NewCallback(LuaSerdeUnpackRemove)

func LuaSerdeUnpackRemove(L *lua.State) int {
	if L.IsNoneOrNil(1) {
		return 0
	}

	L.SetTop(1)

	buffer := L.ToUserData(1)
	defer malloc.Free(buffer)

	L.SetTop(0)

	length := binary.LittleEndian.Uint32(unsafe.Slice((*byte)(unsafe.Pointer(buffer)), 4))
	data := unsafe.Slice((*byte)(unsafe.Add(buffer, 4)), int(length))

	return serdeUnpack(L, data)
}

var luaSerdeRemove = lua.NewCallback(LuaSerdeRemove)

func LuaSerdeRemove(L *lua.State) int {
	if L.IsNoneOrNil(1) {
		return 0
	}

	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	data := L.ToUserData(1)
	sz := L.CheckInteger(2)
	_ = sz

	malloc.Free(data)
	return 0
}

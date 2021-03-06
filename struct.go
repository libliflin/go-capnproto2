package capnp

// Struct is a pointer to a struct.
type Struct struct {
	seg   *Segment
	off   Address
	size  ObjectSize
	flags structFlags
}

// NewStruct creates a new struct, preferring placement in s.
func NewStruct(s *Segment, sz ObjectSize) (Struct, error) {
	if !sz.isValid() {
		return Struct{}, errObjectSize
	}
	sz.DataSize = sz.DataSize.padToWord()
	seg, addr, err := alloc(s, sz.totalSize())
	if err != nil {
		return Struct{}, err
	}
	return Struct{
		seg:  seg,
		off:  addr,
		size: sz,
	}, nil
}

// NewRootStruct creates a new struct, preferring placement in s, then sets the
// message's root to the new struct.
func NewRootStruct(s *Segment, sz ObjectSize) (Struct, error) {
	st, err := NewStruct(s, sz)
	if err != nil {
		return st, err
	}
	if err := s.msg.SetRoot(st); err != nil {
		return st, err
	}
	return st, nil
}

// ToStruct attempts to convert p into a struct.  If p is not a valid
// struct, then it returns an invalid Struct.
func ToStruct(p Pointer) Struct {
	if !IsValid(p) {
		return Struct{}
	}
	s, ok := p.underlying().(Struct)
	if !ok {
		return Struct{}
	}
	return s
}

// ToStructDefault attempts to convert p into a struct, reading the
// default value from def if p is not a struct.
func ToStructDefault(p Pointer, def []byte) (Struct, error) {
	fallback := func() (Struct, error) {
		if def == nil {
			return Struct{}, nil
		}
		defp, err := unmarshalDefault(def)
		if err != nil {
			return Struct{}, err
		}
		return ToStruct(defp), nil
	}
	if !IsValid(p) {
		return fallback()
	}
	s, ok := p.underlying().(Struct)
	if !ok {
		return fallback()
	}
	return s, nil
}

// Segment returns the segment this pointer came from.
func (p Struct) Segment() *Segment {
	return p.seg
}

// Address returns the address the pointer references.
func (p Struct) Address() Address {
	return p.off
}

// HasData reports whether the struct has a non-zero size.
func (p Struct) HasData() bool {
	return !p.size.isZero()
}

// value returns a raw struct pointer.
func (p Struct) value(paddr Address) rawPointer {
	off := makePointerOffset(paddr, p.off)
	return rawStructPointer(off, p.size)
}

func (p Struct) underlying() Pointer {
	return p
}

// Pointer returns the i'th pointer in the struct.
func (p Struct) Pointer(i uint16) (Pointer, error) {
	if p.seg == nil || i >= p.size.PointerCount {
		return nil, nil
	}
	return p.seg.readPtr(p.pointerAddress(i))
}

// SetPointer sets the i'th pointer in the struct to src.
func (p Struct) SetPointer(i uint16, src Pointer) error {
	if p.seg == nil || i >= p.size.PointerCount {
		panic(errOutOfBounds)
	}
	return p.seg.writePtr(copyContext{}, p.pointerAddress(i), src)
}

func (p Struct) pointerAddress(i uint16) Address {
	ptrStart := p.off.addSize(p.size.DataSize)
	return ptrStart.element(int32(i), wordSize)
}

// bitInData reports whether bit is inside p's data section.
func (p Struct) bitInData(bit BitOffset) bool {
	return p.seg != nil && bit < BitOffset(p.size.DataSize*8)
}

// Bit returns the bit that is n bits from the start of the struct.
func (p Struct) Bit(n BitOffset) bool {
	if !p.bitInData(n) {
		return false
	}
	addr := p.off.addOffset(n.offset())
	return p.seg.readUint8(addr)&n.mask() != 0
}

// SetBit sets the bit that is n bits from the start of the struct to v.
func (p Struct) SetBit(n BitOffset, v bool) {
	if !p.bitInData(n) {
		panic(errOutOfBounds)
	}
	addr := p.off.addOffset(n.offset())
	b := p.seg.readUint8(addr)
	if v {
		b |= n.mask()
	} else {
		b &^= n.mask()
	}
	p.seg.writeUint8(addr, b)
}

func (p Struct) dataAddress(off DataOffset, sz Size) (addr Address, ok bool) {
	if p.seg == nil || Size(off)+sz > p.size.DataSize {
		return 0, false
	}
	return p.off.addOffset(off), true
}

// Uint8 returns an 8-bit integer from the struct's data section.
func (p Struct) Uint8(off DataOffset) uint8 {
	addr, ok := p.dataAddress(off, 1)
	if !ok {
		return 0
	}
	return p.seg.readUint8(addr)
}

// Uint16 returns a 16-bit integer from the struct's data section.
func (p Struct) Uint16(off DataOffset) uint16 {
	addr, ok := p.dataAddress(off, 2)
	if !ok {
		return 0
	}
	return p.seg.readUint16(addr)
}

// Uint32 returns a 32-bit integer from the struct's data section.
func (p Struct) Uint32(off DataOffset) uint32 {
	addr, ok := p.dataAddress(off, 4)
	if !ok {
		return 0
	}
	return p.seg.readUint32(addr)
}

// Uint64 returns a 64-bit integer from the struct's data section.
func (p Struct) Uint64(off DataOffset) uint64 {
	addr, ok := p.dataAddress(off, 8)
	if !ok {
		return 0
	}
	return p.seg.readUint64(addr)
}

// SetUint8 sets the 8-bit integer that is off bytes from the start of the struct to v.
func (p Struct) SetUint8(off DataOffset, v uint8) {
	addr, ok := p.dataAddress(off, 1)
	if !ok {
		panic(errOutOfBounds)
	}
	p.seg.writeUint8(addr, v)
}

// SetUint16 sets the 16-bit integer that is off bytes from the start of the struct to v.
func (p Struct) SetUint16(off DataOffset, v uint16) {
	addr, ok := p.dataAddress(off, 2)
	if !ok {
		panic(errOutOfBounds)
	}
	p.seg.writeUint16(addr, v)
}

// SetUint32 sets the 32-bit integer that is off bytes from the start of the struct to v.
func (p Struct) SetUint32(off DataOffset, v uint32) {
	addr, ok := p.dataAddress(off, 4)
	if !ok {
		panic(errOutOfBounds)
	}
	p.seg.writeUint32(addr, v)
}

// SetUint64 sets the 64-bit integer that is off bytes from the start of the struct to v.
func (p Struct) SetUint64(off DataOffset, v uint64) {
	addr, ok := p.dataAddress(off, 8)
	if !ok {
		panic(errOutOfBounds)
	}
	p.seg.writeUint64(addr, v)
}

// structFlags is a bitmask of flags for a pointer.
type structFlags uint8

// Pointer flags.
const (
	isListMember structFlags = 1 << iota
)

// copyStruct makes a deep copy of src into dst.
func copyStruct(cc copyContext, dst, src Struct) error {
	if dst.seg == nil {
		return nil
	}

	// Q: how does version handling happen here, when the
	//    destination toData[] slice can be bigger or smaller
	//    than the source data slice, which is in
	//    src.seg.Data[src.off:src.off+src.size.DataSize] ?
	//
	// A: Newer fields only come *after* old fields. Note that
	//    copy only copies min(len(src), len(dst)) size,
	//    and then we manually zero the rest in the for loop
	//    that writes toData[j] = 0.
	//

	// data section:
	srcData := src.seg.slice(src.off, src.size.DataSize)
	dstData := dst.seg.slice(dst.off, dst.size.DataSize)
	copyCount := copy(dstData, srcData)
	dstData = dstData[copyCount:]
	for j := range dstData {
		dstData[j] = 0
	}

	// ptrs section:

	// version handling: we ignore any extra-newer-pointers in src,
	// i.e. the case when srcPtrSize > dstPtrSize, by only
	// running j over the size of dstPtrSize, the destination size.
	srcPtrSect := src.off.addSize(src.size.DataSize)
	dstPtrSect := dst.off.addSize(dst.size.DataSize)
	numSrcPtrs := src.size.PointerCount
	numDstPtrs := dst.size.PointerCount
	for j := uint16(0); j < numSrcPtrs && j < numDstPtrs; j++ {
		srcAddr := srcPtrSect.element(int32(j), wordSize)
		dstAddr := dstPtrSect.element(int32(j), wordSize)
		m, err := src.seg.readPtr(srcAddr)
		if err != nil {
			return err
		}
		err = dst.seg.writePtr(cc.incDepth(), dstAddr, m)
		if err != nil {
			return err
		}
	}
	for j := numSrcPtrs; j < numDstPtrs; j++ {
		// destination p is a newer version than source so these extra new pointer fields in p must be zeroed.
		addr := dstPtrSect.element(int32(j), wordSize)
		dst.seg.writeRawPointer(addr, 0)
	}
	// Nothing more here: so any other pointers in srcPtrSize beyond
	// those in dstPtrSize are ignored and discarded.

	return nil
}

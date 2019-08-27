package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go.uber.org/zap"
	"io"
)

// http://dwarfstd.org/doc/Dwarf3.pdf
type CommonInformationEntry struct {
	length uint32
	cie_id uint64
	version byte
	augmentation string
	code_alignment_factor uint64
	data_alignment_factor int64
	return_address_register uint64
	initial_instructions []byte
	// `padding`, Enough DW_CFA_nop instructions to make the size of this entry match the length value above.
}

func (cie *CommonInformationEntry) String() string {
	if cie == nil {
		return "`cie == nil` is true, please check cie"
	}
	return fmt.Sprintf("len=%d, cie_id=0x%x, " +
		"versiont=%v, augmentation=%s, code_alignment_factor=%d, " +
		"data_alignment_factor=%d, return_address_register=0x%x, " +
		"initial_instructions=%v",
	cie.length,
	cie.cie_id, cie.version, cie.augmentation,
	cie.code_alignment_factor, cie.data_alignment_factor, cie.return_address_register,
	cie.initial_instructions)
}

type FrameDescriptionEntry struct {
	length       uint32
	CIE          *CommonInformationEntry
	instructions []byte
	begin, size  uint64
}

func (fde *FrameDescriptionEntry) String() string {
	if fde == nil {
		return "`fde == nil` is true, please check fde"
	}
	return fmt.Sprintf("len=%d, instructions=%v, begin=0x%x, size=%d",
		fde.length, fde.instructions, fde.begin, fde.size)
}

type VirtualUnwindFrameInformation struct {
	len uint32
	CIE *CommonInformationEntry
	FDE *FrameDescriptionEntry
}

type Frame struct {
	instructions []byte
	address uint64
	cie *CommonInformationEntry
	Offset int64
	cfa *DWRule
	regsRule map[uint64]DWRule
	regs []uint64
	framebase uint64
	loc uint64
}

func parseFrameInformation(buffer *bytes.Buffer) (*VirtualUnwindFrameInformation, error) {
	if buffer.Len() == 0 {
		return nil, io.EOF
	}
	var (
		cieEntry *CommonInformationEntry
		fdeEntry *FrameDescriptionEntry
		err error
		info *VirtualUnwindFrameInformation
	)

	info = &VirtualUnwindFrameInformation{}
	binary.Read(buffer, binary.LittleEndian, &info.len)


	tbytes := buffer.Next(4)
	info.len -= 4

	input := buffer.Next(int(info.len))

	if bytes.Equal(tbytes, []byte{0xff, 0xff, 0xff, 0xff}) {
		// cie
		if cieEntry, err = parseCommonInformationEntryByte(info.len, input); err != nil {
			return nil, err
		}
		info.CIE = cieEntry
	} else {
		// fde
		if fdeEntry, err = parseFrameDescriptionEntryByte(info.len, input); err != nil {
			return nil, err
		}
		info.FDE = fdeEntry
	}
	return info, nil
}

// https://en.wikipedia.org/wiki/LEB128
// result = 0;
// shift = 0;
// while(true) {
// 	 byte = next byte in input;
// 	 result |= (low order 7 bits of byte) << shift;
//	 if (high order bit of byte == 0)
//	 break;
//	 shift += 7;
// }
func DecodeULEB128(buf *bytes.Buffer) (uint64, uint32, error){
	var (
		res uint64
		len uint32
		err error
		b byte
		shift uint64
	)
	for {
		if b, err = buf.ReadByte(); err != nil {
			return 0, 0, err
		}
		len++

		res |= uint64((uint(b) & 0x7f) << shift)

		// If high order bit is 1.
		if b&0x80 == 0 {
			break
		}

		shift += 7
	}

	return res, len, nil
}
// https://en.wikipedia.org/wiki/LEB128
// result = 0;
// shift = 0;
// size = number of bits in signed integer;
// do{
// 	 byte = next byte in input;
//   result |= (low order 7 bits of byte << shift);
//   shift += 7;
// } while (high order bit of byte != 0);
//
// /* sign bit of byte is second high order bit (0x40) */
// if ((shift <size) && (sign bit of byte is set))
// /* sign extend */
//   result |= (~0 << shift);
func DecodeSLEB128(buf *bytes.Buffer) (int64, uint32, error) {
	var (
		res int64
		len uint32
		err error
		b byte
		shift uint64
	)

	for {
		b, err = buf.ReadByte()
		if err != nil {
			return 0, 0, err
		}
		len++

		res |= int64((int64(b) & 0x7f) << shift)
		shift += 7
		if b&0x80 == 0 {
			break
		}
	}

	if (shift < 8*uint64(len)) && (b&0x40 > 0) {
		res |= -(1 << shift)
	}

	return res, len, nil
}

func parseCommonInformationEntryByte(len uint32, data []byte) (*CommonInformationEntry, error){
	var (
		err error
		str string
	)

	cie := &CommonInformationEntry{}
	buf := bytes.NewBuffer(data)

	cie.length = len
	cie.cie_id = 0xffffffff
	cie.version = buf.Next(1)[0]

	// A null-terminated UTF-8 string that identifies the augmentation to this CIE or to the FDEs that use it.
	if str, err = buf.ReadString(0x0); err != nil {
		return nil, err
	}
	cie.augmentation = str

	// code_alignment_factor (unsigned LEB128)
	if cie.code_alignment_factor, _, err = DecodeULEB128(buf); err != nil {
		return nil, err
	}

	// data_alignment_factor (signed LEB128)
	if cie.data_alignment_factor, _, err = DecodeSLEB128(buf); err != nil {
		return nil, err
	}

	// return_address_register (unsigned LEB128)
	if cie.return_address_register, _, err = DecodeULEB128(buf); err != nil {
		return nil, err
	}

	// A sequence of rules that are interpreted to create the initial setting of each column in the  table.
	cie.initial_instructions = buf.Bytes()

	// padding (array of ubyte)
	// Enough DW_CFA_nop instructions to make the size of this entry match the length value  above.
	return cie, nil
}

// ???
func parseFrameDescriptionEntryByte(len uint32, data []byte) (*FrameDescriptionEntry, error) {
	fde := &FrameDescriptionEntry{}

	fde.length = len
	fde.begin = binary.LittleEndian.Uint64(data[:8]) // + ctx.staticBase
	fde.size = binary.LittleEndian.Uint64(data[8:16])
	fde.instructions = data[16:]

	logger.Debug("parseFrameDescriptionEntryByte",
		zap.Uint32("len", len),
		zap.Uint64("begin", fde.begin),
		zap.Uint64("size", fde.size))

	return fde, nil
}
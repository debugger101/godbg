package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go.uber.org/zap"
)

const low_6_offset = 0x3f
const high_2_bits = 0xc0

// https://golang.org/pkg/cmd/internal/dwarf/
const (
	// operand,...
	DW_CFA_nop              = 0x00
	DW_CFA_set_loc          = 0x01 // address
	DW_CFA_advance_loc1     = 0x02 // 1-byte delta
	DW_CFA_advance_loc2     = 0x03 // 2-byte delta
	DW_CFA_advance_loc4     = 0x04 // 4-byte delta
	DW_CFA_offset_extended  = 0x05 // ULEB128 register, ULEB128 offset
	DW_CFA_restore_extended = 0x06 // ULEB128 register
	DW_CFA_undefined        = 0x07 // ULEB128 register
	DW_CFA_same_value       = 0x08 // ULEB128 register
	DW_CFA_register         = 0x09 // ULEB128 register, ULEB128 register
	DW_CFA_remember_state   = 0x0a
	DW_CFA_restore_state    = 0x0b

	DW_CFA_def_cfa            = 0x0c // ULEB128 register, ULEB128 offset
	DW_CFA_def_cfa_register   = 0x0d // ULEB128 register
	DW_CFA_def_cfa_offset     = 0x0e // ULEB128 offset
	DW_CFA_def_cfa_expression = 0x0f // BLOCK
	DW_CFA_expression         = 0x10 // ULEB128 register, BLOCK
	DW_CFA_offset_extended_sf = 0x11 // ULEB128 register, SLEB128 offset
	DW_CFA_def_cfa_sf         = 0x12 // ULEB128 register, SLEB128 offset
	DW_CFA_def_cfa_offset_sf  = 0x13 // SLEB128 offset
	DW_CFA_val_offset         = 0x14 // ULEB128, ULEB128
	DW_CFA_val_offset_sf      = 0x15 // ULEB128, SLEB128
	DW_CFA_val_expression     = 0x16 // ULEB128, BLOCK

	DW_CFA_lo_user = 0x1c
	DW_CFA_hi_user = 0x3f

	// Opcodes that take an addend operand.
	DW_CFA_advance_loc = 0x1 << 6 // +delta
	DW_CFA_offset      = 0x2 << 6 // +register (ULEB128 offset)
	DW_CFA_restore     = 0x3 << 6 // +register
)

// Rules defined for register values.
type Rule byte

const (
	RuleUndefined Rule = iota
	RuleSameVal
	RuleOffset
	RuleValOffset
	RuleRegister
	RuleExpression
	RuleValExpression
	RuleArchitectural
	RuleCFA          // Value is rule.Reg + rule.Offset
	RuleFramePointer // Value is stored at address rule.Reg + rule.Offset, but only if it's less than the current CFA, otherwise same value
)

type DWRule struct {
	offset int64
	reg    uint64
	rule   Rule
}

func execSingleInstruction(frame *Frame, buf *bytes.Buffer) error {
	var (
		byte byte
		err  error
	)

	if byte, err = buf.ReadByte(); err != nil {
		return err
	}
	switch byte & high_2_bits {
	case DW_CFA_advance_loc:
		byte = DW_CFA_advance_loc
		if err = buf.UnreadByte(); err != nil {
			return err
		}
	case DW_CFA_offset:
		byte = DW_CFA_offset
		if err = buf.UnreadByte(); err != nil {
			return err
		}
	case DW_CFA_restore:
		byte = DW_CFA_restore
		if err = buf.UnreadByte(); err != nil {
			return err
		}
	}

	switch byte {
	case DW_CFA_offset_extended:
		reg, _, _ := DecodeULEB128(buf)
		offset, _, _ := DecodeULEB128(buf)
		frame.regsRule[reg] = DWRule{offset: int64(offset) * frame.cie.data_alignment_factor, rule: RuleOffset}
		logger.Debug(fmt.Sprintf("DW_CFA_offset_extended, reg %d, offset %d, dwrule.offset %d\n", reg, offset, frame.regsRule[reg].offset))
	case DW_CFA_def_cfa:
		frame.cfa.reg, _, _ = DecodeULEB128(buf)
		offset, _, _ := DecodeULEB128(buf)
		frame.cfa.offset = int64(offset)
		frame.cfa.rule = RuleCFA
		logger.Debug(fmt.Sprintf("DW_CFA_def_cfa, reg %d, offset %d\n", frame.cfa.reg, frame.cfa.offset))
	case DW_CFA_def_cfa_register:
		reg, _, _ := DecodeULEB128(buf)
		frame.cfa.reg = reg
		frame.cfa.rule = RuleUndefined
		logger.Debug(fmt.Sprintf("DW_CFA_def_cfa_register, cfa.reg %d, cfa.offset %d\n", reg, frame.cfa.offset))
	case DW_CFA_def_cfa_offset_sf:
		offset, _, _ := DecodeSLEB128(buf)
		t := offset
		offset *= frame.cie.data_alignment_factor
		frame.cfa.offset = offset
		logger.Debug(fmt.Sprintf("DW_CFA_def_cfa_offset_sf, offset *= frame.data_aligment_factor, %d = %d * %d\n",
			offset, t, frame.cie.data_alignment_factor))
	case DW_CFA_advance_loc:
		if byte, err = buf.ReadByte(); err != nil {
			return err
		}
		delta := byte & low_6_offset
		frame.loc += uint64(delta) * frame.cie.code_alignment_factor
		logger.Debug(fmt.Sprintf("DW_CFA_advance_loc, delta %d, frame.loc=%d\n", uint64(delta), frame.loc))
	case DW_CFA_advance_loc1:
		delta, err := buf.ReadByte()
		if err != nil {
			return err
		}
		frame.loc += uint64(delta) * frame.cie.code_alignment_factor
		logger.Debug(fmt.Sprintf("DW_CFA_advance_loc1, delta %d, frame.loc=%d\n", uint64(delta), frame.loc))
	case DW_CFA_advance_loc2:
		var delta uint16
		binary.Read(buf, binary.LittleEndian, &delta)
		frame.loc += uint64(delta) * frame.cie.code_alignment_factor
		logger.Debug(fmt.Sprintf("DW_CFA_advance_loc2, delta %d, frame.loc=%d\n", uint64(delta), frame.loc))
	case DW_CFA_restore:
		if byte, err = buf.ReadByte(); err != nil {
			return err
		}
		reg := uint64(byte & low_6_offset)
		frame.regsRule[reg] = DWRule{rule: RuleUndefined}
	case DW_CFA_nop:
		return nil
	default:
		logger.Error("execInstructions unknown byte", zap.Uint8("DW_CFA", byte))
		return fmt.Errorf("execInstructions unknown byte %v", byte)
	}
	return nil
}

func execCIEInstructions(frame *Frame, buf *bytes.Buffer) error {
	for buf.Len() > 0 {
		if err := execSingleInstruction(frame, buf); err != nil {
			return err
		}
	}
	return nil
}

func execFDEInstructions(frame *Frame, buf *bytes.Buffer) error {
	for frame.address >= frame.loc && buf.Len() > 0 {
		if err := execSingleInstruction(frame, buf); err != nil {
			return err
		}
	}
	return nil
}

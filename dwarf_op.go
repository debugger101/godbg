package main

// This table is produced by go compiler and linker. copy from https://golang.org/pkg/cmd/internal/dwarf/
const (
	DW_OP_addr                = 0x03 // 1 constant address (size target specific)
	DW_OP_deref               = 0x06 // 0
	DW_OP_const1u             = 0x08 // 1 1-byte constant
	DW_OP_const1s             = 0x09 // 1 1-byte constant
	DW_OP_const2u             = 0x0a // 1 2-byte constant
	DW_OP_const2s             = 0x0b // 1 2-byte constant
	DW_OP_const4u             = 0x0c // 1 4-byte constant
	DW_OP_const4s             = 0x0d // 1 4-byte constant
	DW_OP_const8u             = 0x0e // 1 8-byte constant
	DW_OP_const8s             = 0x0f // 1 8-byte constant
	DW_OP_constu              = 0x10 // 1 ULEB128 constant
	DW_OP_consts              = 0x11 // 1 SLEB128 constant
	DW_OP_dup                 = 0x12 // 0
	DW_OP_drop                = 0x13 // 0
	DW_OP_over                = 0x14 // 0
	DW_OP_pick                = 0x15 // 1 1-byte stack index
	DW_OP_swap                = 0x16 // 0
	DW_OP_rot                 = 0x17 // 0
	DW_OP_xderef              = 0x18 // 0
	DW_OP_abs                 = 0x19 // 0
	DW_OP_and                 = 0x1a // 0
	DW_OP_div                 = 0x1b // 0
	DW_OP_minus               = 0x1c // 0
	DW_OP_mod                 = 0x1d // 0
	DW_OP_mul                 = 0x1e // 0
	DW_OP_neg                 = 0x1f // 0
	DW_OP_not                 = 0x20 // 0
	DW_OP_or                  = 0x21 // 0
	DW_OP_plus                = 0x22 // 0
	DW_OP_plus_uconst         = 0x23 // 1 ULEB128 addend
	DW_OP_shl                 = 0x24 // 0
	DW_OP_shr                 = 0x25 // 0
	DW_OP_shra                = 0x26 // 0
	DW_OP_xor                 = 0x27 // 0
	DW_OP_skip                = 0x2f // 1 signed 2-byte constant
	DW_OP_bra                 = 0x28 // 1 signed 2-byte constant
	DW_OP_eq                  = 0x29 // 0
	DW_OP_ge                  = 0x2a // 0
	DW_OP_gt                  = 0x2b // 0
	DW_OP_le                  = 0x2c // 0
	DW_OP_lt                  = 0x2d // 0
	DW_OP_ne                  = 0x2e // 0
	DW_OP_lit0                = 0x30 // 0 ...
	DW_OP_lit31               = 0x4f // 0 literals 0..31 = (DW_OP_lit0 + literal)
	DW_OP_reg0                = 0x50 // 0 ..
	DW_OP_reg31               = 0x6f // 0 reg 0..31 = (DW_OP_reg0 + regnum)
	DW_OP_breg0               = 0x70 // 1 ...
	DW_OP_breg31              = 0x8f // 1 SLEB128 offset base register 0..31 = (DW_OP_breg0 + regnum)
	DW_OP_regx                = 0x90 // 1 ULEB128 register
	DW_OP_fbreg               = 0x91 // 1 SLEB128 offset
	DW_OP_bregx               = 0x92 // 2 ULEB128 register followed by SLEB128 offset
	DW_OP_piece               = 0x93 // 1 ULEB128 size of piece addressed
	DW_OP_deref_size          = 0x94 // 1 1-byte size of data retrieved
	DW_OP_xderef_size         = 0x95 // 1 1-byte size of data retrieved
	DW_OP_nop                 = 0x96 // 0
	DW_OP_push_object_address = 0x97 // 0
	DW_OP_call2               = 0x98 // 1 2-byte offset of DIE
	DW_OP_call4               = 0x99 // 1 4-byte offset of DIE
	DW_OP_call_ref            = 0x9a // 1 4- or 8-byte offset of DIE
	DW_OP_form_tls_address    = 0x9b // 0
	DW_OP_call_frame_cfa      = 0x9c // 0
	DW_OP_bit_piece           = 0x9d // 2
	DW_OP_lo_user             = 0xe0
	DW_OP_hi_user             = 0xff
)

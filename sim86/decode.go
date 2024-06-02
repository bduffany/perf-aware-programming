package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var (
	execute = flag.Bool("exec", false, "Execute instructions")
	name    = flag.String("name", "", "File name hint (for file on stdin)")
)

var debugMode = os.Getenv("DEBUG") == "1"

func debugf(format string, args ...any) {
	if debugMode {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

type debugByteReader struct {
	io.ByteReader
	i int
}

func (r *debugByteReader) ReadByte() (byte, error) {
	b, err := r.ByteReader.ReadByte()
	if err == nil {
		debugf("\x1b[30mbyte %d: %08b\x1b[m", r.i+1, b)
	}
	r.i++
	return b, err
}

func (r *debugByteReader) ResetIndex() {
	r.i = 0
}

func main() {
	flag.Parse()

	var process func(instruction *Instruction) error
	if *execute {
		var sim Sim
		fmt.Printf("--- %s execution ---\n", *name)
		process = func(instruction *Instruction) error {
			return sim.Execute(instruction)
		}
		defer func() {
			sim.PrintRegisters()
		}()
	} else {
		// Print decoded output
		fmt.Printf("bits 16\n")
		process = func(instruction *Instruction) error {
			fmt.Printf("%s %s\n", instruction.Op, strings.Join(instruction.Args, ", "))
			return nil
		}
	}

	r := io.ByteReader(bufio.NewReader(os.Stdin))
	if debugMode {
		r = &debugByteReader{r, 0}
	}
	for {
		instruction, err := decodeInstruction(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if debugMode {
			r.(*debugByteReader).ResetIndex()
		}
		debugf("\x1b[33m%s %s\x1b[m", instruction.Op, strings.Join(instruction.Args, ", "))
		if err := process(instruction); err != nil {
			log.Fatal(err)
		}
	}
}

func decodeInstruction(r io.ByteReader) (*Instruction, error) {
	debugf("---")

	byte1, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	var op string
	var modRegRM, modXXXRM, arithmetic, immediate, jmp bool
	var d, w, s, mod, reg, rm, arithmeticOp byte

	switch true {
	case byte1&0b11111100 == 0b10001000:
		debugf("mov reg <-> {mem|reg}")
		op = "mov"
		modRegRM = true

	case byte1&0b11110000 == 0b10110000:
		debugf("mov reg <- imm")
		op = "mov"
		immediate = true

	case byte1&0b11000100 == 0b00000000:
		debugf("<arithmetic> reg <-> {mem|reg}")
		modRegRM = true
		arithmetic = true
		arithmeticOp = (byte1 & 0b00111000) >> 3

	case byte1&0b11111100 == 0b10000000:
		debugf("<arithmetic> {reg|mem} <- imm")
		modXXXRM = true
		immediate = true
		arithmetic = true

	case byte1&0b11000100 == 0b00000100:
		debugf("<?arithmetic> reg <- imm")
		immediate = true
		arithmetic = true
		arithmeticOp = (byte1 & 0b00111000) >> 3

	case byte1 == 0b0111_0100:
		op = "je"
		jmp = true
	case byte1 == 0b0111_1100:
		op = "jl"
		jmp = true
	case byte1 == 0b0111_1110:
		op = "jle"
		jmp = true
	case byte1 == 0b0111_0010:
		op = "jb"
		jmp = true
	case byte1 == 0b0111_0110:
		op = "jbe"
		jmp = true
	case byte1 == 0b0111_1010:
		op = "jp"
		jmp = true
	case byte1 == 0b0111_0000:
		op = "jo"
		jmp = true
	case byte1 == 0b0111_1000:
		op = "js"
		jmp = true
	case byte1 == 0b0111_0101:
		op = "jne"
		jmp = true
	case byte1 == 0b0111_1101:
		op = "jnl"
		jmp = true
	case byte1 == 0b0111_1111:
		op = "jnle"
		jmp = true
	case byte1 == 0b0111_0011:
		op = "jnb"
		jmp = true
	case byte1 == 0b0111_0111:
		op = "jnbe"
		jmp = true
	case byte1 == 0b0111_1011:
		op = "jnp"
		jmp = true
	case byte1 == 0b0111_0001:
		op = "jno"
		jmp = true
	case byte1 == 0b0111_1001:
		op = "jns"
		jmp = true
	case byte1 == 0b1110_0010:
		op = "loop"
		jmp = true
	case byte1 == 0b1110_0001:
		op = "loopz"
		jmp = true
	case byte1 == 0b1110_0000:
		op = "loopnz"
		jmp = true
	case byte1 == 0b1110_0011:
		op = "jcxz"
		jmp = true

	default:
		return nil, fmt.Errorf("unhandled instruction [%08b]", byte1)
	}

	if jmp {
		debugf("%s <ip-inc8>", op)
		var ipInc8 byte
		ipInc8, err = r.ReadByte()
		if err != nil {
			return nil, err
		}
		var arg string
		if offset := int8(ipInc8) + 2; offset == 0 {
			arg = "$+0"
		} else if offset > 0 {
			arg = fmt.Sprintf("$+%d+0", offset)
		} else {
			arg = fmt.Sprintf("$%d+0", offset)
		}
		return &Instruction{Op: op, Args: []string{arg}}, nil
	}

	if modRegRM {
		d = (byte1 & 0b00000010) >> 1
		w = (byte1 & 0b00000001) >> 0
		debugf("[d w] = [%b %b]", d, w)
	}
	if !modRegRM && !modXXXRM && !arithmetic {
		w = (byte1 & 0b00001000) >> 3
		// REG bits come from byte 1
		reg = (byte1 & 0b00000111) >> 0
		debugf("[w reg] = [%b %03b]", w, reg)
	}
	if arithmetic && modXXXRM {
		s = (byte1 & 0b00000010) >> 1
		w = (byte1 & 0b00000001) >> 0
		debugf("[s w] = [%b %b]", s, w)
	}
	if arithmetic && immediate && !modRegRM && !modXXXRM {
		w = (byte1 & 0b00000001) >> 0
		reg = 0b000 // accumulator (AL / AX)
	}

	if modRegRM || modXXXRM {
		// Read second byte and interpret as [ MOD | {REG or 000} | R/M ]
		var byte2 byte
		byte2, err = r.ReadByte()
		if err != nil {
			return nil, err
		}
		mod = (byte2 & 0b11000000) >> 6
		if modRegRM {
			reg = (byte2 & 0b00111000) >> 3
			debugf("[mod reg rm] = [%02b %03b %03b]", mod, reg, rm)
		} else if modXXXRM && arithmetic {
			arithmeticOp = (byte2 & 0b00111000) >> 3
			debugf("[mod op rm] = [%02b %03b %03b]", mod, arithmeticOp, rm)
		}
		rm = (byte2 & 0b00000111) >> 0
	}

	if arithmetic {
		switch arithmeticOp {
		case 0b000:
			op = "add"
		case 0b101:
			op = "sub"
		case 0b111:
			op = "cmp"
		default:
			err = fmt.Errorf("unhandled arithmetic op %03b", arithmeticOp)
		}
	}

	// Parse REG name from REG and W fields
	regName := decodeRegName(reg, w)

	// Handle MOD and R/M
	var memoryMode, directAddress bool
	var disp uint16
	var displacementBits uint8
	if modRegRM || modXXXRM {
		switch mod {
		case 0b00:
			// Memory mode, no displacement, unless R/M = 0b110 (direct address)
			memoryMode = true
			if rm == 0b110 {
				displacementBits = 16
				directAddress = true
				debugf("mod: mem; direct address (16-bit)")
			} else {
				debugf("mod: mem; no displacement")
			}
		case 0b01:
			// Memory mode, 8-bit displacement follows
			memoryMode = true
			displacementBits = 8
			debugf("mod: mem; 8-bit")
		case 0b10:
			// Memory mode: 16-bit displacement follows
			memoryMode = true
			displacementBits = 16
			debugf("mod: mem; 16-bit")
		case 0b11:
			// Register mode, no displacement
			memoryMode = false
			displacementBits = 0
			debugf("mod: reg")
		default:
			panic("non-exhaustive switch")
		}
	}

	// Read displacement bytes
	var dispLO, dispHI byte
	if displacementBits >= 8 {
		dispLO, err = r.ReadByte()
		if err != nil {
			return nil, err
		}
		disp += uint16(dispLO)
	}
	if displacementBits == 16 {
		dispHI, err = r.ReadByte()
		if err != nil {
			return nil, err
		}
		disp += uint16(dispHI) << 8
	}

	// Read data bytes (immediate)
	var dataLO, dataHI byte
	var data uint16
	var dataBits uint8
	if immediate {
		dataBits = 8
		dataLO, err = r.ReadByte()
		if err != nil {
			return nil, err
		}
		data += uint16(dataLO)

		if w == 1 {
			dataBits = 16
		}

		if s == 0 && w == 1 {
			dataHI, err = r.ReadByte()
			if err != nil {
				return nil, err
			}
			data += uint16(dataHI) << 8
		} else if s == 1 {
			// Sign-extend data: set HI bits to all 1s if the leftmost LO bit is a 1
			if dataLO&0b10000000 == 0b10000000 {
				data |= uint16(0xFF00)
			}
		}
	}

	var rmArg string
	if directAddress {
		rmArg = fmt.Sprintf("[%d]", disp)
	} else if memoryMode {
		addr := decodeRM(rm)
		if displacementBits > 0 {
			addr += fmt.Sprintf(" + %d", disp)
		}
		rmArg = "[" + addr + "]"
	} else { // register mode
		rmArg = decodeRegName(rm, w)
	}

	var src, dst string
	if immediate {
		src = fmt.Sprint(data)
		if modRegRM || modXXXRM {
			dst = rmArg
			if memoryMode {
				if dataBits == 8 {
					dst = "byte " + dst
				} else {
					dst = "word " + dst
				}
			}
		} else {
			dst = regName
		}
	} else if d == 0 {
		src = regName
		dst = rmArg
	} else {
		src = rmArg
		dst = regName
	}

	return &Instruction{Op: op, Args: []string{dst, src}}, nil
}

func decodeRM(rm byte) string {
	switch rm {
	case 0b000:
		return "bx + si"
	case 0b001:
		return "bx + di"
	case 0b010:
		return "bp + si"
	case 0b011:
		return "bp + di"
	case 0b100:
		return "si"
	case 0b101:
		return "di"
	case 0b110:
		// NOTE: direct address is handled in special case above
		return "bp"
	case 0b111:
		return "bx"
	default:
		panic("invalid rm bits")
	}
}

func decodeRegName(reg, w byte) string {
	if w == 0 {
		switch reg {
		case 0b000:
			return "al"
		case 0b001:
			return "cl"
		case 0b010:
			return "dl"
		case 0b011:
			return "bl"
		case 0b100:
			return "ah"
		case 0b101:
			return "ch"
		case 0b110:
			return "dh"
		case 0b111:
			return "bh"
		default:
			panic("non-exhaustive switch")
		}
	} else {
		switch reg {
		case 0b000:
			return "ax"
		case 0b001:
			return "cx"
		case 0b010:
			return "dx"
		case 0b011:
			return "bx"
		case 0b100:
			return "sp"
		case 0b101:
			return "bp"
		case 0b110:
			return "si"
		case 0b111:
			return "di"
		default:
			panic("non-exhaustive switch")
		}
	}
}

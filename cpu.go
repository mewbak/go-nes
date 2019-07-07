package main

import (
	"bufio"
	"github.com/veandco/go-sdl2/sdl"
	"math"
	"os"
)

type Cpu struct {
	RAM      []int
	PPU      *PPU
	APU      []int
	Register *Register
	PrgROM   []byte
	ChrROM   []byte
	GamePad  *GamePad
	GamePadIndex int
	interrupts *Interrupts
}

func NewCpu(prgRom []byte) *Cpu {
	interrupts := &Interrupts{}
	cpu := &Cpu{
		RAM: make([]int, 0x0800),
		Register: &Register{
			P: &StatusRegister{},
		},
		PrgROM: prgRom,
		GamePad: &GamePad{},
		PPU: NewPPU(interrupts),
		interrupts: interrupts,
	}
	cpu.Reset()
	return cpu
}

func (cpu *Cpu) Write(index int, value int) {
	if index < 0x0800 {
		cpu.RAM[index] = value
	} else if index < 0x2000 {
		cpu.RAM[index-0x0800] = value
	} else if index < 0x2008 {
		cpu.PPU.Write(index-0x2000, value)
	} else if index < 0x4000 {

	} else if index == 0x4014 {
		// sprite DMA transfer
		addr := value << 8
		for i := 0; i < 0xFF; i+=4 {
			base := i+addr
			cpu.PPU.spriteRAM[i]   = cpu.RAM[base]
			cpu.PPU.spriteRAM[i+1] = cpu.RAM[base+1]
			cpu.PPU.spriteRAM[i+2] = cpu.RAM[base+2]
			cpu.PPU.spriteRAM[i+3] = cpu.RAM[base+3]
		}
	} else if index == 0x4016 {
		// key input
		// TODO: impl
		if value == 0 {
			cpu.GamePadIndex = 0
			cpu.GamePad.Reset()
		}
	} else if index < 0x4020 {

	} else if index < 0x6000 {

	} else if index < 0x8000 {

	} else {
		if len(cpu.PrgROM) == 0x8000 {
			cpu.PrgROM[index-0x8000] = byte(value)
		} else {
			cpu.PrgROM[index-0xC000] = byte(value)
		}
	}
}

func (cpu *Cpu) Read(index int) int {
	if index < 0x0800 {
		return cpu.RAM[index]
	}
	if index < 0x2000 {
		return cpu.RAM[index-0x800]
	}
	if index < 0x2008 {
		return cpu.PPU.Read(index - 0x2000)
	}
	if index < 0x4000 {

	}
	if index == 0x4016 {
		// key input
		sdl.PumpEvents()
		states := sdl.GetKeyboardState()
		var result int
		switch cpu.GamePadIndex {
		case 0:
			result = bool2int(states[sdl.SCANCODE_Z] == 1)
		case 1:
			result = bool2int(states[sdl.SCANCODE_X] == 1)
		case 2:
			result = bool2int(states[sdl.SCANCODE_S] == 1)
		case 3:
			result = bool2int(states[sdl.SCANCODE_D] == 1)
		case 4:
			result = bool2int(states[sdl.SCANCODE_UP] == 1)
		case 5:
			result = bool2int(states[sdl.SCANCODE_DOWN] == 1)
		case 6:
			result = bool2int(states[sdl.SCANCODE_LEFT] == 1)
		case 7:
			result = bool2int(states[sdl.SCANCODE_RIGHT] == 1)
		}
		if cpu.GamePadIndex == 7 {
			cpu.GamePadIndex = 0
		} else {
			cpu.GamePadIndex++
		}
		return result
	}
	if index < 0x4020 {

	}
	if index < 0x6000 {

	}
	if index < 0x8000 {

	}
	if index >= 0xC000 {
		if len(cpu.PrgROM) == 0x8000 {
			return int(cpu.PrgROM[index-0x8000])
		}
		return int(cpu.PrgROM[index-0xC000])
	}
	return int(cpu.PrgROM[index-0x8000])
}

func (cpu *Cpu) Reset() {
	var f, s int
	// TODO: impl
	if len(cpu.PrgROM) == 0x4000 {
		f = cpu.Read(0xBFFC)
		s = cpu.Read(0xBFFD)
	} else {
		f = cpu.Read(0xFFFC)
		s = cpu.Read(0xFFFD)
	}
	cpu.Register.PC = s*256 + f
}

func (cpu *Cpu) ProcessNMI() {
	var f, s int
	// TODO: impl
	if len(cpu.PrgROM) == 0x4000 {
		f = cpu.Read(0xBFFA)
		s = cpu.Read(0xBFFB)
	} else {
		f = cpu.Read(0xFFFA)
		s = cpu.Read(0xFFFB)
	}
	cpu.PushStack((cpu.Register.PC >> 8) & 0xff)
	cpu.PushStack(cpu.Register.PC & 0xff)
	cpu.PushStack(cpu.Register.P.Int())
	cpu.Register.P.Interrupt = true
	cpu.interrupts.Nmi = false
	cpu.Register.PC = s*256 + f
}

func (cpu *Cpu) ProcessIrq() {
	var f, s int
	// TODO: impl
	if len(cpu.PrgROM) == 0x4000 {
		f = cpu.Read(0xBFFE)
		s = cpu.Read(0xBFFF)
	} else {
		f = cpu.Read(0xFFFE)
		s = cpu.Read(0xFFFF)
	}
	cpu.Register.P.Break = false
	cpu.Register.P.Interrupt = true
	cpu.Register.PC = s*256 + f
}

func (cpu *Cpu) Fetch() int {
	ret := cpu.Read(cpu.Register.PC)
	cpu.Register.PC++
	return ret
}

var dbg bool

func (cpu *Cpu) Run() int {
	if cpu.interrupts.Nmi {
		cpu.ProcessNMI()
	}

	opCodeRaw := cpu.Fetch()
	opCode := opCodeList[opCodeRaw]
	opCode.FetchOperand(cpu)
	if false {
		dbg = true
		debug(cpu.Register.PC)
		debug(opCode)
		debug(cpu.Register)
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
	}
	cpu.Execute(opCode)
	return cycles[opCodeRaw]
}

func (cpu *Cpu) Execute(opCode *OpCode) {
	var data int
	switch opCode.Base {
	case "ADC":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		r := cpu.Register.A + data + bool2int(cpu.Register.P.Carry)
		cpu.Register.P.Negative = r>>6 == 1
		//cpu.Register.P.Overflow = r
		cpu.Register.P.Zero = r == 0
		//cpu.Register.P.Carry = r == 0
		cpu.Register.A = r
	case "SBC":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		r := cpu.Register.A - data + bool2int(!cpu.Register.P.Carry)
		cpu.Register.P.Negative = r>>6 == 1
		//cpu.Register.P.Overflow = r
		cpu.Register.P.Zero = r == 0
		//cpu.Register.P.Carry = r == 0
		cpu.Register.A = r
	case "AND":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		r := cpu.Register.A & data
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		cpu.Register.A = r
	case "ORA":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		r := cpu.Register.A | data
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		cpu.Register.A = r
	case "EOR":
		r := cpu.Register.A ^ data
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		cpu.Register.A = r
	case "ASL":
		cpu.Register.A <<= 1
		r := cpu.Register.A & int(math.Pow(2, 7))
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		cpu.Register.P.Carry = r != 0
	case "LSR":
		cpu.Register.A >>= 1
		r := cpu.Register.A & int(math.Pow(2, 0))
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		cpu.Register.P.Carry = r != 0
	case "ROL":
		cpu.Register.A = cpu.Register.A<<1 + bool2int(cpu.Register.P.Carry)
		r := cpu.Register.A & int(math.Pow(2, 7))
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		cpu.Register.P.Carry = r != 0
	case "ROR":
		cpu.Register.A = cpu.Register.A>>1 + bool2int(cpu.Register.P.Carry)*int(math.Pow(2, 7))
		r := cpu.Register.A & int(math.Pow(2, 0))
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		cpu.Register.P.Carry = r != 0
	case "BCC":
		if !cpu.Register.P.Carry {
			cpu.Register.PC = opCode.Operand
		}
	case "BCS":
		if cpu.Register.P.Carry {
			cpu.Register.PC = opCode.Operand
		}
	case "BEQ":
		if cpu.Register.P.Zero {
			cpu.Register.PC = opCode.Operand
		}
	case "BNE":
		if !cpu.Register.P.Zero {
			cpu.Register.PC = opCode.Operand
		}
	case "BVC":
		if !cpu.Register.P.Overflow {
			cpu.Register.PC = opCode.Operand
		}
	case "BVS":
		if cpu.Register.P.Overflow {
			cpu.Register.PC = opCode.Operand
		}
	case "BPL":
		if !cpu.Register.P.Negative {
			cpu.Register.PC = opCode.Operand
		}
	case "BMI":
		if cpu.Register.P.Negative {
			cpu.Register.PC = opCode.Operand
		}
	case "BIT":
		// TODO: impl
	case "JMP":
		cpu.Register.PC = opCode.Operand
	case "JSR":
		cpu.PushStack(cpu.Register.PC)
		cpu.Register.PC = opCode.Operand
	case "RTS":
		cpu.Register.PC = cpu.PopStack()
	case "BRK":
	case "RTI":
		status := cpu.PopStack()
		l := cpu.PopStack()
		h := cpu.PopStack()
		cpu.Register.P.Set(status)
		cpu.Register.PC = h*256 + l
	case "CMP":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		r := cpu.Register.A - data
		if r > 0 {
			cpu.Register.P.Carry = true
		} else {
			cpu.Register.P.Carry = false
		}
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		// cpu.Register.P.Carry
	case "CPX":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		r := cpu.Register.X - data
		if r > 0 {
			cpu.Register.P.Carry = true
		} else {
			cpu.Register.P.Carry = false
		}
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		// cpu.Register.P.Carry
	case "CPY":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		r := cpu.Register.Y - data
		if r > 0 {
			cpu.Register.P.Carry = true
		} else {
			cpu.Register.P.Carry = false
		}
		cpu.Register.P.Negative = r>>6 == 1
		cpu.Register.P.Zero = r == 0
		cpu.Register.P.Carry = r >= 0
	case "INC":
		data = cpu.Read(opCode.Operand)
		cpu.Write(opCode.Operand, data+1)
		cpu.Register.P.Negative = (data+1)>>6 == 1
		cpu.Register.P.Zero = data+1 == 0
	case "DEC":
		data = cpu.Read(opCode.Operand)
		cpu.Write(opCode.Operand, data-1)
		cpu.Register.P.Negative = (data-1)>>6 == 1
		cpu.Register.P.Zero = data-1 == 0
	case "INX":
		cpu.Register.X++
		cpu.Register.P.Negative = cpu.Register.X>>6 == 1
		cpu.Register.P.Zero = cpu.Register.X == 0
	case "DEX":
		cpu.Register.X--
		cpu.Register.P.Negative = cpu.Register.X>>6 == 1
		cpu.Register.P.Zero = cpu.Register.X == 0
	case "INY":
		cpu.Register.Y++
		cpu.Register.P.Negative = cpu.Register.Y>>6 == 1
		cpu.Register.P.Zero = cpu.Register.Y == 0
	case "DEY":
		cpu.Register.Y--
		cpu.Register.P.Negative = cpu.Register.Y>>6 == 1
		cpu.Register.P.Zero = cpu.Register.Y == 0
	case "CLC":
		cpu.Register.P.Carry = false
	case "SEC":
		cpu.Register.P.Carry = true
	case "CLI":
		cpu.Register.P.Interrupt = false
	case "SEI":
		cpu.Register.P.Interrupt = true
	case "CLD":
		cpu.Register.P.Decimal = false
	case "SED":
		cpu.Register.P.Decimal = true
	case "CLV":
		cpu.Register.P.Overflow = false
	case "LDA":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		cpu.Register.A = data
		cpu.Register.P.Negative = data>>7 == 1
		cpu.Register.P.Zero = data == 0
	case "LDX":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		cpu.Register.X = data
		cpu.Register.P.Negative = data>>6 == 1
		cpu.Register.P.Zero = data == 0
	case "LDY":
		if opCode.Mode == ADDR_IMMEDIATE {
			data = opCode.Operand
		} else {
			data = cpu.Read(opCode.Operand)
		}
		cpu.Register.Y = data
		cpu.Register.P.Negative = data>>6 == 1
		cpu.Register.P.Zero = data == 0
	case "STA":
		cpu.Write(opCode.Operand, cpu.Register.A)
	case "STX":
		cpu.Write(opCode.Operand, cpu.Register.X)
	case "STY":
		cpu.Write(opCode.Operand, cpu.Register.Y)
	case "TAX":
		cpu.Register.X = cpu.Register.A
		cpu.Register.P.Negative = cpu.Register.A>>6 == 1
		cpu.Register.P.Zero = cpu.Register.A == 0
	case "TXA":
		cpu.Register.A = cpu.Register.X
		cpu.Register.P.Negative = cpu.Register.X>>6 == 1
		cpu.Register.P.Zero = cpu.Register.X == 0
	case "TAY":
		cpu.Register.Y = cpu.Register.A
		cpu.Register.P.Negative = cpu.Register.A>>6 == 1
		cpu.Register.P.Zero = cpu.Register.A == 0
	case "TYA":
		cpu.Register.A = cpu.Register.Y
		cpu.Register.P.Negative = cpu.Register.Y>>6 == 1
		cpu.Register.P.Zero = cpu.Register.Y == 0
	case "TSX":
		cpu.Register.X = cpu.Register.SP
		cpu.Register.P.Negative = cpu.Register.SP>>6 == 1
		cpu.Register.P.Zero = cpu.Register.SP == 0
	case "TXS":
		cpu.Register.SP = cpu.Register.X
	case "PHA":
		cpu.PushStack(cpu.Register.A)
	case "PLA":
		cpu.Register.A = cpu.PopStack()
		cpu.Register.P.Negative = cpu.Register.A>>6 == 1
		cpu.Register.P.Zero = cpu.Register.A == 0
	case "PHP":
		cpu.PushStack(cpu.Register.P.Int())
	case "PLP":
		cpu.Register.P.Set(cpu.PopStack())
	case "NOP":
		return
	}
}

func (cpu *Cpu) PushStack(value int) {
	cpu.RAM[0x100+cpu.Register.SP] = value
	cpu.Register.SP++
}

func (cpu *Cpu) PopStack() int {
	cpu.Register.SP--
	return cpu.RAM[0x100+cpu.Register.SP]
}

package main

func main() {
	const input = 46

	const ioRegister = 10

	code := []byte{
		// _Z14fib_imperativei
		0x2a, 0x86, // 0:  c.mv    a2, a0
		0x89, 0x47, // 2:  c.li    a5, 2
		0x05, 0x45, // 4:  c.li    a0, 1
		0x63, 0xda, 0xc7, 0x00, // 6:  bge    a5, a2, 20 # .L4
		0x05, 0x47, // 10:  c.li    a4, 1
		// .L3
		0xaa, 0x86, // 12:  c.mv    a3, a0
		0x85, 0x07, // 14:  c.addi    a5, 1
		0x3a, 0x95, // 16:  c.add    a0, a4
		0x36, 0x87, // 18:  c.mv    a4, a3
		0xe3, 0x1c, 0xf6, 0xfe, // 20:  bne    a2, a5, -8 # .L3
		0x02, 0x90, // c.ebreak
		// .L4
		0x02, 0x90, // c.ebreak
	}

	dec := decoder{code: code}
	instructions := dec.decodeInstructions()

	vm := vm{
		instructions: instructions,
	}

	vm.registers[ioRegister] = input

	vm.run(false)

	println(vm.registers[ioRegister])
}

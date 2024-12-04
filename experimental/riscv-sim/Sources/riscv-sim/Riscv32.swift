//
//  Riscv32.swift
//  Riscv32
//

public class Riscv32 {
    
    public static func decodeInstructions( memory: Memory,
                                           instructionSize: Int ) -> [Instruction] {
        var instructions = [Instruction]()
        let cNop = CNop()!
        
        var address: UInt32 = 0
        let endAddress = address + UInt32( instructionSize )
        
        let fetcher = Riscv32.Fetcher( memory: memory, address: address )
        let decoder = Riscv32.Decoder()
        while address < endAddress {
            guard let bitPattern = fetcher.fetchInstruction() else { break }
            
            if let instruction = decoder.decodeInstruction( instruction: bitPattern ) {
                instructions.append( instruction )
                if instruction.size == 4 {
                    instructions.append( cNop )
                }
            }
            address = fetcher.address
        }
        
        return instructions
    }
}

public class RiscvCmd {
}

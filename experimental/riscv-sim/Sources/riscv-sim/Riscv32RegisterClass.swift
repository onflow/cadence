//
//  Riscv32RegisterClass.swift
//  Riscv32
//

protocol Riscv32RegisterClass {
    var bitPattern: UInt32 { get }
    
    init?( bitPattern: UInt32 )
    func isLegal() -> Bool
}

extension Riscv32 {
    
    public enum csr_sysreg {
        case ustatus
        case fflags
        case frm
        case fcsr
        case uie
        case utvec
        case vstart
        case vxsat
        case vxrm
        case vcsr
        case seed
        case uscratch
        case uepc
        case ucause
        case utval
        case uip
        case sstatus
        case sedeleg
        case sideleg
        case sie
        case stvec
        case scounteren
        case senvcfg
        case sstateen0
        case sstateen1
        case sstateen2
        case sstateen3
        case sscratch
        case sepc
        case scause
        case stval
        case sip
        case stimecmp
        case stimecmph
        case satp
        case vsstatus
        case vsie
        case vstvec
        case vsscratch
        case vsepc
        case vscause
        case vstval
        case vsip
        case vstimecmp
        case vstimecmph
        case vsatp
        case mstatus
        case misa
        case medeleg
        case mideleg
        case mie
        case mtvec
        case mcounteren
        case menvcfg
        case mstateen0
        case mstateen1
        case mstateen2
        case mstateen3
        case mstatush
        case menvcfgh
        case mstateen0h
        case mstateen1h
        case mstateen2h
        case mstateen3h
        case mcountinhibit
        case mhpmevent3
        case mhpmevent4
        case mhpmevent5
        case mhpmevent6
        case mhpmevent7
        case mhpmevent8
        case mhpmevent9
        case mhpmevent10
        case mhpmevent11
        case mhpmevent12
        case mhpmevent13
        case mhpmevent14
        case mhpmevent15
        case mhpmevent16
        case mhpmevent17
        case mhpmevent18
        case mhpmevent19
        case mhpmevent20
        case mhpmevent21
        case mhpmevent22
        case mhpmevent23
        case mhpmevent24
        case mhpmevent25
        case mhpmevent26
        case mhpmevent27
        case mhpmevent28
        case mhpmevent29
        case mhpmevent30
        case mhpmevent31
        case mscratch
        case mepc
        case mcause
        case mtval
        case mip
        case mtinst
        case mtval2
        case pmpcfg0
        case pmpcfg1
        case pmpcfg2
        case pmpcfg3
        case pmpcfg4
        case pmpcfg5
        case pmpcfg6
        case pmpcfg7
        case pmpcfg8
        case pmpcfg9
        case pmpcfg10
        case pmpcfg11
        case pmpcfg12
        case pmpcfg13
        case pmpcfg14
        case pmpcfg15
        case pmpaddr0
        case pmpaddr1
        case pmpaddr2
        case pmpaddr3
        case pmpaddr4
        case pmpaddr5
        case pmpaddr6
        case pmpaddr7
        case pmpaddr8
        case pmpaddr9
        case pmpaddr10
        case pmpaddr11
        case pmpaddr12
        case pmpaddr13
        case pmpaddr14
        case pmpaddr15
        case pmpaddr16
        case pmpaddr17
        case pmpaddr18
        case pmpaddr19
        case pmpaddr20
        case pmpaddr21
        case pmpaddr22
        case pmpaddr23
        case pmpaddr24
        case pmpaddr25
        case pmpaddr26
        case pmpaddr27
        case pmpaddr28
        case pmpaddr29
        case pmpaddr30
        case pmpaddr31
        case pmpaddr32
        case pmpaddr33
        case pmpaddr34
        case pmpaddr35
        case pmpaddr36
        case pmpaddr37
        case pmpaddr38
        case pmpaddr39
        case pmpaddr40
        case pmpaddr41
        case pmpaddr42
        case pmpaddr43
        case pmpaddr44
        case pmpaddr45
        case pmpaddr46
        case pmpaddr47
        case pmpaddr48
        case pmpaddr49
        case pmpaddr50
        case pmpaddr51
        case pmpaddr52
        case pmpaddr53
        case pmpaddr54
        case pmpaddr55
        case pmpaddr56
        case pmpaddr57
        case pmpaddr58
        case pmpaddr59
        case pmpaddr60
        case pmpaddr61
        case pmpaddr62
        case pmpaddr63
        case scontext
        case hstatus
        case hedeleg
        case hideleg
        case hie
        case htimedelta
        case hcounteren
        case hgeie
        case henvcfg
        case hstateen0
        case hstateen1
        case hstateen2
        case hstateen3
        case htimedeltah
        case henvcfgh
        case hstateen0h
        case hstateen1h
        case hstateen2h
        case hstateen3h
        case htval
        case hip
        case hvip
        case htinst
        case hgatp
        case hcontext
        case mhpmevent3h
        case mhpmevent4h
        case mhpmevent5h
        case mhpmevent6h
        case mhpmevent7h
        case mhpmevent8h
        case mhpmevent9h
        case mhpmevent10h
        case mhpmevent11h
        case mhpmevent12h
        case mhpmevent13h
        case mhpmevent14h
        case mhpmevent15h
        case mhpmevent16h
        case mhpmevent17h
        case mhpmevent18h
        case mhpmevent19h
        case mhpmevent20h
        case mhpmevent21h
        case mhpmevent22h
        case mhpmevent23h
        case mhpmevent24h
        case mhpmevent25h
        case mhpmevent26h
        case mhpmevent27h
        case mhpmevent28h
        case mhpmevent29h
        case mhpmevent30h
        case mhpmevent31h
        case mseccfg
        case mseccfgh
        case tselect
        case tdata1
        case tdata2
        case tdata3
        case mcontext
        case dcsr
        case dpc
        case dscratch0
        case dscratch1
        case mcycle
        case minstret
        case mhpmcounter3
        case mhpmcounter4
        case mhpmcounter5
        case mhpmcounter6
        case mhpmcounter7
        case mhpmcounter8
        case mhpmcounter9
        case mhpmcounter10
        case mhpmcounter11
        case mhpmcounter12
        case mhpmcounter13
        case mhpmcounter14
        case mhpmcounter15
        case mhpmcounter16
        case mhpmcounter17
        case mhpmcounter18
        case mhpmcounter19
        case mhpmcounter20
        case mhpmcounter21
        case mhpmcounter22
        case mhpmcounter23
        case mhpmcounter24
        case mhpmcounter25
        case mhpmcounter26
        case mhpmcounter27
        case mhpmcounter28
        case mhpmcounter29
        case mhpmcounter30
        case mhpmcounter31
        case mcycleh
        case minstreth
        case mhpmcounter3h
        case mhpmcounter4h
        case mhpmcounter5h
        case mhpmcounter6h
        case mhpmcounter7h
        case mhpmcounter8h
        case mhpmcounter9h
        case mhpmcounter10h
        case mhpmcounter11h
        case mhpmcounter12h
        case mhpmcounter13h
        case mhpmcounter14h
        case mhpmcounter15h
        case mhpmcounter16h
        case mhpmcounter17h
        case mhpmcounter18h
        case mhpmcounter19h
        case mhpmcounter20h
        case mhpmcounter21h
        case mhpmcounter22h
        case mhpmcounter23h
        case mhpmcounter24h
        case mhpmcounter25h
        case mhpmcounter26h
        case mhpmcounter27h
        case mhpmcounter28h
        case mhpmcounter29h
        case mhpmcounter30h
        case mhpmcounter31h
        case cycle
        case time
        case instret
        case hpmcounter3
        case hpmcounter4
        case hpmcounter5
        case hpmcounter6
        case hpmcounter7
        case hpmcounter8
        case hpmcounter9
        case hpmcounter10
        case hpmcounter11
        case hpmcounter12
        case hpmcounter13
        case hpmcounter14
        case hpmcounter15
        case hpmcounter16
        case hpmcounter17
        case hpmcounter18
        case hpmcounter19
        case hpmcounter20
        case hpmcounter21
        case hpmcounter22
        case hpmcounter23
        case hpmcounter24
        case hpmcounter25
        case hpmcounter26
        case hpmcounter27
        case hpmcounter28
        case hpmcounter29
        case hpmcounter30
        case hpmcounter31
        case vl
        case vtype
        case vlenb
        case cycleh
        case timeh
        case instreth
        case hpmcounter3h
        case hpmcounter4h
        case hpmcounter5h
        case hpmcounter6h
        case hpmcounter7h
        case hpmcounter8h
        case hpmcounter9h
        case hpmcounter10h
        case hpmcounter11h
        case hpmcounter12h
        case hpmcounter13h
        case hpmcounter14h
        case hpmcounter15h
        case hpmcounter16h
        case hpmcounter17h
        case hpmcounter18h
        case hpmcounter19h
        case hpmcounter20h
        case hpmcounter21h
        case hpmcounter22h
        case hpmcounter23h
        case hpmcounter24h
        case hpmcounter25h
        case hpmcounter26h
        case hpmcounter27h
        case hpmcounter28h
        case hpmcounter29h
        case hpmcounter30h
        case hpmcounter31h
        case scountovf
        case hgeip
        case mvendorid
        case marchid
        case mimpid
        case mhartid
        case mconfigptr
        
        public func name() -> String {
            switch self {
            case .ustatus: return "ustatus"
            case .fflags: return "fflags"
            case .frm: return "frm"
            case .fcsr: return "fcsr"
            case .uie: return "uie"
            case .utvec: return "utvec"
            case .vstart: return "vstart"
            case .vxsat: return "vxsat"
            case .vxrm: return "vxrm"
            case .vcsr: return "vcsr"
            case .seed: return "seed"
            case .uscratch: return "uscratch"
            case .uepc: return "uepc"
            case .ucause: return "ucause"
            case .utval: return "utval"
            case .uip: return "uip"
            case .sstatus: return "sstatus"
            case .sedeleg: return "sedeleg"
            case .sideleg: return "sideleg"
            case .sie: return "sie"
            case .stvec: return "stvec"
            case .scounteren: return "scounteren"
            case .senvcfg: return "senvcfg"
            case .sstateen0: return "sstateen0"
            case .sstateen1: return "sstateen1"
            case .sstateen2: return "sstateen2"
            case .sstateen3: return "sstateen3"
            case .sscratch: return "sscratch"
            case .sepc: return "sepc"
            case .scause: return "scause"
            case .stval: return "stval"
            case .sip: return "sip"
            case .stimecmp: return "stimecmp"
            case .stimecmph: return "stimecmph"
            case .satp: return "satp"
            case .vsstatus: return "vsstatus"
            case .vsie: return "vsie"
            case .vstvec: return "vstvec"
            case .vsscratch: return "vsscratch"
            case .vsepc: return "vsepc"
            case .vscause: return "vscause"
            case .vstval: return "vstval"
            case .vsip: return "vsip"
            case .vstimecmp: return "vstimecmp"
            case .vstimecmph: return "vstimecmph"
            case .vsatp: return "vsatp"
            case .mstatus: return "mstatus"
            case .misa: return "misa"
            case .medeleg: return "medeleg"
            case .mideleg: return "mideleg"
            case .mie: return "mie"
            case .mtvec: return "mtvec"
            case .mcounteren: return "mcounteren"
            case .menvcfg: return "menvcfg"
            case .mstateen0: return "mstateen0"
            case .mstateen1: return "mstateen1"
            case .mstateen2: return "mstateen2"
            case .mstateen3: return "mstateen3"
            case .mstatush: return "mstatush"
            case .menvcfgh: return "menvcfgh"
            case .mstateen0h: return "mstateen0h"
            case .mstateen1h: return "mstateen1h"
            case .mstateen2h: return "mstateen2h"
            case .mstateen3h: return "mstateen3h"
            case .mcountinhibit: return "mcountinhibit"
            case .mhpmevent3: return "mhpmevent3"
            case .mhpmevent4: return "mhpmevent4"
            case .mhpmevent5: return "mhpmevent5"
            case .mhpmevent6: return "mhpmevent6"
            case .mhpmevent7: return "mhpmevent7"
            case .mhpmevent8: return "mhpmevent8"
            case .mhpmevent9: return "mhpmevent9"
            case .mhpmevent10: return "mhpmevent10"
            case .mhpmevent11: return "mhpmevent11"
            case .mhpmevent12: return "mhpmevent12"
            case .mhpmevent13: return "mhpmevent13"
            case .mhpmevent14: return "mhpmevent14"
            case .mhpmevent15: return "mhpmevent15"
            case .mhpmevent16: return "mhpmevent16"
            case .mhpmevent17: return "mhpmevent17"
            case .mhpmevent18: return "mhpmevent18"
            case .mhpmevent19: return "mhpmevent19"
            case .mhpmevent20: return "mhpmevent20"
            case .mhpmevent21: return "mhpmevent21"
            case .mhpmevent22: return "mhpmevent22"
            case .mhpmevent23: return "mhpmevent23"
            case .mhpmevent24: return "mhpmevent24"
            case .mhpmevent25: return "mhpmevent25"
            case .mhpmevent26: return "mhpmevent26"
            case .mhpmevent27: return "mhpmevent27"
            case .mhpmevent28: return "mhpmevent28"
            case .mhpmevent29: return "mhpmevent29"
            case .mhpmevent30: return "mhpmevent30"
            case .mhpmevent31: return "mhpmevent31"
            case .mscratch: return "mscratch"
            case .mepc: return "mepc"
            case .mcause: return "mcause"
            case .mtval: return "mtval"
            case .mip: return "mip"
            case .mtinst: return "mtinst"
            case .mtval2: return "mtval2"
            case .pmpcfg0: return "pmpcfg0"
            case .pmpcfg1: return "pmpcfg1"
            case .pmpcfg2: return "pmpcfg2"
            case .pmpcfg3: return "pmpcfg3"
            case .pmpcfg4: return "pmpcfg4"
            case .pmpcfg5: return "pmpcfg5"
            case .pmpcfg6: return "pmpcfg6"
            case .pmpcfg7: return "pmpcfg7"
            case .pmpcfg8: return "pmpcfg8"
            case .pmpcfg9: return "pmpcfg9"
            case .pmpcfg10: return "pmpcfg10"
            case .pmpcfg11: return "pmpcfg11"
            case .pmpcfg12: return "pmpcfg12"
            case .pmpcfg13: return "pmpcfg13"
            case .pmpcfg14: return "pmpcfg14"
            case .pmpcfg15: return "pmpcfg15"
            case .pmpaddr0: return "pmpaddr0"
            case .pmpaddr1: return "pmpaddr1"
            case .pmpaddr2: return "pmpaddr2"
            case .pmpaddr3: return "pmpaddr3"
            case .pmpaddr4: return "pmpaddr4"
            case .pmpaddr5: return "pmpaddr5"
            case .pmpaddr6: return "pmpaddr6"
            case .pmpaddr7: return "pmpaddr7"
            case .pmpaddr8: return "pmpaddr8"
            case .pmpaddr9: return "pmpaddr9"
            case .pmpaddr10: return "pmpaddr10"
            case .pmpaddr11: return "pmpaddr11"
            case .pmpaddr12: return "pmpaddr12"
            case .pmpaddr13: return "pmpaddr13"
            case .pmpaddr14: return "pmpaddr14"
            case .pmpaddr15: return "pmpaddr15"
            case .pmpaddr16: return "pmpaddr16"
            case .pmpaddr17: return "pmpaddr17"
            case .pmpaddr18: return "pmpaddr18"
            case .pmpaddr19: return "pmpaddr19"
            case .pmpaddr20: return "pmpaddr20"
            case .pmpaddr21: return "pmpaddr21"
            case .pmpaddr22: return "pmpaddr22"
            case .pmpaddr23: return "pmpaddr23"
            case .pmpaddr24: return "pmpaddr24"
            case .pmpaddr25: return "pmpaddr25"
            case .pmpaddr26: return "pmpaddr26"
            case .pmpaddr27: return "pmpaddr27"
            case .pmpaddr28: return "pmpaddr28"
            case .pmpaddr29: return "pmpaddr29"
            case .pmpaddr30: return "pmpaddr30"
            case .pmpaddr31: return "pmpaddr31"
            case .pmpaddr32: return "pmpaddr32"
            case .pmpaddr33: return "pmpaddr33"
            case .pmpaddr34: return "pmpaddr34"
            case .pmpaddr35: return "pmpaddr35"
            case .pmpaddr36: return "pmpaddr36"
            case .pmpaddr37: return "pmpaddr37"
            case .pmpaddr38: return "pmpaddr38"
            case .pmpaddr39: return "pmpaddr39"
            case .pmpaddr40: return "pmpaddr40"
            case .pmpaddr41: return "pmpaddr41"
            case .pmpaddr42: return "pmpaddr42"
            case .pmpaddr43: return "pmpaddr43"
            case .pmpaddr44: return "pmpaddr44"
            case .pmpaddr45: return "pmpaddr45"
            case .pmpaddr46: return "pmpaddr46"
            case .pmpaddr47: return "pmpaddr47"
            case .pmpaddr48: return "pmpaddr48"
            case .pmpaddr49: return "pmpaddr49"
            case .pmpaddr50: return "pmpaddr50"
            case .pmpaddr51: return "pmpaddr51"
            case .pmpaddr52: return "pmpaddr52"
            case .pmpaddr53: return "pmpaddr53"
            case .pmpaddr54: return "pmpaddr54"
            case .pmpaddr55: return "pmpaddr55"
            case .pmpaddr56: return "pmpaddr56"
            case .pmpaddr57: return "pmpaddr57"
            case .pmpaddr58: return "pmpaddr58"
            case .pmpaddr59: return "pmpaddr59"
            case .pmpaddr60: return "pmpaddr60"
            case .pmpaddr61: return "pmpaddr61"
            case .pmpaddr62: return "pmpaddr62"
            case .pmpaddr63: return "pmpaddr63"
            case .scontext: return "scontext"
            case .hstatus: return "hstatus"
            case .hedeleg: return "hedeleg"
            case .hideleg: return "hideleg"
            case .hie: return "hie"
            case .htimedelta: return "htimedelta"
            case .hcounteren: return "hcounteren"
            case .hgeie: return "hgeie"
            case .henvcfg: return "henvcfg"
            case .hstateen0: return "hstateen0"
            case .hstateen1: return "hstateen1"
            case .hstateen2: return "hstateen2"
            case .hstateen3: return "hstateen3"
            case .htimedeltah: return "htimedeltah"
            case .henvcfgh: return "henvcfgh"
            case .hstateen0h: return "hstateen0h"
            case .hstateen1h: return "hstateen1h"
            case .hstateen2h: return "hstateen2h"
            case .hstateen3h: return "hstateen3h"
            case .htval: return "htval"
            case .hip: return "hip"
            case .hvip: return "hvip"
            case .htinst: return "htinst"
            case .hgatp: return "hgatp"
            case .hcontext: return "hcontext"
            case .mhpmevent3h: return "mhpmevent3h"
            case .mhpmevent4h: return "mhpmevent4h"
            case .mhpmevent5h: return "mhpmevent5h"
            case .mhpmevent6h: return "mhpmevent6h"
            case .mhpmevent7h: return "mhpmevent7h"
            case .mhpmevent8h: return "mhpmevent8h"
            case .mhpmevent9h: return "mhpmevent9h"
            case .mhpmevent10h: return "mhpmevent10h"
            case .mhpmevent11h: return "mhpmevent11h"
            case .mhpmevent12h: return "mhpmevent12h"
            case .mhpmevent13h: return "mhpmevent13h"
            case .mhpmevent14h: return "mhpmevent14h"
            case .mhpmevent15h: return "mhpmevent15h"
            case .mhpmevent16h: return "mhpmevent16h"
            case .mhpmevent17h: return "mhpmevent17h"
            case .mhpmevent18h: return "mhpmevent18h"
            case .mhpmevent19h: return "mhpmevent19h"
            case .mhpmevent20h: return "mhpmevent20h"
            case .mhpmevent21h: return "mhpmevent21h"
            case .mhpmevent22h: return "mhpmevent22h"
            case .mhpmevent23h: return "mhpmevent23h"
            case .mhpmevent24h: return "mhpmevent24h"
            case .mhpmevent25h: return "mhpmevent25h"
            case .mhpmevent26h: return "mhpmevent26h"
            case .mhpmevent27h: return "mhpmevent27h"
            case .mhpmevent28h: return "mhpmevent28h"
            case .mhpmevent29h: return "mhpmevent29h"
            case .mhpmevent30h: return "mhpmevent30h"
            case .mhpmevent31h: return "mhpmevent31h"
            case .mseccfg: return "mseccfg"
            case .mseccfgh: return "mseccfgh"
            case .tselect: return "tselect"
            case .tdata1: return "tdata1"
            case .tdata2: return "tdata2"
            case .tdata3: return "tdata3"
            case .mcontext: return "mcontext"
            case .dcsr: return "dcsr"
            case .dpc: return "dpc"
            case .dscratch0: return "dscratch0"
            case .dscratch1: return "dscratch1"
            case .mcycle: return "mcycle"
            case .minstret: return "minstret"
            case .mhpmcounter3: return "mhpmcounter3"
            case .mhpmcounter4: return "mhpmcounter4"
            case .mhpmcounter5: return "mhpmcounter5"
            case .mhpmcounter6: return "mhpmcounter6"
            case .mhpmcounter7: return "mhpmcounter7"
            case .mhpmcounter8: return "mhpmcounter8"
            case .mhpmcounter9: return "mhpmcounter9"
            case .mhpmcounter10: return "mhpmcounter10"
            case .mhpmcounter11: return "mhpmcounter11"
            case .mhpmcounter12: return "mhpmcounter12"
            case .mhpmcounter13: return "mhpmcounter13"
            case .mhpmcounter14: return "mhpmcounter14"
            case .mhpmcounter15: return "mhpmcounter15"
            case .mhpmcounter16: return "mhpmcounter16"
            case .mhpmcounter17: return "mhpmcounter17"
            case .mhpmcounter18: return "mhpmcounter18"
            case .mhpmcounter19: return "mhpmcounter19"
            case .mhpmcounter20: return "mhpmcounter20"
            case .mhpmcounter21: return "mhpmcounter21"
            case .mhpmcounter22: return "mhpmcounter22"
            case .mhpmcounter23: return "mhpmcounter23"
            case .mhpmcounter24: return "mhpmcounter24"
            case .mhpmcounter25: return "mhpmcounter25"
            case .mhpmcounter26: return "mhpmcounter26"
            case .mhpmcounter27: return "mhpmcounter27"
            case .mhpmcounter28: return "mhpmcounter28"
            case .mhpmcounter29: return "mhpmcounter29"
            case .mhpmcounter30: return "mhpmcounter30"
            case .mhpmcounter31: return "mhpmcounter31"
            case .mcycleh: return "mcycleh"
            case .minstreth: return "minstreth"
            case .mhpmcounter3h: return "mhpmcounter3h"
            case .mhpmcounter4h: return "mhpmcounter4h"
            case .mhpmcounter5h: return "mhpmcounter5h"
            case .mhpmcounter6h: return "mhpmcounter6h"
            case .mhpmcounter7h: return "mhpmcounter7h"
            case .mhpmcounter8h: return "mhpmcounter8h"
            case .mhpmcounter9h: return "mhpmcounter9h"
            case .mhpmcounter10h: return "mhpmcounter10h"
            case .mhpmcounter11h: return "mhpmcounter11h"
            case .mhpmcounter12h: return "mhpmcounter12h"
            case .mhpmcounter13h: return "mhpmcounter13h"
            case .mhpmcounter14h: return "mhpmcounter14h"
            case .mhpmcounter15h: return "mhpmcounter15h"
            case .mhpmcounter16h: return "mhpmcounter16h"
            case .mhpmcounter17h: return "mhpmcounter17h"
            case .mhpmcounter18h: return "mhpmcounter18h"
            case .mhpmcounter19h: return "mhpmcounter19h"
            case .mhpmcounter20h: return "mhpmcounter20h"
            case .mhpmcounter21h: return "mhpmcounter21h"
            case .mhpmcounter22h: return "mhpmcounter22h"
            case .mhpmcounter23h: return "mhpmcounter23h"
            case .mhpmcounter24h: return "mhpmcounter24h"
            case .mhpmcounter25h: return "mhpmcounter25h"
            case .mhpmcounter26h: return "mhpmcounter26h"
            case .mhpmcounter27h: return "mhpmcounter27h"
            case .mhpmcounter28h: return "mhpmcounter28h"
            case .mhpmcounter29h: return "mhpmcounter29h"
            case .mhpmcounter30h: return "mhpmcounter30h"
            case .mhpmcounter31h: return "mhpmcounter31h"
            case .cycle: return "cycle"
            case .time: return "time"
            case .instret: return "instret"
            case .hpmcounter3: return "hpmcounter3"
            case .hpmcounter4: return "hpmcounter4"
            case .hpmcounter5: return "hpmcounter5"
            case .hpmcounter6: return "hpmcounter6"
            case .hpmcounter7: return "hpmcounter7"
            case .hpmcounter8: return "hpmcounter8"
            case .hpmcounter9: return "hpmcounter9"
            case .hpmcounter10: return "hpmcounter10"
            case .hpmcounter11: return "hpmcounter11"
            case .hpmcounter12: return "hpmcounter12"
            case .hpmcounter13: return "hpmcounter13"
            case .hpmcounter14: return "hpmcounter14"
            case .hpmcounter15: return "hpmcounter15"
            case .hpmcounter16: return "hpmcounter16"
            case .hpmcounter17: return "hpmcounter17"
            case .hpmcounter18: return "hpmcounter18"
            case .hpmcounter19: return "hpmcounter19"
            case .hpmcounter20: return "hpmcounter20"
            case .hpmcounter21: return "hpmcounter21"
            case .hpmcounter22: return "hpmcounter22"
            case .hpmcounter23: return "hpmcounter23"
            case .hpmcounter24: return "hpmcounter24"
            case .hpmcounter25: return "hpmcounter25"
            case .hpmcounter26: return "hpmcounter26"
            case .hpmcounter27: return "hpmcounter27"
            case .hpmcounter28: return "hpmcounter28"
            case .hpmcounter29: return "hpmcounter29"
            case .hpmcounter30: return "hpmcounter30"
            case .hpmcounter31: return "hpmcounter31"
            case .vl: return "vl"
            case .vtype: return "vtype"
            case .vlenb: return "vlenb"
            case .cycleh: return "cycleh"
            case .timeh: return "timeh"
            case .instreth: return "instreth"
            case .hpmcounter3h: return "hpmcounter3h"
            case .hpmcounter4h: return "hpmcounter4h"
            case .hpmcounter5h: return "hpmcounter5h"
            case .hpmcounter6h: return "hpmcounter6h"
            case .hpmcounter7h: return "hpmcounter7h"
            case .hpmcounter8h: return "hpmcounter8h"
            case .hpmcounter9h: return "hpmcounter9h"
            case .hpmcounter10h: return "hpmcounter10h"
            case .hpmcounter11h: return "hpmcounter11h"
            case .hpmcounter12h: return "hpmcounter12h"
            case .hpmcounter13h: return "hpmcounter13h"
            case .hpmcounter14h: return "hpmcounter14h"
            case .hpmcounter15h: return "hpmcounter15h"
            case .hpmcounter16h: return "hpmcounter16h"
            case .hpmcounter17h: return "hpmcounter17h"
            case .hpmcounter18h: return "hpmcounter18h"
            case .hpmcounter19h: return "hpmcounter19h"
            case .hpmcounter20h: return "hpmcounter20h"
            case .hpmcounter21h: return "hpmcounter21h"
            case .hpmcounter22h: return "hpmcounter22h"
            case .hpmcounter23h: return "hpmcounter23h"
            case .hpmcounter24h: return "hpmcounter24h"
            case .hpmcounter25h: return "hpmcounter25h"
            case .hpmcounter26h: return "hpmcounter26h"
            case .hpmcounter27h: return "hpmcounter27h"
            case .hpmcounter28h: return "hpmcounter28h"
            case .hpmcounter29h: return "hpmcounter29h"
            case .hpmcounter30h: return "hpmcounter30h"
            case .hpmcounter31h: return "hpmcounter31h"
            case .scountovf: return "scountovf"
            case .hgeip: return "hgeip"
            case .mvendorid: return "mvendorid"
            case .marchid: return "marchid"
            case .mimpid: return "mimpid"
            case .mhartid: return "mhartid"
            case .mconfigptr: return "mconfigptr"
            }
        }
        
        public func encode() -> UInt32 {
            switch self {
            case .ustatus: return 0x0
            case .fflags: return 0x1
            case .frm: return 0x2
            case .fcsr: return 0x3
            case .uie: return 0x4
            case .utvec: return 0x5
            case .vstart: return 0x8
            case .vxsat: return 0x9
            case .vxrm: return 0xA
            case .vcsr: return 0xF
            case .seed: return 0x15
            case .uscratch: return 0x40
            case .uepc: return 0x41
            case .ucause: return 0x42
            case .utval: return 0x43
            case .uip: return 0x44
            case .sstatus: return 0x100
            case .sedeleg: return 0x102
            case .sideleg: return 0x103
            case .sie: return 0x104
            case .stvec: return 0x105
            case .scounteren: return 0x106
            case .senvcfg: return 0x10A
            case .sstateen0: return 0x10C
            case .sstateen1: return 0x10D
            case .sstateen2: return 0x10E
            case .sstateen3: return 0x10F
            case .sscratch: return 0x140
            case .sepc: return 0x141
            case .scause: return 0x142
            case .stval: return 0x143
            case .sip: return 0x144
            case .stimecmp: return 0x14D
            case .stimecmph: return 0x15D
            case .satp: return 0x180
            case .vsstatus: return 0x200
            case .vsie: return 0x204
            case .vstvec: return 0x205
            case .vsscratch: return 0x240
            case .vsepc: return 0x241
            case .vscause: return 0x242
            case .vstval: return 0x243
            case .vsip: return 0x244
            case .vstimecmp: return 0x24D
            case .vstimecmph: return 0x25D
            case .vsatp: return 0x280
            case .mstatus: return 0x300
            case .misa: return 0x301
            case .medeleg: return 0x302
            case .mideleg: return 0x303
            case .mie: return 0x304
            case .mtvec: return 0x305
            case .mcounteren: return 0x306
            case .menvcfg: return 0x30A
            case .mstateen0: return 0x30C
            case .mstateen1: return 0x30D
            case .mstateen2: return 0x30E
            case .mstateen3: return 0x30F
            case .mstatush: return 0x310
            case .menvcfgh: return 0x31A
            case .mstateen0h: return 0x31C
            case .mstateen1h: return 0x31D
            case .mstateen2h: return 0x31E
            case .mstateen3h: return 0x31F
            case .mcountinhibit: return 0x320
            case .mhpmevent3: return 0x323
            case .mhpmevent4: return 0x324
            case .mhpmevent5: return 0x325
            case .mhpmevent6: return 0x326
            case .mhpmevent7: return 0x327
            case .mhpmevent8: return 0x328
            case .mhpmevent9: return 0x329
            case .mhpmevent10: return 0x32A
            case .mhpmevent11: return 0x32B
            case .mhpmevent12: return 0x32C
            case .mhpmevent13: return 0x32D
            case .mhpmevent14: return 0x32E
            case .mhpmevent15: return 0x32F
            case .mhpmevent16: return 0x330
            case .mhpmevent17: return 0x331
            case .mhpmevent18: return 0x332
            case .mhpmevent19: return 0x333
            case .mhpmevent20: return 0x334
            case .mhpmevent21: return 0x335
            case .mhpmevent22: return 0x336
            case .mhpmevent23: return 0x337
            case .mhpmevent24: return 0x338
            case .mhpmevent25: return 0x339
            case .mhpmevent26: return 0x33A
            case .mhpmevent27: return 0x33B
            case .mhpmevent28: return 0x33C
            case .mhpmevent29: return 0x33D
            case .mhpmevent30: return 0x33E
            case .mhpmevent31: return 0x33F
            case .mscratch: return 0x340
            case .mepc: return 0x341
            case .mcause: return 0x342
            case .mtval: return 0x343
            case .mip: return 0x344
            case .mtinst: return 0x34A
            case .mtval2: return 0x34B
            case .pmpcfg0: return 0x3A0
            case .pmpcfg1: return 0x3A1
            case .pmpcfg2: return 0x3A2
            case .pmpcfg3: return 0x3A3
            case .pmpcfg4: return 0x3A4
            case .pmpcfg5: return 0x3A5
            case .pmpcfg6: return 0x3A6
            case .pmpcfg7: return 0x3A7
            case .pmpcfg8: return 0x3A8
            case .pmpcfg9: return 0x3A9
            case .pmpcfg10: return 0x3AA
            case .pmpcfg11: return 0x3AB
            case .pmpcfg12: return 0x3AC
            case .pmpcfg13: return 0x3AD
            case .pmpcfg14: return 0x3AE
            case .pmpcfg15: return 0x3AF
            case .pmpaddr0: return 0x3B0
            case .pmpaddr1: return 0x3B1
            case .pmpaddr2: return 0x3B2
            case .pmpaddr3: return 0x3B3
            case .pmpaddr4: return 0x3B4
            case .pmpaddr5: return 0x3B5
            case .pmpaddr6: return 0x3B6
            case .pmpaddr7: return 0x3B7
            case .pmpaddr8: return 0x3B8
            case .pmpaddr9: return 0x3B9
            case .pmpaddr10: return 0x3BA
            case .pmpaddr11: return 0x3BB
            case .pmpaddr12: return 0x3BC
            case .pmpaddr13: return 0x3BD
            case .pmpaddr14: return 0x3BE
            case .pmpaddr15: return 0x3BF
            case .pmpaddr16: return 0x3C0
            case .pmpaddr17: return 0x3C1
            case .pmpaddr18: return 0x3C2
            case .pmpaddr19: return 0x3C3
            case .pmpaddr20: return 0x3C4
            case .pmpaddr21: return 0x3C5
            case .pmpaddr22: return 0x3C6
            case .pmpaddr23: return 0x3C7
            case .pmpaddr24: return 0x3C8
            case .pmpaddr25: return 0x3C9
            case .pmpaddr26: return 0x3CA
            case .pmpaddr27: return 0x3CB
            case .pmpaddr28: return 0x3CC
            case .pmpaddr29: return 0x3CD
            case .pmpaddr30: return 0x3CE
            case .pmpaddr31: return 0x3CF
            case .pmpaddr32: return 0x3D0
            case .pmpaddr33: return 0x3D1
            case .pmpaddr34: return 0x3D2
            case .pmpaddr35: return 0x3D3
            case .pmpaddr36: return 0x3D4
            case .pmpaddr37: return 0x3D5
            case .pmpaddr38: return 0x3D6
            case .pmpaddr39: return 0x3D7
            case .pmpaddr40: return 0x3D8
            case .pmpaddr41: return 0x3D9
            case .pmpaddr42: return 0x3DA
            case .pmpaddr43: return 0x3DB
            case .pmpaddr44: return 0x3DC
            case .pmpaddr45: return 0x3DD
            case .pmpaddr46: return 0x3DE
            case .pmpaddr47: return 0x3DF
            case .pmpaddr48: return 0x3E0
            case .pmpaddr49: return 0x3E1
            case .pmpaddr50: return 0x3E2
            case .pmpaddr51: return 0x3E3
            case .pmpaddr52: return 0x3E4
            case .pmpaddr53: return 0x3E5
            case .pmpaddr54: return 0x3E6
            case .pmpaddr55: return 0x3E7
            case .pmpaddr56: return 0x3E8
            case .pmpaddr57: return 0x3E9
            case .pmpaddr58: return 0x3EA
            case .pmpaddr59: return 0x3EB
            case .pmpaddr60: return 0x3EC
            case .pmpaddr61: return 0x3ED
            case .pmpaddr62: return 0x3EE
            case .pmpaddr63: return 0x3EF
            case .scontext: return 0x5A8
            case .hstatus: return 0x600
            case .hedeleg: return 0x602
            case .hideleg: return 0x603
            case .hie: return 0x604
            case .htimedelta: return 0x605
            case .hcounteren: return 0x606
            case .hgeie: return 0x607
            case .henvcfg: return 0x60A
            case .hstateen0: return 0x60C
            case .hstateen1: return 0x60D
            case .hstateen2: return 0x60E
            case .hstateen3: return 0x60F
            case .htimedeltah: return 0x615
            case .henvcfgh: return 0x61A
            case .hstateen0h: return 0x61C
            case .hstateen1h: return 0x61D
            case .hstateen2h: return 0x61E
            case .hstateen3h: return 0x61F
            case .htval: return 0x643
            case .hip: return 0x644
            case .hvip: return 0x645
            case .htinst: return 0x64A
            case .hgatp: return 0x680
            case .hcontext: return 0x6A8
            case .mhpmevent3h: return 0x723
            case .mhpmevent4h: return 0x724
            case .mhpmevent5h: return 0x725
            case .mhpmevent6h: return 0x726
            case .mhpmevent7h: return 0x727
            case .mhpmevent8h: return 0x728
            case .mhpmevent9h: return 0x729
            case .mhpmevent10h: return 0x72A
            case .mhpmevent11h: return 0x72B
            case .mhpmevent12h: return 0x72C
            case .mhpmevent13h: return 0x72D
            case .mhpmevent14h: return 0x72E
            case .mhpmevent15h: return 0x72F
            case .mhpmevent16h: return 0x730
            case .mhpmevent17h: return 0x731
            case .mhpmevent18h: return 0x732
            case .mhpmevent19h: return 0x733
            case .mhpmevent20h: return 0x734
            case .mhpmevent21h: return 0x735
            case .mhpmevent22h: return 0x736
            case .mhpmevent23h: return 0x737
            case .mhpmevent24h: return 0x738
            case .mhpmevent25h: return 0x739
            case .mhpmevent26h: return 0x73A
            case .mhpmevent27h: return 0x73B
            case .mhpmevent28h: return 0x73C
            case .mhpmevent29h: return 0x73D
            case .mhpmevent30h: return 0x73E
            case .mhpmevent31h: return 0x73F
            case .mseccfg: return 0x747
            case .mseccfgh: return 0x757
            case .tselect: return 0x7A0
            case .tdata1: return 0x7A1
            case .tdata2: return 0x7A2
            case .tdata3: return 0x7A3
            case .mcontext: return 0x7A8
            case .dcsr: return 0x7B0
            case .dpc: return 0x7B1
            case .dscratch0: return 0x7B2
            case .dscratch1: return 0x7B3
            case .mcycle: return 0xB00
            case .minstret: return 0xB02
            case .mhpmcounter3: return 0xB03
            case .mhpmcounter4: return 0xB04
            case .mhpmcounter5: return 0xB05
            case .mhpmcounter6: return 0xB06
            case .mhpmcounter7: return 0xB07
            case .mhpmcounter8: return 0xB08
            case .mhpmcounter9: return 0xB09
            case .mhpmcounter10: return 0xB0A
            case .mhpmcounter11: return 0xB0B
            case .mhpmcounter12: return 0xB0C
            case .mhpmcounter13: return 0xB0D
            case .mhpmcounter14: return 0xB0E
            case .mhpmcounter15: return 0xB0F
            case .mhpmcounter16: return 0xB10
            case .mhpmcounter17: return 0xB11
            case .mhpmcounter18: return 0xB12
            case .mhpmcounter19: return 0xB13
            case .mhpmcounter20: return 0xB14
            case .mhpmcounter21: return 0xB15
            case .mhpmcounter22: return 0xB16
            case .mhpmcounter23: return 0xB17
            case .mhpmcounter24: return 0xB18
            case .mhpmcounter25: return 0xB19
            case .mhpmcounter26: return 0xB1A
            case .mhpmcounter27: return 0xB1B
            case .mhpmcounter28: return 0xB1C
            case .mhpmcounter29: return 0xB1D
            case .mhpmcounter30: return 0xB1E
            case .mhpmcounter31: return 0xB1F
            case .mcycleh: return 0xB80
            case .minstreth: return 0xB82
            case .mhpmcounter3h: return 0xB83
            case .mhpmcounter4h: return 0xB84
            case .mhpmcounter5h: return 0xB85
            case .mhpmcounter6h: return 0xB86
            case .mhpmcounter7h: return 0xB87
            case .mhpmcounter8h: return 0xB88
            case .mhpmcounter9h: return 0xB89
            case .mhpmcounter10h: return 0xB8A
            case .mhpmcounter11h: return 0xB8B
            case .mhpmcounter12h: return 0xB8C
            case .mhpmcounter13h: return 0xB8D
            case .mhpmcounter14h: return 0xB8E
            case .mhpmcounter15h: return 0xB8F
            case .mhpmcounter16h: return 0xB90
            case .mhpmcounter17h: return 0xB91
            case .mhpmcounter18h: return 0xB92
            case .mhpmcounter19h: return 0xB93
            case .mhpmcounter20h: return 0xB94
            case .mhpmcounter21h: return 0xB95
            case .mhpmcounter22h: return 0xB96
            case .mhpmcounter23h: return 0xB97
            case .mhpmcounter24h: return 0xB98
            case .mhpmcounter25h: return 0xB99
            case .mhpmcounter26h: return 0xB9A
            case .mhpmcounter27h: return 0xB9B
            case .mhpmcounter28h: return 0xB9C
            case .mhpmcounter29h: return 0xB9D
            case .mhpmcounter30h: return 0xB9E
            case .mhpmcounter31h: return 0xB9F
            case .cycle: return 0xC00
            case .time: return 0xC01
            case .instret: return 0xC02
            case .hpmcounter3: return 0xC03
            case .hpmcounter4: return 0xC04
            case .hpmcounter5: return 0xC05
            case .hpmcounter6: return 0xC06
            case .hpmcounter7: return 0xC07
            case .hpmcounter8: return 0xC08
            case .hpmcounter9: return 0xC09
            case .hpmcounter10: return 0xC0A
            case .hpmcounter11: return 0xC0B
            case .hpmcounter12: return 0xC0C
            case .hpmcounter13: return 0xC0D
            case .hpmcounter14: return 0xC0E
            case .hpmcounter15: return 0xC0F
            case .hpmcounter16: return 0xC10
            case .hpmcounter17: return 0xC11
            case .hpmcounter18: return 0xC12
            case .hpmcounter19: return 0xC13
            case .hpmcounter20: return 0xC14
            case .hpmcounter21: return 0xC15
            case .hpmcounter22: return 0xC16
            case .hpmcounter23: return 0xC17
            case .hpmcounter24: return 0xC18
            case .hpmcounter25: return 0xC19
            case .hpmcounter26: return 0xC1A
            case .hpmcounter27: return 0xC1B
            case .hpmcounter28: return 0xC1C
            case .hpmcounter29: return 0xC1D
            case .hpmcounter30: return 0xC1E
            case .hpmcounter31: return 0xC1F
            case .vl: return 0xC20
            case .vtype: return 0xC21
            case .vlenb: return 0xC22
            case .cycleh: return 0xC80
            case .timeh: return 0xC81
            case .instreth: return 0xC82
            case .hpmcounter3h: return 0xC83
            case .hpmcounter4h: return 0xC84
            case .hpmcounter5h: return 0xC85
            case .hpmcounter6h: return 0xC86
            case .hpmcounter7h: return 0xC87
            case .hpmcounter8h: return 0xC88
            case .hpmcounter9h: return 0xC89
            case .hpmcounter10h: return 0xC8A
            case .hpmcounter11h: return 0xC8B
            case .hpmcounter12h: return 0xC8C
            case .hpmcounter13h: return 0xC8D
            case .hpmcounter14h: return 0xC8E
            case .hpmcounter15h: return 0xC8F
            case .hpmcounter16h: return 0xC90
            case .hpmcounter17h: return 0xC91
            case .hpmcounter18h: return 0xC92
            case .hpmcounter19h: return 0xC93
            case .hpmcounter20h: return 0xC94
            case .hpmcounter21h: return 0xC95
            case .hpmcounter22h: return 0xC96
            case .hpmcounter23h: return 0xC97
            case .hpmcounter24h: return 0xC98
            case .hpmcounter25h: return 0xC99
            case .hpmcounter26h: return 0xC9A
            case .hpmcounter27h: return 0xC9B
            case .hpmcounter28h: return 0xC9C
            case .hpmcounter29h: return 0xC9D
            case .hpmcounter30h: return 0xC9E
            case .hpmcounter31h: return 0xC9F
            case .scountovf: return 0xDA0
            case .hgeip: return 0xE12
            case .mvendorid: return 0xF11
            case .marchid: return 0xF12
            case .mimpid: return 0xF13
            case .mhartid: return 0xF14
            case .mconfigptr: return 0xF15
            }
        }
        
        static public func decode( _ encoding: UInt32 ) -> csr_sysreg? {
            switch encoding {
            case 0x0: return .ustatus
            case 0x1: return .fflags
            case 0x2: return .frm
            case 0x3: return .fcsr
            case 0x4: return .uie
            case 0x5: return .utvec
            case 0x8: return .vstart
            case 0x9: return .vxsat
            case 0xA: return .vxrm
            case 0xF: return .vcsr
            case 0x15: return .seed
            case 0x40: return .uscratch
            case 0x41: return .uepc
            case 0x42: return .ucause
            case 0x43: return .utval
            case 0x44: return .uip
            case 0x100: return .sstatus
            case 0x102: return .sedeleg
            case 0x103: return .sideleg
            case 0x104: return .sie
            case 0x105: return .stvec
            case 0x106: return .scounteren
            case 0x10A: return .senvcfg
            case 0x10C: return .sstateen0
            case 0x10D: return .sstateen1
            case 0x10E: return .sstateen2
            case 0x10F: return .sstateen3
            case 0x140: return .sscratch
            case 0x141: return .sepc
            case 0x142: return .scause
            case 0x143: return .stval
            case 0x144: return .sip
            case 0x14D: return .stimecmp
            case 0x15D: return .stimecmph
            case 0x180: return .satp
            case 0x200: return .vsstatus
            case 0x204: return .vsie
            case 0x205: return .vstvec
            case 0x240: return .vsscratch
            case 0x241: return .vsepc
            case 0x242: return .vscause
            case 0x243: return .vstval
            case 0x244: return .vsip
            case 0x24D: return .vstimecmp
            case 0x25D: return .vstimecmph
            case 0x280: return .vsatp
            case 0x300: return .mstatus
            case 0x301: return .misa
            case 0x302: return .medeleg
            case 0x303: return .mideleg
            case 0x304: return .mie
            case 0x305: return .mtvec
            case 0x306: return .mcounteren
            case 0x30A: return .menvcfg
            case 0x30C: return .mstateen0
            case 0x30D: return .mstateen1
            case 0x30E: return .mstateen2
            case 0x30F: return .mstateen3
            case 0x310: return .mstatush
            case 0x31A: return .menvcfgh
            case 0x31C: return .mstateen0h
            case 0x31D: return .mstateen1h
            case 0x31E: return .mstateen2h
            case 0x31F: return .mstateen3h
            case 0x320: return .mcountinhibit
            case 0x323: return .mhpmevent3
            case 0x324: return .mhpmevent4
            case 0x325: return .mhpmevent5
            case 0x326: return .mhpmevent6
            case 0x327: return .mhpmevent7
            case 0x328: return .mhpmevent8
            case 0x329: return .mhpmevent9
            case 0x32A: return .mhpmevent10
            case 0x32B: return .mhpmevent11
            case 0x32C: return .mhpmevent12
            case 0x32D: return .mhpmevent13
            case 0x32E: return .mhpmevent14
            case 0x32F: return .mhpmevent15
            case 0x330: return .mhpmevent16
            case 0x331: return .mhpmevent17
            case 0x332: return .mhpmevent18
            case 0x333: return .mhpmevent19
            case 0x334: return .mhpmevent20
            case 0x335: return .mhpmevent21
            case 0x336: return .mhpmevent22
            case 0x337: return .mhpmevent23
            case 0x338: return .mhpmevent24
            case 0x339: return .mhpmevent25
            case 0x33A: return .mhpmevent26
            case 0x33B: return .mhpmevent27
            case 0x33C: return .mhpmevent28
            case 0x33D: return .mhpmevent29
            case 0x33E: return .mhpmevent30
            case 0x33F: return .mhpmevent31
            case 0x340: return .mscratch
            case 0x341: return .mepc
            case 0x342: return .mcause
            case 0x343: return .mtval
            case 0x344: return .mip
            case 0x34A: return .mtinst
            case 0x34B: return .mtval2
            case 0x3A0: return .pmpcfg0
            case 0x3A1: return .pmpcfg1
            case 0x3A2: return .pmpcfg2
            case 0x3A3: return .pmpcfg3
            case 0x3A4: return .pmpcfg4
            case 0x3A5: return .pmpcfg5
            case 0x3A6: return .pmpcfg6
            case 0x3A7: return .pmpcfg7
            case 0x3A8: return .pmpcfg8
            case 0x3A9: return .pmpcfg9
            case 0x3AA: return .pmpcfg10
            case 0x3AB: return .pmpcfg11
            case 0x3AC: return .pmpcfg12
            case 0x3AD: return .pmpcfg13
            case 0x3AE: return .pmpcfg14
            case 0x3AF: return .pmpcfg15
            case 0x3B0: return .pmpaddr0
            case 0x3B1: return .pmpaddr1
            case 0x3B2: return .pmpaddr2
            case 0x3B3: return .pmpaddr3
            case 0x3B4: return .pmpaddr4
            case 0x3B5: return .pmpaddr5
            case 0x3B6: return .pmpaddr6
            case 0x3B7: return .pmpaddr7
            case 0x3B8: return .pmpaddr8
            case 0x3B9: return .pmpaddr9
            case 0x3BA: return .pmpaddr10
            case 0x3BB: return .pmpaddr11
            case 0x3BC: return .pmpaddr12
            case 0x3BD: return .pmpaddr13
            case 0x3BE: return .pmpaddr14
            case 0x3BF: return .pmpaddr15
            case 0x3C0: return .pmpaddr16
            case 0x3C1: return .pmpaddr17
            case 0x3C2: return .pmpaddr18
            case 0x3C3: return .pmpaddr19
            case 0x3C4: return .pmpaddr20
            case 0x3C5: return .pmpaddr21
            case 0x3C6: return .pmpaddr22
            case 0x3C7: return .pmpaddr23
            case 0x3C8: return .pmpaddr24
            case 0x3C9: return .pmpaddr25
            case 0x3CA: return .pmpaddr26
            case 0x3CB: return .pmpaddr27
            case 0x3CC: return .pmpaddr28
            case 0x3CD: return .pmpaddr29
            case 0x3CE: return .pmpaddr30
            case 0x3CF: return .pmpaddr31
            case 0x3D0: return .pmpaddr32
            case 0x3D1: return .pmpaddr33
            case 0x3D2: return .pmpaddr34
            case 0x3D3: return .pmpaddr35
            case 0x3D4: return .pmpaddr36
            case 0x3D5: return .pmpaddr37
            case 0x3D6: return .pmpaddr38
            case 0x3D7: return .pmpaddr39
            case 0x3D8: return .pmpaddr40
            case 0x3D9: return .pmpaddr41
            case 0x3DA: return .pmpaddr42
            case 0x3DB: return .pmpaddr43
            case 0x3DC: return .pmpaddr44
            case 0x3DD: return .pmpaddr45
            case 0x3DE: return .pmpaddr46
            case 0x3DF: return .pmpaddr47
            case 0x3E0: return .pmpaddr48
            case 0x3E1: return .pmpaddr49
            case 0x3E2: return .pmpaddr50
            case 0x3E3: return .pmpaddr51
            case 0x3E4: return .pmpaddr52
            case 0x3E5: return .pmpaddr53
            case 0x3E6: return .pmpaddr54
            case 0x3E7: return .pmpaddr55
            case 0x3E8: return .pmpaddr56
            case 0x3E9: return .pmpaddr57
            case 0x3EA: return .pmpaddr58
            case 0x3EB: return .pmpaddr59
            case 0x3EC: return .pmpaddr60
            case 0x3ED: return .pmpaddr61
            case 0x3EE: return .pmpaddr62
            case 0x3EF: return .pmpaddr63
            case 0x5A8: return .scontext
            case 0x600: return .hstatus
            case 0x602: return .hedeleg
            case 0x603: return .hideleg
            case 0x604: return .hie
            case 0x605: return .htimedelta
            case 0x606: return .hcounteren
            case 0x607: return .hgeie
            case 0x60A: return .henvcfg
            case 0x60C: return .hstateen0
            case 0x60D: return .hstateen1
            case 0x60E: return .hstateen2
            case 0x60F: return .hstateen3
            case 0x615: return .htimedeltah
            case 0x61A: return .henvcfgh
            case 0x61C: return .hstateen0h
            case 0x61D: return .hstateen1h
            case 0x61E: return .hstateen2h
            case 0x61F: return .hstateen3h
            case 0x643: return .htval
            case 0x644: return .hip
            case 0x645: return .hvip
            case 0x64A: return .htinst
            case 0x680: return .hgatp
            case 0x6A8: return .hcontext
            case 0x723: return .mhpmevent3h
            case 0x724: return .mhpmevent4h
            case 0x725: return .mhpmevent5h
            case 0x726: return .mhpmevent6h
            case 0x727: return .mhpmevent7h
            case 0x728: return .mhpmevent8h
            case 0x729: return .mhpmevent9h
            case 0x72A: return .mhpmevent10h
            case 0x72B: return .mhpmevent11h
            case 0x72C: return .mhpmevent12h
            case 0x72D: return .mhpmevent13h
            case 0x72E: return .mhpmevent14h
            case 0x72F: return .mhpmevent15h
            case 0x730: return .mhpmevent16h
            case 0x731: return .mhpmevent17h
            case 0x732: return .mhpmevent18h
            case 0x733: return .mhpmevent19h
            case 0x734: return .mhpmevent20h
            case 0x735: return .mhpmevent21h
            case 0x736: return .mhpmevent22h
            case 0x737: return .mhpmevent23h
            case 0x738: return .mhpmevent24h
            case 0x739: return .mhpmevent25h
            case 0x73A: return .mhpmevent26h
            case 0x73B: return .mhpmevent27h
            case 0x73C: return .mhpmevent28h
            case 0x73D: return .mhpmevent29h
            case 0x73E: return .mhpmevent30h
            case 0x73F: return .mhpmevent31h
            case 0x747: return .mseccfg
            case 0x757: return .mseccfgh
            case 0x7A0: return .tselect
            case 0x7A1: return .tdata1
            case 0x7A2: return .tdata2
            case 0x7A3: return .tdata3
            case 0x7A8: return .mcontext
            case 0x7B0: return .dcsr
            case 0x7B1: return .dpc
            case 0x7B2: return .dscratch0
            case 0x7B3: return .dscratch1
            case 0xB00: return .mcycle
            case 0xB02: return .minstret
            case 0xB03: return .mhpmcounter3
            case 0xB04: return .mhpmcounter4
            case 0xB05: return .mhpmcounter5
            case 0xB06: return .mhpmcounter6
            case 0xB07: return .mhpmcounter7
            case 0xB08: return .mhpmcounter8
            case 0xB09: return .mhpmcounter9
            case 0xB0A: return .mhpmcounter10
            case 0xB0B: return .mhpmcounter11
            case 0xB0C: return .mhpmcounter12
            case 0xB0D: return .mhpmcounter13
            case 0xB0E: return .mhpmcounter14
            case 0xB0F: return .mhpmcounter15
            case 0xB10: return .mhpmcounter16
            case 0xB11: return .mhpmcounter17
            case 0xB12: return .mhpmcounter18
            case 0xB13: return .mhpmcounter19
            case 0xB14: return .mhpmcounter20
            case 0xB15: return .mhpmcounter21
            case 0xB16: return .mhpmcounter22
            case 0xB17: return .mhpmcounter23
            case 0xB18: return .mhpmcounter24
            case 0xB19: return .mhpmcounter25
            case 0xB1A: return .mhpmcounter26
            case 0xB1B: return .mhpmcounter27
            case 0xB1C: return .mhpmcounter28
            case 0xB1D: return .mhpmcounter29
            case 0xB1E: return .mhpmcounter30
            case 0xB1F: return .mhpmcounter31
            case 0xB80: return .mcycleh
            case 0xB82: return .minstreth
            case 0xB83: return .mhpmcounter3h
            case 0xB84: return .mhpmcounter4h
            case 0xB85: return .mhpmcounter5h
            case 0xB86: return .mhpmcounter6h
            case 0xB87: return .mhpmcounter7h
            case 0xB88: return .mhpmcounter8h
            case 0xB89: return .mhpmcounter9h
            case 0xB8A: return .mhpmcounter10h
            case 0xB8B: return .mhpmcounter11h
            case 0xB8C: return .mhpmcounter12h
            case 0xB8D: return .mhpmcounter13h
            case 0xB8E: return .mhpmcounter14h
            case 0xB8F: return .mhpmcounter15h
            case 0xB90: return .mhpmcounter16h
            case 0xB91: return .mhpmcounter17h
            case 0xB92: return .mhpmcounter18h
            case 0xB93: return .mhpmcounter19h
            case 0xB94: return .mhpmcounter20h
            case 0xB95: return .mhpmcounter21h
            case 0xB96: return .mhpmcounter22h
            case 0xB97: return .mhpmcounter23h
            case 0xB98: return .mhpmcounter24h
            case 0xB99: return .mhpmcounter25h
            case 0xB9A: return .mhpmcounter26h
            case 0xB9B: return .mhpmcounter27h
            case 0xB9C: return .mhpmcounter28h
            case 0xB9D: return .mhpmcounter29h
            case 0xB9E: return .mhpmcounter30h
            case 0xB9F: return .mhpmcounter31h
            case 0xC00: return .cycle
            case 0xC01: return .time
            case 0xC02: return .instret
            case 0xC03: return .hpmcounter3
            case 0xC04: return .hpmcounter4
            case 0xC05: return .hpmcounter5
            case 0xC06: return .hpmcounter6
            case 0xC07: return .hpmcounter7
            case 0xC08: return .hpmcounter8
            case 0xC09: return .hpmcounter9
            case 0xC0A: return .hpmcounter10
            case 0xC0B: return .hpmcounter11
            case 0xC0C: return .hpmcounter12
            case 0xC0D: return .hpmcounter13
            case 0xC0E: return .hpmcounter14
            case 0xC0F: return .hpmcounter15
            case 0xC10: return .hpmcounter16
            case 0xC11: return .hpmcounter17
            case 0xC12: return .hpmcounter18
            case 0xC13: return .hpmcounter19
            case 0xC14: return .hpmcounter20
            case 0xC15: return .hpmcounter21
            case 0xC16: return .hpmcounter22
            case 0xC17: return .hpmcounter23
            case 0xC18: return .hpmcounter24
            case 0xC19: return .hpmcounter25
            case 0xC1A: return .hpmcounter26
            case 0xC1B: return .hpmcounter27
            case 0xC1C: return .hpmcounter28
            case 0xC1D: return .hpmcounter29
            case 0xC1E: return .hpmcounter30
            case 0xC1F: return .hpmcounter31
            case 0xC20: return .vl
            case 0xC21: return .vtype
            case 0xC22: return .vlenb
            case 0xC80: return .cycleh
            case 0xC81: return .timeh
            case 0xC82: return .instreth
            case 0xC83: return .hpmcounter3h
            case 0xC84: return .hpmcounter4h
            case 0xC85: return .hpmcounter5h
            case 0xC86: return .hpmcounter6h
            case 0xC87: return .hpmcounter7h
            case 0xC88: return .hpmcounter8h
            case 0xC89: return .hpmcounter9h
            case 0xC8A: return .hpmcounter10h
            case 0xC8B: return .hpmcounter11h
            case 0xC8C: return .hpmcounter12h
            case 0xC8D: return .hpmcounter13h
            case 0xC8E: return .hpmcounter14h
            case 0xC8F: return .hpmcounter15h
            case 0xC90: return .hpmcounter16h
            case 0xC91: return .hpmcounter17h
            case 0xC92: return .hpmcounter18h
            case 0xC93: return .hpmcounter19h
            case 0xC94: return .hpmcounter20h
            case 0xC95: return .hpmcounter21h
            case 0xC96: return .hpmcounter22h
            case 0xC97: return .hpmcounter23h
            case 0xC98: return .hpmcounter24h
            case 0xC99: return .hpmcounter25h
            case 0xC9A: return .hpmcounter26h
            case 0xC9B: return .hpmcounter27h
            case 0xC9C: return .hpmcounter28h
            case 0xC9D: return .hpmcounter29h
            case 0xC9E: return .hpmcounter30h
            case 0xC9F: return .hpmcounter31h
            case 0xDA0: return .scountovf
            case 0xE12: return .hgeip
            case 0xF11: return .mvendorid
            case 0xF12: return .marchid
            case 0xF13: return .mimpid
            case 0xF14: return .mhartid
            case 0xF15: return .mconfigptr
            default: return nil
            }
        }
        
//        static public func matchCsrSysreg( buffer: Buffer ) -> csr_sysreg? {
//            return nil
//        }
    }
}

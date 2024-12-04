//
//  Main.swift
//  Riscv32
//

let input: UInt32 = 46
let inputRegister = Riscv32.GPR.x10.rawValue
let outputRegister = Riscv32.GPR.x10.rawValue

let bytes: [UInt8] = [
    // _Z14fib_imperativei
    0x2a, 0x86,                     // 0:  c.mv    a2, a0
    0x89, 0x47,                     // 2:  c.li    a5, 2
    0x05, 0x45,                     // 4:  c.li    a0, 1
    0x63, 0xda, 0xc7, 0x00,         // 6:  bge    a5, a2, 20 # .L4
    0x05, 0x47,                     // 10:  c.li    a4, 1
    // .L3
    0xaa, 0x86,                     // 12:  c.mv    a3, a0
    0x85, 0x07,                     // 14:  c.addi    a5, 1
    0x3a, 0x95,                     // 16:  c.add    a0, a4
    0x36, 0x87,                     // 18:  c.mv    a4, a3
    0xe3, 0x1c, 0xf6, 0xfe,         // 20:  bne    a2, a5, -8 # .L3
    0x02, 0x90,                     // c.ebreak
    // .L4
    0x02, 0x90,                     // c.ebreak
]

//let bytes: [UInt8] = [
//    // _Z13fib_recursivei
//    0x75, 0x71,                     // 0:  c.addi16sp    -144, SP
//    0x26, 0xc3,                     // 2:  c.swsp    s1, 132(SP)
//    0x06, 0xc7,                     // 4:  c.swsp    ra, 140(SP)
//    0x22, 0xc5,                     // 6:  c.swsp    s0, 136(SP)
//    0x4a, 0xc1,                     // 8:  c.swsp    s2, 128(SP)
//    0xce, 0xde,                     // 10:  c.swsp    s3, 124(SP)
//    0xd2, 0xdc,                     // 12:  c.swsp    s4, 120(SP)
//    0xd6, 0xda,                     // 14:  c.swsp    s5, 116(SP)
//    0xda, 0xd8,                     // 16:  c.swsp    s6, 112(SP)
//    0xde, 0xd6,                     // 18:  c.swsp    s7, 108(SP)
//    0xe2, 0xd4,                     // 20:  c.swsp    s8, 104(SP)
//    0xe6, 0xd2,                     // 22:  c.swsp    s9, 100(SP)
//    0xea, 0xd0,                     // 24:  c.swsp    s10, 96(SP)
//    0xee, 0xce,                     // 26:  c.swsp    s11, 92(SP)
//    0x85, 0x47,                     // 28:  c.li    a5, 1
//    0xaa, 0x84,                     // 30:  c.mv    s1, a0
//    0x63, 0xd0, 0xa7, 0x2a,         // 32:  bge    a5, a0, 672 # .L30
//    0x93, 0x07, 0xf5, 0xff,         // 36:  addi    a5, a0, -1
//    0x13, 0xf7, 0xe7, 0xff,         // 40:  andi    a4, a5, -2
//    0x33, 0x0c, 0xe5, 0x40,         // 44:  sub    s8, a0, a4
//    0xe2, 0x89,                     // 48:  c.mv    s3, s8
//    0x01, 0x4b,                     // 50:  c.li    s6, 0
//    0x85, 0x45,                     // 52:  c.li    a1, 1
//    0x3e, 0x84,                     // 54:  c.mv    s0, a5
//    0x63, 0x81, 0x34, 0x29,         // 56:  beq    s1, 642 # .L35
//    // .L3
//    0x13, 0x8d, 0xe4, 0xff,         // 60:  addi    s10, s1, -2
//    0x93, 0x77, 0xed, 0xff,         // 64:  andi    a5, s10, -2
//    0xb3, 0x04, 0xf4, 0x40,         // 68:  sub    s1, s0, a5
//    0x81, 0x4a,                     // 72:  c.li    s5, 0
//    0x4e, 0x8c,                     // 74:  c.mv    s8, s3
//    0xea, 0x8b,                     // 76:  c.mv    s7, s10
//    // .L8
//    0x93, 0x0c, 0xf4, 0xff,         // 78:  addi    s9, s0, -1
//    0x63, 0x06, 0x94, 0x24,         // 82:  beq    s0, 588 # .L36
//    // .L6
//    0x79, 0x14,                     // 86:  c.addi    s0, -2
//    0x93, 0x77, 0xe4, 0xff,         // 88:  andi    a5, s0, -2
//    0x33, 0x89, 0xfc, 0x40,         // 92:  sub    s2, s9, a5
//    0x81, 0x49,                     // 96:  c.li    s3, 0
//    0x56, 0x8a,                     // 98:  c.mv    s4, s5
//    // .L11
//    0x93, 0x8d, 0xfc, 0xff,         // 100:  addi    s11, s9, -1
//    0x63, 0x80, 0x2c, 0x23,         // 104:  beq    s9, 544 # .L37
//    0x13, 0x8d, 0xec, 0xff,         // 108:  addi    s10, s9, -2
//    0x93, 0x77, 0xed, 0xff,         // 112:  andi    a5, s10, -2
//    0xb3, 0x86, 0xfd, 0x40,         // 116:  sub    a3, s11, a5
//    0x6e, 0x86,                     // 120:  c.mv    a2, s11
//    0xea, 0x8a,                     // 122:  c.mv    s5, s10
//    0xe2, 0x8d,                     // 124:  c.mv    s11, s8
//    0x81, 0x4c,                     // 126:  c.li    s9, 0
//    0x36, 0x8c,                     // 128:  c.mv    s8, a3
//    0x22, 0x8d,                     // 130:  c.mv    s10, s0
//    0xda, 0x86,                     // 132:  c.mv    a3, s6
//    0x4a, 0x8b,                     // 134:  c.mv    s6, s2
//    // .L14
//    0x13, 0x07, 0xf6, 0xff,         // 136:  addi    a4, a2, -1
//    0x63, 0x05, 0x86, 0x1f,         // 140:  beq    a2, 490 # .L38
//    0x79, 0x16,                     // 144:  c.addi    a2, -2
//    0x13, 0x79, 0xe6, 0xff,         // 146:  andi    s2, a2, -2
//    0x33, 0x09, 0x27, 0x41,         // 150:  sub    s2, a4, s2
//    0x01, 0x44,                     // 154:  c.li    s0, 0
//    0xb6, 0x8e,                     // 156:  c.mv    t4, a3
//    0x52, 0x83,                     // 158:  c.mv    t1, s4
//    0xce, 0x88,                     // 160:  c.mv    a7, s3
//    0xd6, 0x86,                     // 162:  c.mv    a3, s5
//    0x32, 0x8e,                     // 164:  c.mv    t3, a2
//    0xe2, 0x8a,                     // 166:  c.mv    s5, s8
//    0x5e, 0x86,                     // 168:  c.mv    a2, s7
//    0x66, 0x88,                     // 170:  c.mv    a6, s9
//    0xea, 0x8b,                     // 172:  c.mv    s7, s10
//    0xba, 0x89,                     // 174:  c.mv    s3, a4
//    0x5a, 0x8d,                     // 176:  c.mv    s10, s6
//    0x22, 0x8a,                     // 178:  c.mv    s4, s0
//    0x26, 0x8c,                     // 180:  c.mv    s8, s1
//    0x4a, 0x8b,                     // 182:  c.mv    s6, s2
//    // .L17
//    0x93, 0x87, 0xf9, 0xff,         // 184:  addi    a5, s3, -1
//    0x63, 0x8d, 0x69, 0x19,         // 188:  beq    s3, 410 # .L39
//    0xf9, 0x19,                     // 192:  c.addi    s3, -2
//    0x13, 0xf9, 0xe9, 0xff,         // 194:  andi    s2, s3, -2
//    0x33, 0x89, 0x27, 0x41,         // 198:  sub    s2, a5, s2
//    0x81, 0x4c,                     // 202:  c.li    s9, 0
//    // .L20
//    0x13, 0x84, 0xf7, 0xff,         // 204:  addi    s0, a5, -1
//    0x63, 0x87, 0x27, 0x15,         // 208:  beq    a5, 334 # .L40
//    0x13, 0x87, 0xe7, 0xff,         // 212:  addi    a4, a5, -2
//    0x13, 0x75, 0xe7, 0xff,         // 216:  andi    a0, a4, -2
//    0x93, 0x82, 0xd7, 0xff,         // 220:  addi    t0, a5, -3
//    0x93, 0x83, 0xb7, 0xff,         // 224:  addi    t2, a5, -5
//    0xb3, 0x07, 0xa4, 0x40,         // 228:  sub    a5, s0, a0
//    0x3e, 0xcc,                     // 232:  c.swsp    a5, 24(SP)
//    0xf6, 0x84,                     // 234:  c.mv    s1, t4
//    0x81, 0x47,                     // 236:  c.li    a5, 0
//    0x3a, 0xc4,                     // 238:  c.swsp    a4, 8(SP)
//    // .L23
//    0x62, 0x47,                     // 240:  c.lwsp    a4, 24(SP)
//    0x13, 0x0f, 0xf4, 0xff,         // 242:  addi    t5, s0, -1
//    0x63, 0x0b, 0xe4, 0x12,         // 246:  beq    s0, 310 # .L41
//    0x13, 0xf5, 0xe2, 0xff,         // 250:  andi    a0, t0, -2
//    0x33, 0x07, 0xaf, 0x40,         // 254:  sub    a4, t5, a0
//    0x3a, 0xc6,                     // 258:  c.swsp    a4, 12(SP)
//    0x1a, 0xc8,                     // 260:  c.swsp    t1, 16(SP)
//    0x7a, 0x85,                     // 262:  c.mv    a0, t5
//    0xde, 0x8f,                     // 264:  c.mv    t6, s7
//    0x32, 0x8f,                     // 266:  c.mv    t5, a2
//    0xda, 0x8b,                     // 268:  c.mv    s7, s6
//    0x4e, 0x86,                     // 270:  c.mv    a2, s3
//    0x56, 0x8b,                     // 272:  c.mv    s6, s5
//    0x1e, 0x87,                     // 274:  c.mv    a4, t2
//    0x81, 0x4e,                     // 276:  c.li    t4, 0
//    0x3e, 0xca,                     // 278:  c.swsp    a5, 20(SP)
//    0x4a, 0x83,                     // 280:  c.mv    t1, s2
//    0xa2, 0x8a,                     // 282:  c.mv    s5, s0
//    0xa6, 0x89,                     // 284:  c.mv    s3, s1
//    // .L26
//    0xb2, 0x47,                     // 286:  c.lwsp    a5, 12(SP)
//    0x63, 0x0b, 0xf5, 0x10,         // 288:  beq    a0, 278 # .L42
//    0x93, 0x07, 0xe5, 0xff,         // 292:  addi    a5, a0, -2
//    0x13, 0x74, 0xe7, 0xff,         // 296:  andi    s0, a4, -2
//    0x71, 0x15,                     // 300:  c.addi    a0, -4
//    0x3e, 0x89,                     // 302:  c.mv    s2, a5
//    0x33, 0x04, 0x85, 0x40,         // 304:  sub    s0, a0, s0
//    0x81, 0x44,                     // 308:  c.li    s1, 0
//    // .L28
//    0x4a, 0x85,                     // 310:  c.mv    a0, s2
//    0xbe, 0xc6,                     // 312:  c.swsp    a5, 76(SP)
//    0xfa, 0xc4,                     // 314:  c.swsp    t5, 72(SP)
//    0xb2, 0xc2,                     // 316:  c.swsp    a2, 68(SP)
//    0xf2, 0xc0,                     // 318:  c.swsp    t3, 64(SP)
//    0x36, 0xde,                     // 320:  c.swsp    a3, 60(SP)
//    0x7e, 0xdc,                     // 322:  c.swsp    t6, 56(SP)
//    0x3a, 0xda,                     // 324:  c.swsp    a4, 52(SP)
//    0x16, 0xd8,                     // 326:  c.swsp    t0, 48(SP)
//    0x1e, 0xd6,                     // 328:  c.swsp    t2, 44(SP)
//    0x1a, 0xd4,                     // 330:  c.swsp    t1, 40(SP)
//    0x76, 0xd2,                     // 332:  c.swsp    t4, 36(SP)
//    0x42, 0xd0,                     // 334:  c.swsp    a6, 32(SP)
//    0x46, 0xce,                     // 336:  c.swsp    a7, 28(SP)
//    0x79, 0x19,                     // 338:  c.addi    s2, -2
//    0x97, 0x00, 0x00, 0x00,         // 340:  auipc    ra, 0
//    0xe7, 0x80, 0x00, 0x00,         // 344:  jalr    ra, 0(ra) # .L28+0x30
//    0xf2, 0x48,                     // 348:  c.lwsp    a7, 28(SP)
//    0x02, 0x58,                     // 350:  c.lwsp    a6, 32(SP)
//    0x92, 0x5e,                     // 352:  c.lwsp    t4, 36(SP)
//    0x22, 0x53,                     // 354:  c.lwsp    t1, 40(SP)
//    0xb2, 0x53,                     // 356:  c.lwsp    t2, 44(SP)
//    0xc2, 0x52,                     // 358:  c.lwsp    t0, 48(SP)
//    0x52, 0x57,                     // 360:  c.lwsp    a4, 52(SP)
//    0xe2, 0x5f,                     // 362:  c.lwsp    t6, 56(SP)
//    0xf2, 0x56,                     // 364:  c.lwsp    a3, 60(SP)
//    0x06, 0x4e,                     // 366:  c.lwsp    t3, 64(SP)
//    0x16, 0x46,                     // 368:  c.lwsp    a2, 68(SP)
//    0x26, 0x4f,                     // 370:  c.lwsp    t5, 72(SP)
//    0xb6, 0x47,                     // 372:  c.lwsp    a5, 76(SP)
//    0xaa, 0x94,                     // 374:  c.add    s1, a0
//    0x85, 0x45,                     // 376:  c.li    a1, 1
//    0xe3, 0x1e, 0x24, 0xfb,         // 378:  bne    s0, s2, -68 # .L28
//    0x13, 0x74, 0x17, 0x00,         // 382:  andi    s0, a4, 1
//    0x26, 0x94,                     // 386:  c.add    s0, s1
//    0x3e, 0x85,                     // 388:  c.mv    a0, a5
//    0xa2, 0x9e,                     // 390:  c.add    t4, s0
//    0x79, 0x17,                     // 392:  c.addi    a4, -2
//    0xe3, 0xca, 0xf5, 0xf8,         // 394:  blt    a1, a5, -108 # .L26
//    0x1a, 0x89,                     // 398:  c.mv    s2, t1
//    0xd2, 0x47,                     // 400:  c.lwsp    a5, 20(SP)
//    0x42, 0x43,                     // 402:  c.lwsp    t1, 16(SP)
//    0xce, 0x84,                     // 404:  c.mv    s1, s3
//    0x56, 0x84,                     // 406:  c.mv    s0, s5
//    0xb2, 0x89,                     // 408:  c.mv    s3, a2
//    0xda, 0x8a,                     // 410:  c.mv    s5, s6
//    0x7a, 0x86,                     // 412:  c.mv    a2, t5
//    0x5e, 0x8b,                     // 414:  c.mv    s6, s7
//    0x2a, 0x8f,                     // 416:  c.mv    t5, a0
//    0xfe, 0x8b,                     // 418:  c.mv    s7, t6
//    // .L25
//    0x76, 0x9f,                     // 420:  c.add    t5, t4
//    0x79, 0x14,                     // 422:  c.addi    s0, -2
//    0xfa, 0x97,                     // 424:  c.add    a5, t5
//    0xf9, 0x12,                     // 426:  c.addi    t0, -2
//    0xf9, 0x13,                     // 428:  c.addi    t2, -2
//    0xe3, 0xc1, 0x85, 0xf4,         // 430:  blt    a1, s0, -190 # .L23
//    0x22, 0x47,                     // 434:  c.lwsp    a4, 8(SP)
//    0xa6, 0x8e,                     // 436:  c.mv    t4, s1
//    // .L22
//    0x3e, 0x94,                     // 438:  c.add    s0, a5
//    0xa2, 0x9c,                     // 440:  c.add    s9, s0
//    0xba, 0x87,                     // 442:  c.mv    a5, a4
//    0xe3, 0xc8, 0xe5, 0xf0,         // 444:  blt    a1, a4, -240 # .L20
//    0xe6, 0x97,                     // 448:  c.add    a5, s9
//    0x3e, 0x9a,                     // 450:  c.add    s4, a5
//    0xe3, 0xca, 0x35, 0xef,         // 452:  blt    a1, s3, -268 # .L17
//    // .L44
//    0x4e, 0x87,                     // 456:  c.mv    a4, s3
//    0x52, 0x84,                     // 458:  c.mv    s0, s4
//    0xe2, 0x84,                     // 460:  c.mv    s1, s8
//    0x6a, 0x8b,                     // 462:  c.mv    s6, s10
//    0x56, 0x8c,                     // 464:  c.mv    s8, s5
//    0x5e, 0x8d,                     // 466:  c.mv    s10, s7
//    0xb6, 0x8a,                     // 468:  c.mv    s5, a3
//    0xb2, 0x8b,                     // 470:  c.mv    s7, a2
//    0xc2, 0x8c,                     // 472:  c.mv    s9, a6
//    0xc6, 0x89,                     // 474:  c.mv    s3, a7
//    0x1a, 0x8a,                     // 476:  c.mv    s4, t1
//    0xf6, 0x86,                     // 478:  c.mv    a3, t4
//    0x72, 0x86,                     // 480:  c.mv    a2, t3
//    // .L16
//    0x22, 0x97,                     // 482:  c.add    a4, s0
//    0xba, 0x9c,                     // 484:  c.add    s9, a4
//    0xe3, 0xc1, 0xc5, 0xea,         // 486:  blt    a1, a2, -350 # .L14
//    0x6e, 0x8c,                     // 490:  c.mv    s8, s11
//    0x5a, 0x89,                     // 492:  c.mv    s2, s6
//    0x6a, 0x84,                     // 494:  c.mv    s0, s10
//    0xb2, 0x8d,                     // 496:  c.mv    s11, a2
//    0x36, 0x8b,                     // 498:  c.mv    s6, a3
//    0x56, 0x8d,                     // 500:  c.mv    s10, s5
//    // .L13
//    0xe6, 0x9d,                     // 502:  c.add    s11, s9
//    0xee, 0x99,                     // 504:  c.add    s3, s11
//    0xea, 0x8c,                     // 506:  c.mv    s9, s10
//    0xe3, 0xc4, 0xa5, 0xe7,         // 508:  blt    a1, s10, -408 # .L11
//    0xd2, 0x8a,                     // 512:  c.mv    s5, s4
//    0xce, 0x9c,                     // 514:  c.add    s9, s3
//    0xe6, 0x9a,                     // 516:  c.add    s5, s9
//    0xe3, 0xc4, 0x85, 0xe4,         // 518:  blt    a1, s0, -440 # .L8
//    // .L45
//    0x5e, 0x8d,                     // 522:  c.mv    s10, s7
//    0x56, 0x94,                     // 524:  c.add    s0, s5
//    0xe2, 0x89,                     // 526:  c.mv    s3, s8
//    0xea, 0x84,                     // 528:  c.mv    s1, s10
//    0x22, 0x9b,                     // 530:  c.add    s6, s0
//    0x63, 0xce, 0xa5, 0x09,         // 532:  blt    a1, s10, 156 # .L43
//    // .L31
//    0xb3, 0x04, 0x6d, 0x01,         // 536:  add    s1, s10, s6
//    0x55, 0xa0,                     // 540:  c.j    164 # .L30
//    // .L40
//    0xf9, 0x17,                     // 542:  c.addi    a5, -2
//    0xa2, 0x9c,                     // 544:  c.add    s9, s0
//    0xe6, 0x97,                     // 546:  c.add    a5, s9
//    0x3e, 0x9a,                     // 548:  c.add    s4, a5
//    0xe3, 0xc9, 0x35, 0xe9,         // 550:  blt    a1, s3, -366 # .L17
//    0x79, 0xbf,                     // 554:  c.j    -98 # .L44
//    // .L41
//    0x22, 0x47,                     // 556:  c.lwsp    a4, 8(SP)
//    0xa6, 0x8e,                     // 558:  c.mv    t4, s1
//    0x79, 0x14,                     // 560:  c.addi    s0, -2
//    0xfa, 0x97,                     // 562:  c.add    a5, t5
//    0x49, 0xb7,                     // 564:  c.j    -126 # .L22
//    // .L42
//    0x56, 0x84,                     // 566:  c.mv    s0, s5
//    0xda, 0x8a,                     // 568:  c.mv    s5, s6
//    0x5e, 0x8b,                     // 570:  c.mv    s6, s7
//    0xfe, 0x8b,                     // 572:  c.mv    s7, t6
//    0x93, 0x0f, 0xf5, 0xff,         // 574:  addi    t6, a0, -1
//    0xce, 0x84,                     // 578:  c.mv    s1, s3
//    0x1a, 0x89,                     // 580:  c.mv    s2, t1
//    0xb2, 0x89,                     // 582:  c.mv    s3, a2
//    0xd2, 0x47,                     // 584:  c.lwsp    a5, 20(SP)
//    0x7a, 0x86,                     // 586:  c.mv    a2, t5
//    0x42, 0x43,                     // 588:  c.lwsp    t1, 16(SP)
//    0x13, 0x0f, 0xe5, 0xff,         // 590:  addi    t5, a0, -2
//    0xfe, 0x9e,                     // 594:  c.add    t4, t6
//    0x81, 0xbf,                     // 596:  c.j    -176 # .L25
//    // .L39
//    0x4e, 0x87,                     // 598:  c.mv    a4, s3
//    0x52, 0x84,                     // 600:  c.mv    s0, s4
//    0xe2, 0x84,                     // 602:  c.mv    s1, s8
//    0x6a, 0x8b,                     // 604:  c.mv    s6, s10
//    0x56, 0x8c,                     // 606:  c.mv    s8, s5
//    0x5e, 0x8d,                     // 608:  c.mv    s10, s7
//    0xb6, 0x8a,                     // 610:  c.mv    s5, a3
//    0xb2, 0x8b,                     // 612:  c.mv    s7, a2
//    0xc2, 0x8c,                     // 614:  c.mv    s9, a6
//    0xc6, 0x89,                     // 616:  c.mv    s3, a7
//    0x1a, 0x8a,                     // 618:  c.mv    s4, t1
//    0xf6, 0x86,                     // 620:  c.mv    a3, t4
//    0x72, 0x86,                     // 622:  c.mv    a2, t3
//    0x79, 0x17,                     // 624:  c.addi    a4, -2
//    0x3e, 0x94,                     // 626:  c.add    s0, a5
//    0xbd, 0xb7,                     // 628:  c.j    -146 # .L16
//    // .L38
//    0x6e, 0x8c,                     // 630:  c.mv    s8, s11
//    0x5a, 0x89,                     // 632:  c.mv    s2, s6
//    0x6a, 0x84,                     // 634:  c.mv    s0, s10
//    0x36, 0x8b,                     // 636:  c.mv    s6, a3
//    0x56, 0x8d,                     // 638:  c.mv    s10, s5
//    0x93, 0x0d, 0xe6, 0xff,         // 640:  addi    s11, a2, -2
//    0xba, 0x9c,                     // 644:  c.add    s9, a4
//    0x85, 0xbf,                     // 646:  c.j    -144 # .L13
//    // .L37
//    0xf9, 0x1c,                     // 648:  c.addi    s9, -2
//    0xee, 0x99,                     // 650:  c.add    s3, s11
//    0xd2, 0x8a,                     // 652:  c.mv    s5, s4
//    0xce, 0x9c,                     // 654:  c.add    s9, s3
//    0xe6, 0x9a,                     // 656:  c.add    s5, s9
//    0xe3, 0xdc, 0x85, 0xf6,         // 658:  bge    a1, s0, -136 # .L45
//    0x93, 0x0c, 0xf4, 0xff,         // 662:  addi    s9, s0, -1
//    0xe3, 0x1e, 0x94, 0xda,         // 666:  bne    s0, s1, -580 # .L6
//    // .L36
//    0x79, 0x14,                     // 670:  c.addi    s0, -2
//    0xe6, 0x9a,                     // 672:  c.add    s5, s9
//    0x5e, 0x8d,                     // 674:  c.mv    s10, s7
//    0x56, 0x94,                     // 676:  c.add    s0, s5
//    0xe2, 0x89,                     // 678:  c.mv    s3, s8
//    0xea, 0x84,                     // 680:  c.mv    s1, s10
//    0x22, 0x9b,                     // 682:  c.add    s6, s0
//    0xe3, 0xd6, 0xa5, 0xf7,         // 684:  bge    a1, s10, -148 # .L31
//    // .L43
//    0x93, 0x07, 0xfd, 0xff,         // 688:  addi    a5, s10, -1
//    0x3e, 0x84,                     // 692:  c.mv    s0, a5
//    0xe3, 0x93, 0x34, 0xd9,         // 694:  bne    s1, s3, -634 # .L3
//    // .L35
//    0xf9, 0x14,                     // 698:  c.addi    s1, -2
//    0xda, 0x97,                     // 700:  c.add    a5, s6
//    0xbe, 0x94,                     // 702:  c.add    s1, a5
//    // .L30
//    0xba, 0x40,                     // 704:  c.lwsp    ra, 140(SP)
//    0x2a, 0x44,                     // 706:  c.lwsp    s0, 136(SP)
//    0x0a, 0x49,                     // 708:  c.lwsp    s2, 128(SP)
//    0xf6, 0x59,                     // 710:  c.lwsp    s3, 124(SP)
//    0x66, 0x5a,                     // 712:  c.lwsp    s4, 120(SP)
//    0xd6, 0x5a,                     // 714:  c.lwsp    s5, 116(SP)
//    0x46, 0x5b,                     // 716:  c.lwsp    s6, 112(SP)
//    0xb6, 0x5b,                     // 718:  c.lwsp    s7, 108(SP)
//    0x26, 0x5c,                     // 720:  c.lwsp    s8, 104(SP)
//    0x96, 0x5c,                     // 722:  c.lwsp    s9, 100(SP)
//    0x06, 0x5d,                     // 724:  c.lwsp    s10, 96(SP)
//    0xf6, 0x4d,                     // 726:  c.lwsp    s11, 92(SP)
//    0x26, 0x85,                     // 728:  c.mv    a0, s1
//    0x9a, 0x44,                     // 730:  c.lwsp    s1, 132(SP)
//    0x49, 0x61,                     // 732:  c.addi16sp    144, SP
//    0x02, 0x90,                     // c.ebreak
//]

@main
public class Main {
    static public func main() {
        func fibImperative( _ n: UInt32 ) -> UInt32 {
            var fib1: UInt32 = 1
            var fib2: UInt32 = 1
            var fibonacci: UInt32 = fib1
            var i: UInt32 = 2
            while ( i < n ) {
                fibonacci = fib1 + fib2
                fib1 = fib2
                fib2 = fibonacci
                i = i + 1
            }
            return fibonacci
        }
        
        print( "input: \( input )" )
//        print( "expected: \( fibImperative( input ) )" )
        
        let memory = Memory( bytes: bytes )
        let simulator = Riscv32.Simulator( memory: memory,
                                           instructionSize: bytes.count,
                                           verbose: false )
        
        simulator.x[ inputRegister ] = input
        
        simulator.executeInstructions()
        
        print( "result: \( simulator.x[ outputRegister ] )" )
    }
}

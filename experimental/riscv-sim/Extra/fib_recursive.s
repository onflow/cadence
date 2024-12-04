
/Users/jsprowes/artificer/InternalCadence/tmp/fib_recursive.o:     file format elf32-littleriscv


Disassembly of section .text:

00000000 <_Z13fib_recursivei>:
   0:	7175                	add	sp,sp,-144
   2:	c326                	sw	s1,132(sp)
   4:	c706                	sw	ra,140(sp)
   6:	c522                	sw	s0,136(sp)
   8:	c14a                	sw	s2,128(sp)
   a:	dece                	sw	s3,124(sp)
   c:	dcd2                	sw	s4,120(sp)
   e:	dad6                	sw	s5,116(sp)
  10:	d8da                	sw	s6,112(sp)
  12:	d6de                	sw	s7,108(sp)
  14:	d4e2                	sw	s8,104(sp)
  16:	d2e6                	sw	s9,100(sp)
  18:	d0ea                	sw	s10,96(sp)
  1a:	ceee                	sw	s11,92(sp)
  1c:	4785                	li	a5,1
  1e:	84aa                	mv	s1,a0
  20:	2aa7d063          	bge	a5,a0,2c0 <.L30>
  24:	fff50793          	add	a5,a0,-1
  28:	ffe7f713          	and	a4,a5,-2
  2c:	40e50c33          	sub	s8,a0,a4
  30:	89e2                	mv	s3,s8
  32:	4b01                	li	s6,0
  34:	4585                	li	a1,1
  36:	843e                	mv	s0,a5
  38:	29348163          	beq	s1,s3,2ba <.L35>

0000003c <.L3>:
  3c:	ffe48d13          	add	s10,s1,-2
  40:	ffed7793          	and	a5,s10,-2
  44:	40f404b3          	sub	s1,s0,a5
  48:	4a81                	li	s5,0
  4a:	8c4e                	mv	s8,s3
  4c:	8bea                	mv	s7,s10

0000004e <.L8>:
  4e:	fff40c93          	add	s9,s0,-1
  52:	24940663          	beq	s0,s1,29e <.L36>

00000056 <.L6>:
  56:	1479                	add	s0,s0,-2
  58:	ffe47793          	and	a5,s0,-2
  5c:	40fc8933          	sub	s2,s9,a5
  60:	4981                	li	s3,0
  62:	8a56                	mv	s4,s5

00000064 <.L11>:
  64:	fffc8d93          	add	s11,s9,-1
  68:	232c8063          	beq	s9,s2,288 <.L37>
  6c:	ffec8d13          	add	s10,s9,-2
  70:	ffed7793          	and	a5,s10,-2
  74:	40fd86b3          	sub	a3,s11,a5
  78:	866e                	mv	a2,s11
  7a:	8aea                	mv	s5,s10
  7c:	8de2                	mv	s11,s8
  7e:	4c81                	li	s9,0
  80:	8c36                	mv	s8,a3
  82:	8d22                	mv	s10,s0
  84:	86da                	mv	a3,s6
  86:	8b4a                	mv	s6,s2

00000088 <.L14>:
  88:	fff60713          	add	a4,a2,-1
  8c:	1f860563          	beq	a2,s8,276 <.L38>
  90:	1679                	add	a2,a2,-2
  92:	ffe67913          	and	s2,a2,-2
  96:	41270933          	sub	s2,a4,s2
  9a:	4401                	li	s0,0
  9c:	8eb6                	mv	t4,a3
  9e:	8352                	mv	t1,s4
  a0:	88ce                	mv	a7,s3
  a2:	86d6                	mv	a3,s5
  a4:	8e32                	mv	t3,a2
  a6:	8ae2                	mv	s5,s8
  a8:	865e                	mv	a2,s7
  aa:	8866                	mv	a6,s9
  ac:	8bea                	mv	s7,s10
  ae:	89ba                	mv	s3,a4
  b0:	8d5a                	mv	s10,s6
  b2:	8a22                	mv	s4,s0
  b4:	8c26                	mv	s8,s1
  b6:	8b4a                	mv	s6,s2

000000b8 <.L17>:
  b8:	fff98793          	add	a5,s3,-1
  bc:	19698d63          	beq	s3,s6,256 <.L39>
  c0:	19f9                	add	s3,s3,-2
  c2:	ffe9f913          	and	s2,s3,-2
  c6:	41278933          	sub	s2,a5,s2
  ca:	4c81                	li	s9,0

000000cc <.L20>:
  cc:	fff78413          	add	s0,a5,-1
  d0:	15278763          	beq	a5,s2,21e <.L40>
  d4:	ffe78713          	add	a4,a5,-2
  d8:	ffe77513          	and	a0,a4,-2
  dc:	ffd78293          	add	t0,a5,-3
  e0:	ffb78393          	add	t2,a5,-5
  e4:	40a407b3          	sub	a5,s0,a0
  e8:	cc3e                	sw	a5,24(sp)
  ea:	84f6                	mv	s1,t4
  ec:	4781                	li	a5,0
  ee:	c43a                	sw	a4,8(sp)

000000f0 <.L23>:
  f0:	4762                	lw	a4,24(sp)
  f2:	fff40f13          	add	t5,s0,-1
  f6:	12e40b63          	beq	s0,a4,22c <.L41>
  fa:	ffe2f513          	and	a0,t0,-2
  fe:	40af0733          	sub	a4,t5,a0
 102:	c63a                	sw	a4,12(sp)
 104:	c81a                	sw	t1,16(sp)
 106:	857a                	mv	a0,t5
 108:	8fde                	mv	t6,s7
 10a:	8f32                	mv	t5,a2
 10c:	8bda                	mv	s7,s6
 10e:	864e                	mv	a2,s3
 110:	8b56                	mv	s6,s5
 112:	871e                	mv	a4,t2
 114:	4e81                	li	t4,0
 116:	ca3e                	sw	a5,20(sp)
 118:	834a                	mv	t1,s2
 11a:	8aa2                	mv	s5,s0
 11c:	89a6                	mv	s3,s1

0000011e <.L26>:
 11e:	47b2                	lw	a5,12(sp)
 120:	10f50b63          	beq	a0,a5,236 <.L42>
 124:	ffe50793          	add	a5,a0,-2
 128:	ffe77413          	and	s0,a4,-2
 12c:	1571                	add	a0,a0,-4
 12e:	893e                	mv	s2,a5
 130:	40850433          	sub	s0,a0,s0
 134:	4481                	li	s1,0

00000136 <.L28>:
 136:	854a                	mv	a0,s2
 138:	c6be                	sw	a5,76(sp)
 13a:	c4fa                	sw	t5,72(sp)
 13c:	c2b2                	sw	a2,68(sp)
 13e:	c0f2                	sw	t3,64(sp)
 140:	de36                	sw	a3,60(sp)
 142:	dc7e                	sw	t6,56(sp)
 144:	da3a                	sw	a4,52(sp)
 146:	d816                	sw	t0,48(sp)
 148:	d61e                	sw	t2,44(sp)
 14a:	d41a                	sw	t1,40(sp)
 14c:	d276                	sw	t4,36(sp)
 14e:	d042                	sw	a6,32(sp)
 150:	ce46                	sw	a7,28(sp)
 152:	1979                	add	s2,s2,-2
 154:	00000097          	auipc	ra,0x0
 158:	000080e7          	jalr	ra # 154 <.L28+0x1e>
 15c:	48f2                	lw	a7,28(sp)
 15e:	5802                	lw	a6,32(sp)
 160:	5e92                	lw	t4,36(sp)
 162:	5322                	lw	t1,40(sp)
 164:	53b2                	lw	t2,44(sp)
 166:	52c2                	lw	t0,48(sp)
 168:	5752                	lw	a4,52(sp)
 16a:	5fe2                	lw	t6,56(sp)
 16c:	56f2                	lw	a3,60(sp)
 16e:	4e06                	lw	t3,64(sp)
 170:	4616                	lw	a2,68(sp)
 172:	4f26                	lw	t5,72(sp)
 174:	47b6                	lw	a5,76(sp)
 176:	94aa                	add	s1,s1,a0
 178:	4585                	li	a1,1
 17a:	fb241ee3          	bne	s0,s2,136 <.L28>
 17e:	00177413          	and	s0,a4,1
 182:	9426                	add	s0,s0,s1
 184:	853e                	mv	a0,a5
 186:	9ea2                	add	t4,t4,s0
 188:	1779                	add	a4,a4,-2
 18a:	f8f5cae3          	blt	a1,a5,11e <.L26>
 18e:	891a                	mv	s2,t1
 190:	47d2                	lw	a5,20(sp)
 192:	4342                	lw	t1,16(sp)
 194:	84ce                	mv	s1,s3
 196:	8456                	mv	s0,s5
 198:	89b2                	mv	s3,a2
 19a:	8ada                	mv	s5,s6
 19c:	867a                	mv	a2,t5
 19e:	8b5e                	mv	s6,s7
 1a0:	8f2a                	mv	t5,a0
 1a2:	8bfe                	mv	s7,t6

000001a4 <.L25>:
 1a4:	9f76                	add	t5,t5,t4
 1a6:	1479                	add	s0,s0,-2
 1a8:	97fa                	add	a5,a5,t5
 1aa:	12f9                	add	t0,t0,-2
 1ac:	13f9                	add	t2,t2,-2
 1ae:	f485c1e3          	blt	a1,s0,f0 <.L23>
 1b2:	4722                	lw	a4,8(sp)
 1b4:	8ea6                	mv	t4,s1

000001b6 <.L22>:
 1b6:	943e                	add	s0,s0,a5
 1b8:	9ca2                	add	s9,s9,s0
 1ba:	87ba                	mv	a5,a4
 1bc:	f0e5c8e3          	blt	a1,a4,cc <.L20>
 1c0:	97e6                	add	a5,a5,s9
 1c2:	9a3e                	add	s4,s4,a5
 1c4:	ef35cae3          	blt	a1,s3,b8 <.L17>

000001c8 <.L44>:
 1c8:	874e                	mv	a4,s3
 1ca:	8452                	mv	s0,s4
 1cc:	84e2                	mv	s1,s8
 1ce:	8b6a                	mv	s6,s10
 1d0:	8c56                	mv	s8,s5
 1d2:	8d5e                	mv	s10,s7
 1d4:	8ab6                	mv	s5,a3
 1d6:	8bb2                	mv	s7,a2
 1d8:	8cc2                	mv	s9,a6
 1da:	89c6                	mv	s3,a7
 1dc:	8a1a                	mv	s4,t1
 1de:	86f6                	mv	a3,t4
 1e0:	8672                	mv	a2,t3

000001e2 <.L16>:
 1e2:	9722                	add	a4,a4,s0
 1e4:	9cba                	add	s9,s9,a4
 1e6:	eac5c1e3          	blt	a1,a2,88 <.L14>
 1ea:	8c6e                	mv	s8,s11
 1ec:	895a                	mv	s2,s6
 1ee:	846a                	mv	s0,s10
 1f0:	8db2                	mv	s11,a2
 1f2:	8b36                	mv	s6,a3
 1f4:	8d56                	mv	s10,s5

000001f6 <.L13>:
 1f6:	9de6                	add	s11,s11,s9
 1f8:	99ee                	add	s3,s3,s11
 1fa:	8cea                	mv	s9,s10
 1fc:	e7a5c4e3          	blt	a1,s10,64 <.L11>
 200:	8ad2                	mv	s5,s4
 202:	9cce                	add	s9,s9,s3
 204:	9ae6                	add	s5,s5,s9
 206:	e485c4e3          	blt	a1,s0,4e <.L8>

0000020a <.L45>:
 20a:	8d5e                	mv	s10,s7
 20c:	9456                	add	s0,s0,s5
 20e:	89e2                	mv	s3,s8
 210:	84ea                	mv	s1,s10
 212:	9b22                	add	s6,s6,s0
 214:	09a5ce63          	blt	a1,s10,2b0 <.L43>

00000218 <.L31>:
 218:	016d04b3          	add	s1,s10,s6
 21c:	a055                	j	2c0 <.L30>

0000021e <.L40>:
 21e:	17f9                	add	a5,a5,-2
 220:	9ca2                	add	s9,s9,s0
 222:	97e6                	add	a5,a5,s9
 224:	9a3e                	add	s4,s4,a5
 226:	e935c9e3          	blt	a1,s3,b8 <.L17>
 22a:	bf79                	j	1c8 <.L44>

0000022c <.L41>:
 22c:	4722                	lw	a4,8(sp)
 22e:	8ea6                	mv	t4,s1
 230:	1479                	add	s0,s0,-2
 232:	97fa                	add	a5,a5,t5
 234:	b749                	j	1b6 <.L22>

00000236 <.L42>:
 236:	8456                	mv	s0,s5
 238:	8ada                	mv	s5,s6
 23a:	8b5e                	mv	s6,s7
 23c:	8bfe                	mv	s7,t6
 23e:	fff50f93          	add	t6,a0,-1
 242:	84ce                	mv	s1,s3
 244:	891a                	mv	s2,t1
 246:	89b2                	mv	s3,a2
 248:	47d2                	lw	a5,20(sp)
 24a:	867a                	mv	a2,t5
 24c:	4342                	lw	t1,16(sp)
 24e:	ffe50f13          	add	t5,a0,-2
 252:	9efe                	add	t4,t4,t6
 254:	bf81                	j	1a4 <.L25>

00000256 <.L39>:
 256:	874e                	mv	a4,s3
 258:	8452                	mv	s0,s4
 25a:	84e2                	mv	s1,s8
 25c:	8b6a                	mv	s6,s10
 25e:	8c56                	mv	s8,s5
 260:	8d5e                	mv	s10,s7
 262:	8ab6                	mv	s5,a3
 264:	8bb2                	mv	s7,a2
 266:	8cc2                	mv	s9,a6
 268:	89c6                	mv	s3,a7
 26a:	8a1a                	mv	s4,t1
 26c:	86f6                	mv	a3,t4
 26e:	8672                	mv	a2,t3
 270:	1779                	add	a4,a4,-2
 272:	943e                	add	s0,s0,a5
 274:	b7bd                	j	1e2 <.L16>

00000276 <.L38>:
 276:	8c6e                	mv	s8,s11
 278:	895a                	mv	s2,s6
 27a:	846a                	mv	s0,s10
 27c:	8b36                	mv	s6,a3
 27e:	8d56                	mv	s10,s5
 280:	ffe60d93          	add	s11,a2,-2
 284:	9cba                	add	s9,s9,a4
 286:	bf85                	j	1f6 <.L13>

00000288 <.L37>:
 288:	1cf9                	add	s9,s9,-2
 28a:	99ee                	add	s3,s3,s11
 28c:	8ad2                	mv	s5,s4
 28e:	9cce                	add	s9,s9,s3
 290:	9ae6                	add	s5,s5,s9
 292:	f685dce3          	bge	a1,s0,20a <.L45>
 296:	fff40c93          	add	s9,s0,-1
 29a:	da941ee3          	bne	s0,s1,56 <.L6>

0000029e <.L36>:
 29e:	1479                	add	s0,s0,-2
 2a0:	9ae6                	add	s5,s5,s9
 2a2:	8d5e                	mv	s10,s7
 2a4:	9456                	add	s0,s0,s5
 2a6:	89e2                	mv	s3,s8
 2a8:	84ea                	mv	s1,s10
 2aa:	9b22                	add	s6,s6,s0
 2ac:	f7a5d6e3          	bge	a1,s10,218 <.L31>

000002b0 <.L43>:
 2b0:	fffd0793          	add	a5,s10,-1
 2b4:	843e                	mv	s0,a5
 2b6:	d93493e3          	bne	s1,s3,3c <.L3>

000002ba <.L35>:
 2ba:	14f9                	add	s1,s1,-2
 2bc:	97da                	add	a5,a5,s6
 2be:	94be                	add	s1,s1,a5

000002c0 <.L30>:
 2c0:	40ba                	lw	ra,140(sp)
 2c2:	442a                	lw	s0,136(sp)
 2c4:	490a                	lw	s2,128(sp)
 2c6:	59f6                	lw	s3,124(sp)
 2c8:	5a66                	lw	s4,120(sp)
 2ca:	5ad6                	lw	s5,116(sp)
 2cc:	5b46                	lw	s6,112(sp)
 2ce:	5bb6                	lw	s7,108(sp)
 2d0:	5c26                	lw	s8,104(sp)
 2d2:	5c96                	lw	s9,100(sp)
 2d4:	5d06                	lw	s10,96(sp)
 2d6:	4df6                	lw	s11,92(sp)
 2d8:	8526                	mv	a0,s1
 2da:	449a                	lw	s1,132(sp)
 2dc:	6149                	add	sp,sp,144
 2de:	8082                	ret


/Users/jsprowes/artificer/InternalCadence/tmp/fib_imperative.o:     file format elf32-littleriscv


Disassembly of section .text:

00000000 <_Z14fib_imperativei>:
   0:	862a                	mv	a2,a0
   2:	4789                	li	a5,2
   4:	4505                	li	a0,1
   6:	00c7da63          	bge	a5,a2,1a <.L4>
   a:	4705                	li	a4,1

0000000c <.L3>:
   c:	86aa                	mv	a3,a0
   e:	0785                	add	a5,a5,1
  10:	953a                	add	a0,a0,a4
  12:	8736                	mv	a4,a3
  14:	fef61ce3          	bne	a2,a5,c <.L3>
  18:	8082                	ret

0000001a <.L4>:
  1a:	8082                	ret

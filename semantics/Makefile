.PHONY: kompiled test clean

kompiled: fpl-kompiled/timestamp

fpl-kompiled/timestamp: fpl.k
	kompile fpl.k --backend llvm --main-module FPL-TESTING -ccopt -g

test: kompiled
	tests/run_tests.sh

clean:
	rm -rf fpl-kompiled/

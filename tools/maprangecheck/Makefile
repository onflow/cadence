D := $(or $(patsubst %/,%,$(dir $(lastword $(MAKEFILE_LIST)))),.)
$(D)/maprangecheck.so: $(addprefix $(D)/,main.go analyzer.go analyzer_test.go go.mod)
	cd $(dir $@) && \
	go build -buildmode=plugin .
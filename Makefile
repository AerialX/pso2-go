CMDS		:=	pso2-ice pso2-afp pso2-text
GO			:=	go
UTIL_GO		:=	$(wildcard util/*.go)
ICE_GO		:=	$(wildcard ice/*.go) $(UTIL_GO)
AFP_GO		:=	$(wildcard afp/*.go) $(UTIL_GO)
TEXT_GO		:=	$(wildcard text/*.go) $(UTIL_GO)

all: $(CMDS)

$(CMDS):
	$(GO) build -o $@ ./cmd/$@

clean:
	rm -f $(CMDS)

pso2-ice: $(ICE_GO) $(wildcard cmd/pso2-ice/*.go)
pso2-text: $(TEXT_GO) $(wildcard cmd/pso2-text/*.go)
pso2-afp: $(AFP_GO) $(wildcard cmd/pso2-afp/*.go)

.PHONY: all clean

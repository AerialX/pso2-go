CMDS			:=	pso2-ice pso2-afp pso2-text pso2-trans pso2-net
GO				:=	go
UTIL_GO			:=	$(wildcard util/*.go)
NET_GO			:=	$(wildcard net/*.go)
NET_PACKETS_GO	:=	$(wildcard net/packets/*.go)
ICE_GO			:=	$(wildcard ice/*.go) $(UTIL_GO)
AFP_GO			:=	$(wildcard afp/*.go) $(UTIL_GO)
TEXT_GO			:=	$(wildcard text/*.go) $(UTIL_GO)
TRANS_GO		:=	$(wildcard trans/*.go) $(UTIL_GO)

all: $(CMDS)

$(CMDS):
	$(GO) build -o $@ ./cmd/$@

clean:
	rm -f $(CMDS)

pso2-ice: $(ICE_GO) $(wildcard cmd/pso2-ice/*.go)
pso2-text: $(TEXT_GO) $(wildcard cmd/pso2-text/*.go)
pso2-trans: $(TRANS_GO) $(TEXT_GO) $(wildcard cmd/pso2-trans/*.go)
pso2-afp: $(AFP_GO) $(wildcard cmd/pso2-afp/*.go)
pso2-net: $(NET_GO) $(NET_PACKETS_GO) $(wildcard cmd/pso2-net/*.go)

.PHONY: all clean

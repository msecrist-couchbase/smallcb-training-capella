CPPFLAGS=-Wall -g
LDLIBS=-lcouchbase -pthread

PROGS = code

all: $(PROGS)

clean:
	rm -f $(PROGS)
	rm -rf *.dSYM
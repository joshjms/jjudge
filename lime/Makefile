CC ?= gcc
BIN := build/lime

SRC := \
	main.c \
	src/run.c \
	src/utils.c \
	src/cgroup.c \
	src/api.c \
	third_party/cJSON.c

OBJ := $(SRC:.c=.o)

CPPFLAGS += -I src/include -I third_party/include
CFLAGS ?= -O2 -g -Wall -Wextra -std=c11
LDFLAGS ?=

.PHONY: all clean

all: $(BIN)

$(BIN): $(OBJ)
	$(CC) $(CFLAGS) $(OBJ) -o $@ $(LDFLAGS)

%.o: %.c
	$(CC) $(CPPFLAGS) $(CFLAGS) -c $< -o $@

clean:
	rm -f $(OBJ) $(BIN)

#pragma once

#include <stddef.h>

typedef struct {
    int stdin_fd;
    const char *stdin_buf;
    size_t stdin_len;

    int stdout_fd;
    char *stdout_buf;

    int stderr_fd;
    char *stderr_buf;

    void *_threads;
} IOContext;

int io_start(IOContext *ctx);

int io_wait(IOContext *ctx);

void io_free(IOContext *ctx);

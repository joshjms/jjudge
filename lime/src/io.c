#include "io.h"

#include <errno.h>
#include <pthread.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "utils.h"

typedef struct {
    int fd;
    const char *buf;
    size_t len;
    int error;
} StdinArgs;

typedef struct {
    int fd;
    char *buf;
    int error;
} ReaderArgs;

typedef struct {
    pthread_t stdin_tid;
    pthread_t stdout_tid;
    pthread_t stderr_tid;

    StdinArgs stdin_args;
    ReaderArgs stdout_args;
    ReaderArgs stderr_args;

    int threads_started;
} IOThreads;

static void *stdin_writer(void *arg) {
    // Block SIGPIPE so a broken pipe doesn't kill the process;
    // write() will return -1/EPIPE instead.
    sigset_t set;
    sigemptyset(&set);
    sigaddset(&set, SIGPIPE);
    pthread_sigmask(SIG_BLOCK, &set, NULL);

    StdinArgs *a = arg;
    size_t written = 0;

    while (written < a->len) {
        ssize_t n = write(a->fd, a->buf + written, a->len - written);
        if (n > 0) {
            written += (size_t)n;
        } else if (n < 0) {
            if (errno == EINTR) continue;
            a->error = errno;
            break;
        }
        // n == 0 shouldn't happen for a pipe, but treat as done
    }
 
    close(a->fd);
    return NULL;
}

static void *pipe_reader(void *arg) {
    ReaderArgs *a = arg;
 
    a->buf = read_all_from_fd(a->fd);
    if (!a->buf) {
        a->error = errno ? errno : EIO;
    }
    
    close(a->fd);
    return NULL;
}

int io_start(IOContext *ctx) {
    if(!ctx) {
        errno = EINVAL;
        return -1;
    }

    IOThreads *t = malloc(sizeof(IOThreads));
    if(!t) {
        return -1;
    }
    ctx -> _threads = t;

    t->stdin_args = (StdinArgs){
        .fd = ctx->stdin_fd,
        .buf = ctx->stdin_buf,
        .len = ctx->stdin_len,
        .error = 0,
    };

    if(pthread_create(&t->stdin_tid, NULL, stdin_writer, &t->stdin_args) != 0) {
        perror("pthread_create stdin_writer");
        free(t);
        ctx->_threads = NULL;
        return -1;
    }

    t->threads_started |= (1 << 0);

    // stdout
    t->stdout_args = (ReaderArgs){ .fd = ctx->stdout_fd };
    if (pthread_create(&t->stdout_tid, NULL, pipe_reader, &t->stdout_args) != 0) {
        perror("pthread_create stdout reader");
        /* stdin thread is running; join it before bailing */
        pthread_join(t->stdin_tid, NULL);
        free(t);
        ctx->_threads = NULL;
        return -1;
    }
    t->threads_started |= (1 << 1);
 
    // stderr
    t->stderr_args = (ReaderArgs){ .fd = ctx->stderr_fd };
    if (pthread_create(&t->stderr_tid, NULL, pipe_reader, &t->stderr_args) != 0) {
        perror("pthread_create stderr reader");
        pthread_join(t->stdin_tid, NULL);
        pthread_join(t->stdout_tid, NULL);
        free(t);
        ctx->_threads = NULL;
        return -1;
    }
    t->threads_started |= (1 << 2);

    return 0;
}

int io_wait(IOContext *ctx) {
    if (!ctx || !ctx->_threads) {
        errno = EINVAL;
        return -1;
    }
 
    IOThreads *t = ctx->_threads;
    int ret = 0;
 
    if (t->threads_started & (1 << 0)) pthread_join(t->stdin_tid,  NULL);
    if (t->threads_started & (1 << 1)) pthread_join(t->stdout_tid, NULL);
    if (t->threads_started & (1 << 2)) pthread_join(t->stderr_tid, NULL);
 
    if (t->stdin_args.error) {
        fprintf(stderr, "io: stdin writer failed: %s\n", strerror(t->stdin_args.error));
        ret = -1;
    }
    if (t->stdout_args.error) {
        fprintf(stderr, "io: stdout reader failed: %s\n", strerror(t->stdout_args.error));
        ret = -1;
    }
    if (t->stderr_args.error) {
        fprintf(stderr, "io: stderr reader failed: %s\n", strerror(t->stderr_args.error));
        ret = -1;
    }
 
    ctx->stdout_buf = t->stdout_args.buf;
    ctx->stderr_buf = t->stderr_args.buf;
 
    free(t);
    ctx->_threads = NULL;
 
    return ret;
}

void io_free(IOContext *ctx) {
    if (!ctx) return;
    free(ctx->stdout_buf);
    free(ctx->stderr_buf);
    ctx->stdout_buf = NULL;
    ctx->stderr_buf = NULL;
}

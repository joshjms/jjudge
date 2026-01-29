#define _GNU_SOURCE

#include "utils.h"

#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

#ifndef O_CLOEXEC
#define O_CLOEXEC 02000000
#endif

int create_directory_if_not_exists(const char *path) {
    struct stat st = {0};
    if (stat(path, &st) == -1) {
        if (mkdir(path, 0755) == -1) {
            int err = errno;
            fprintf(stderr, "mkdir failed for %s: %s\n", path, strerror(err));
            return -1;
        }
    }
    return 0;
}

char *join_paths(const char *base, const char *sub) {
    if(!base || !sub) return NULL;
    size_t base_len = strlen(base) + 1 + strlen(sub) + 1;
    char *result = malloc(base_len);
    if(!result) return NULL;
    snprintf(result, base_len, "%s/%s", base, sub);
    return result;
}

char *read_all_from_stdin() {
    size_t capacity = 4096;
    size_t length = 0;
    char *buffer = malloc(capacity);
    if (!buffer) {
        return NULL;
    }

    ssize_t n;
    while ((n = read(STDIN_FILENO, buffer + length, capacity - length - 1)) > 0) {
        length += n;
        if (length + 1 >= capacity) {
            capacity *= 2;
            char *new_buffer = realloc(buffer, capacity);
            if (!new_buffer) {
                free(buffer);
                return NULL;
            }
            buffer = new_buffer;
        }
    }

    if (n == -1) {
        free(buffer);
        return NULL;
    }

    buffer[length] = '\0';
    return buffer;
}

char *read_all_from_fd(int fd) {
    size_t capacity = 4096;
    size_t length = 0;
    char *buffer = malloc(capacity);
    if (!buffer) {
        return NULL;
    }

    ssize_t n;
    while ((n = read(fd, buffer + length, capacity - length - 1)) > 0) {
        length += n;
        if (length + 1 >= capacity) {
            capacity *= 2;
            char *new_buffer = realloc(buffer, capacity);
            if (!new_buffer) {
                free(buffer);
                return NULL;
            }
            buffer = new_buffer;
        }
    }

    if (n == -1) {
        free(buffer);
        return NULL;
    }

    buffer[length] = '\0';
    return buffer;
}

int remove_directory(const char *path) {
    if (rmdir(path) == -1) {
        int err = errno;
        fprintf(stderr, "rmdir failed for %s: %s\n", path, strerror(err));
        return -1;
    }
    return 0;
}

int run_wait(char *const argv[]) {
    pid_t c = fork();
    if (c < 0) return -1;
    if (c == 0) {
        execvp(argv[0], argv);
        _exit(127);
    }
    int st = 0;
    if (waitpid(c, &st, 0) < 0) return -1;
    if (!WIFEXITED(st) || WEXITSTATUS(st) != 0) {
        errno = EPERM;
        return -1;
    }
    return 0;
}

int write_to_file(const char *path, const char *value) {
    int fd = open(path, O_WRONLY | O_CLOEXEC);
    if(fd == -1) {
        int err = errno;
        fprintf(stderr, "open file failed for %s: %s\n", path, strerror(err));
        return -1;
    }

    if(write(fd, value, strlen(value)) == -1) {
        int err = errno;
        fprintf(stderr, "write to file failed for %s: %s\n", path, strerror(err));
        close(fd);
        return -1;
    }

    close(fd);
    return 0;
}

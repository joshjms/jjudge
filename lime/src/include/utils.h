#ifndef UTILS_H
#define UTILS_H

#include <stddef.h>

int create_directory_if_not_exists(const char *path);
char *join_paths(const char *base, const char *sub);
char *read_all_from_stdin();
char *read_all_from_fd(int fd);
int remove_directory(const char *path);
int run_wait(char *const argv[]);
int write_to_file(const char *path, const char *value);

#endif

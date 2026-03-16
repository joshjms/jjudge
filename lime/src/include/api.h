#ifndef API_H
#define API_H

#include <stdint.h>
#include <stdlib.h>

typedef struct {
    char *id;

    char **args;
    size_t args_c;

    char **envp;
    size_t envp_c;

    uint64_t cpu_time_limit_us;
    uint64_t wall_time_limit_us;
    uint64_t memory_limit_bytes;
    uint32_t max_processes;
    uint64_t output_limit_bytes;
    uint32_t max_open_files;
    uint64_t stack_limit_bytes;

    char *use_cpus; // e.g. "0-3,5"
    char *use_mems; // e.g. "0,2"

    char *stdin;

    char *rootfs_path;  

    /** null-terminated array of "src:dst[:ro|rw]" */
    char **bind_mounts;
    size_t bind_mounts_c;

    int use_overlayfs;
} ExecRequest;

typedef struct {
    char *id;

    int exit_code;
    int term_signal;

    uint64_t cpu_time_us;
    uint64_t wall_time_us;
    uint64_t memory_bytes; // maxrss

    char *stdout;
    char *stderr;
} ExecResponse;

void free_exec_request(ExecRequest *req);
void free_exec_response(ExecResponse *resp);

#endif

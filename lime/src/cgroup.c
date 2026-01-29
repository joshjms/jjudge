#define _GNU_SOURCE

#include "cgroup.h"

#include <fcntl.h>
#include <inttypes.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>

#include "utils.h"

#ifndef O_CLOEXEC
#define O_CLOEXEC 02000000
#endif

static const char *CGROUP_ROOT = "/sys/fs/cgroup";

static bool validate_cgroup(const struct cgroup_config *config);

int create_cgroup(const char *cgroup_root, const struct cgroup_config *config) {
    if(!cgroup_root) {
        cgroup_root = CGROUP_ROOT;
    }

    if (!validate_cgroup(config)) {
        fprintf(stderr, "Cgroup configuration is invalid\n");
        return -1;
    }

    char *cgroup_subtree_path = join_paths(cgroup_root, config->name);
    if (!cgroup_subtree_path) {
        fprintf(stderr, "Failed to build cgroup path\n");
        return -1;
    }

    if (create_directory_if_not_exists(cgroup_subtree_path) == -1) {
        fprintf(stderr, "Failed to create cgroup directory %s\n", cgroup_subtree_path);
        return -1;
    }

    if (config->cpu_weight > 0) {
        char *path = join_paths(cgroup_subtree_path, "cpu.weight");
        if (!path) {
            fprintf(stderr, "Failed to build path for cpu.weight\n");
            remove_directory(cgroup_subtree_path);
            return -1;
        }
        char value_buffer[32];
        snprintf(value_buffer, sizeof(value_buffer), "%d", config->cpu_weight);
        if(write_to_file(path, value_buffer) == -1) {
            fprintf(stderr, "Failed to set cpu weight for cgroup %s\n", config->name);
            remove_directory(cgroup_subtree_path);
            return -1;
        };
        free(path);
    }

    if(config->cpu_quota_us != 0) {
        char *path = join_paths(cgroup_subtree_path, "cpu.max");
        if (!path) {
            fprintf(stderr, "Failed to build path for cpu.max\n");
            remove_directory(cgroup_subtree_path);
            return -1;
        }
        char value_buffer[32];
        if(config->cpu_quota_us == -1) {
            snprintf(value_buffer, sizeof(value_buffer), "max 100000");
        } else {
            snprintf(value_buffer, sizeof(value_buffer), "%d 100000", config->cpu_quota_us);
        }
        if(write_to_file(path, value_buffer) == -1) {
            fprintf(stderr, "Failed to set cpu quota for cgroup %s\n", config->name);
            remove_directory(cgroup_subtree_path);
            return -1;
        };
        free(path);
    }

    if(config->memory_limit_bytes > 0) {
        char *path = join_paths(cgroup_subtree_path, "memory.max");
        if (!path) {
            fprintf(stderr, "Failed to build path for memory.max\n");
            remove_directory(cgroup_subtree_path);
            return -1;
        };
        char value_buffer[32];
        snprintf(value_buffer, sizeof(value_buffer), "%" PRIu64, config->memory_limit_bytes);
        if(write_to_file(path, value_buffer) == -1) {
            fprintf(stderr, "Failed to set memory limit for cgroup %s\n", config->name);
            remove_directory(cgroup_subtree_path);
            return -1;
        };
        free(path);
    }

    if(config->pids_limit != 0) {
        char *path = join_paths(cgroup_subtree_path, "pids.max");
        char value_buffer[32];
        if(config->pids_limit == -1) {
            snprintf(value_buffer, sizeof(value_buffer), "max");
        } else if(config->pids_limit > 0) {
            snprintf(value_buffer, sizeof(value_buffer), "%d", config->pids_limit);
        }
        if(write_to_file(path, value_buffer) == -1) {
            fprintf(stderr, "Failed to set pids limit for cgroup %s\n", config->name);
            remove_directory(cgroup_subtree_path);
            return -1;
        };
        free(path);
    }

    free(cgroup_subtree_path);

    return 0;
}

static bool validate_cgroup(const struct cgroup_config *config) {
    if (config->cpu_weight < 0 || config->cpu_weight > 10000) {
        fprintf(stderr, "Invalid cpu_weight: %d\n", config->cpu_weight);
        return false;
    }
    if (config->pids_limit < -1) {
        fprintf(stderr, "Invalid pids_limit: %d\n", config->pids_limit);
        return false;
    }
    return true;
}

int put_process_in_cgroup(const char *cgroup_root, const char *cgroup_name, pid_t pid) {
    if(!cgroup_root) {
        cgroup_root = CGROUP_ROOT;
    }

    char *cgroup_procs_path = join_paths(cgroup_root, cgroup_name);
    if (!cgroup_procs_path) {
        fprintf(stderr, "Failed to build cgroup procs path\n");
        return -1;
    }
    char *path = join_paths(cgroup_procs_path, "cgroup.procs");
    if (!path) {
        fprintf(stderr, "Failed to build path for cgroup.procs\n");
        free(cgroup_procs_path);
        return -1;
    }
    free(cgroup_procs_path);

    char pid_buffer[32];
    snprintf(pid_buffer, sizeof(pid_buffer), "%d", pid);

    if(write_to_file(path, pid_buffer) == -1) {
        fprintf(stderr, "Failed to add pid %d to cgroup %s\n", pid, cgroup_name);
        free(path);
        return -1;
    };
    free(path);

    return 0;
}

int delete_cgroup(const char *cgroup_root, const char *cgroup_name) {
    if(!cgroup_root) {
        cgroup_root = CGROUP_ROOT;
    }

    char *cgroup_path = join_paths(cgroup_root, cgroup_name);
    if (!cgroup_path) {
        fprintf(stderr, "Failed to build cgroup path\n");
        return -1;
    }

    if (remove_directory(cgroup_path) == -1) {
        fprintf(stderr, "Failed to remove cgroup directory %s\n", cgroup_path);
        free(cgroup_path);
        return -1;
    }

    free(cgroup_path);
    return 0;
}

int get_cgroup_stats(const char *cgroup_root, const char *cgroup_name, struct cgroup_stats *stats) {
    if(!cgroup_root) {
        cgroup_root = CGROUP_ROOT;
    }

    char *cgroup_path = join_paths(cgroup_root, cgroup_name);
    if (!cgroup_path) {
        fprintf(stderr, "Failed to build cgroup path\n");
        return -1;
    }

    // read cpu usage
    char *cpu_usage_path = join_paths(cgroup_path, "cpu.stat");
    if (!cpu_usage_path) {
        fprintf(stderr, "Failed to build cpu.stat path\n");
        free(cgroup_path);
        return -1;
    }
    FILE *cpu_file = fopen(cpu_usage_path, "r");
    if (!cpu_file) {
        fprintf(stderr, "Failed to open cpu.stat file\n");
        free(cpu_usage_path);
        free(cgroup_path);
        return -1;
    }
    char line[256];
    while (fgets(line, sizeof(line), cpu_file)) {
        if (sscanf(line, "usage_usec %" SCNu64, &stats->cpu_usage_us) == 1) {
            break;
        }
    }
    fclose(cpu_file);
    free(cpu_usage_path);

    // read memory
    char *memory_usage_path = join_paths(cgroup_path, "memory.peak");
    if (!memory_usage_path) {
        fprintf(stderr, "Failed to build memory.peak path\n");
        free(cgroup_path);
        return -1;
    }
    FILE *mem_file = fopen(memory_usage_path, "r");
    if (!mem_file) {
        fprintf(stderr, "Failed to open memory.peak file\n");
        free(memory_usage_path);
        free(cgroup_path);
        return -1;
    }
    fscanf(mem_file, "%" SCNu64, &stats->memory_usage_bytes);
    fclose(mem_file);
    free(memory_usage_path);
    
    free(cgroup_path);

    return 0;
}

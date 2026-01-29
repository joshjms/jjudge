#ifndef CGROUP_H
#define CGROUP_H

#include <stdint.h>
#include <sys/types.h>

struct cgroup_config {
    char *name;
    int cpu_weight;
    int cpu_quota_us;
    uint64_t memory_limit_bytes;
    int pids_limit;
};

struct cgroup_stats {
    uint64_t cpu_usage_us;
    uint64_t memory_usage_bytes;
};

int create_cgroup(const char *cgroup_root, const struct cgroup_config *config);
int put_process_in_cgroup(const char *cgroup_root, const char *cgroup_name, pid_t pid);
int delete_cgroup(const char *cgroup_root, const char *cgroup_name);
int get_cgroup_stats(const char *cgroup_root, const char *cgroup_name, struct cgroup_stats *stats);

#endif

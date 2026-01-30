#define _GNU_SOURCE

#include "run.h"

#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <getopt.h>
#include <inttypes.h>
#include <limits.h>
#include <linux/capability.h>
#include <poll.h>
#include <pwd.h>
#include <sched.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/prctl.h>
#include <sys/resource.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <sys/wait.h>
#include <time.h>
#include <unistd.h>

#include "api.h"
#include "cgroup.h"
#include "utils.h"

#include "cJSON.h"

static const int CHILD_STACK_SIZE = 1024 * 1024;

static ExecRequest* read_exec_request_from_stdin();
static ExecRequest* parse_exec_request_from_json(cJSON *json);
static int setup_uid_gid_maps(pid_t pid);
static int waitpid_with_timeout(pid_t pid, int *exit_code, int *signal, uint64_t *wall_time, uint64_t timeout_us);
static int create_response_json(const ExecResponse *resp, char **out_json);

// child bootstrap helpers
static int child_fn(void *arg);
static int setup_rootfs_without_overlayfs(const ExecRequest *cfg, const char *ctr_dir);
static int setup_rootfs_with_overlayfs(const ExecRequest *cfg, const char *ctr_dir);
static int drop_all_caps(void);
static void set_limit(int resource, rlim_t lim);

// socket helpers
static int write_byte(int fd, char b);
static int read_byte(int fd, char *out);

static void print_usage(const char *prog) {
    fprintf(stderr, "Runs a containerized process.\n");
    fprintf(stderr, "\n");
    fprintf(stderr, "Usage:\n");
    fprintf(stderr, "  %s run\n\n", prog);
    fprintf(stderr, "You can specify the container configuration via a JSON file passed to stdin. See src/include/api.h for the definition of ExecRequest.\n");
    fprintf(stderr, "\n");
}

struct child_args {
    int sync_fd;
    int in_fd;
    int out_fd;
    int err_fd;
    ExecRequest *cfg;
};

int handle_run(int argc, char **argv) {
    static struct option long_opts[] = {
        {"help", no_argument, NULL, 0},
        {0, 0, 0, 0},
    };

    opterr = 0;
    optind = 2;
    int opt;
    while((opt = getopt_long(argc, argv, "", long_opts, NULL)) != -1) {
        switch(opt) {
            case 0:
                print_usage(argv[0]);
                return 0;
            case '?':
            default:
                fprintf(stderr, "Unknown option\n");
                print_usage(argv[0]);
                return 1;
        }
    }

    ExecRequest *req = read_exec_request_from_stdin();
    if (!req) {
        return 1;
    }

    void *stack = malloc(CHILD_STACK_SIZE);
    if (!stack) {
        fprintf(stderr, "Failed to allocate stack for child\n");
        free_exec_request(req);
        return 1;
    }

    // make socket pairs for sync
    int sv[2];
    if(socketpair(AF_UNIX, SOCK_STREAM, 0, sv) == -1) {
        perror("socketpair");
        return 1;
    }

    // pipes for stdin/stdout/stderr
    int in_pipe[2];
    if(pipe2(in_pipe, O_CLOEXEC) != 0) {
        perror("pipe2 in_pipe");
        return 1;
    }

    int out_pipe[2];
    if(pipe2(out_pipe, O_CLOEXEC) != 0) {
        perror("pipe2 out_pipe");
        return 1;
    }

    int err_pipe[2];
    if(pipe2(err_pipe, O_CLOEXEC) != 0) {
        perror("pipe2 err_pipe");
        return 1;
    }

    struct child_args ch_args = {
        .sync_fd = sv[1],
        .in_fd = in_pipe[0],
        .out_fd = out_pipe[1],
        .err_fd = err_pipe[1],
        .cfg = req,
    };

    if(write(in_pipe[1], req->stdin, strlen(req->stdin)) == -1) {
        perror("write stdin to in_pipe");
        return 1;
    }
    close(in_pipe[1]);

    int flags = SIGCHLD | CLONE_NEWUSER | CLONE_NEWPID | CLONE_NEWNS | CLONE_NEWUTS | CLONE_NEWIPC | CLONE_NEWNET | CLONE_NEWCGROUP | CLONE_NEWTIME;

    pid_t child_pid = clone(child_fn, stack + CHILD_STACK_SIZE, flags, &ch_args);
    if (child_pid == -1) {
        fprintf(stderr, "Failed to clone: %m\n");
        close(sv[0]);
        close(sv[1]);
        free(stack);
        return 1;
    }

    // setup uid/gid maps
    if(setup_uid_gid_maps(child_pid) != 0) {
        fprintf(stderr, "Failed to setup uid/gid maps\n");
        kill(child_pid, SIGKILL);
        close(sv[0]);
        close(sv[1]);
        free(stack);
        return 1;
    }

    // notify child that uid/gid maps are setup

    if(write_byte(sv[0], 'A') != 0) {
        perror("write_byte A");
        kill(child_pid, SIGKILL);
        close(sv[0]);
        close(sv[1]);
        free(stack);
        return 1;
    }

    // wait for child to finish bootstrapping 
    char b = 0;
    if (read_byte(sv[0], &b) != 0 || b != 'B') {
        if(b == 'X') {
            fprintf(stderr, "Child reported an error during bootstrap\n");
        } else {
            perror("read_byte B");
        }
        kill(child_pid, SIGKILL);
        close(sv[0]);
        close(sv[1]);
        free(stack);
        return 1;
    }

    struct cgroup_config cg_cfg = {
        .name = req->id,
        .cpu_weight = 1000,
        .cpu_quota_us = 100000,
        .memory_limit_bytes = req->memory_limit_bytes,
        .pids_limit = req->max_processes,
        .use_cpus = req->use_cpus,
        .use_mems = req->use_mems,
    };

    const char *cgroup_root = getenv("LIME_CGROUP_ROOT");

    if(create_cgroup(cgroup_root, &cg_cfg) != 0) {
        fprintf(stderr, "Failed to create cgroup\n");
        kill(child_pid, SIGKILL);
        close(sv[0]);
        close(sv[1]);
        free(stack);
        return 1;
    }

    if(put_process_in_cgroup(cgroup_root, req->id, child_pid) != 0) {
        fprintf(stderr, "Failed to put process in cgroup\n");
        delete_cgroup(cgroup_root, req->id);
        kill(child_pid, SIGKILL);
        close(sv[0]);
        close(sv[1]);
        free(stack);
        return 1;
    }

    // notify child to drop capabilities
    if(write_byte(sv[0], 'C') != 0) {
        perror("write_byte C");
        delete_cgroup(cgroup_root, req->id);
        kill(child_pid, SIGKILL);
        close(sv[0]);
        close(sv[1]);
        free(stack);
        return 1;
    }

    int exit_code = 0, signal = 0;
    uint64_t wall_time = 0;
    if(waitpid_with_timeout(child_pid, &exit_code, &signal, &wall_time, req->wall_time_limit_us) != 0) {
        if(errno != ETIMEDOUT) {
            fprintf(stderr, "Child process timed out or error occurred\n");
            kill(child_pid, SIGKILL);
            delete_cgroup(cgroup_root, req->id);
            close(sv[0]);
            close(sv[1]);
            free(stack);
            free_exec_request(req);
            return 1;
        }
    }
    fprintf(stderr, "Child process exited with exit code %d and signal %d\n", exit_code, signal);
    
    close(sv[0]);
    close(sv[1]);
    // delete_cgroup(cgroup_root, req->id);
    free(stack);

    struct cgroup_stats stats;
    if(get_cgroup_stats(cgroup_root, req->id, &stats) != 0) {
        fprintf(stderr, "Failed to get cgroup stats\n");
        delete_cgroup(cgroup_root, req->id);
        free_exec_request(req);
        return 1;
    }

    // read stdout/stderr from pipes
    close(out_pipe[1]);
    close(err_pipe[1]);
    char *stdout_output = read_all_from_fd(out_pipe[0]);
    char *stderr_output = read_all_from_fd(err_pipe[0]);

    ExecResponse *resp = malloc(sizeof(ExecResponse));
    if (!resp) {
        fprintf(stderr, "Failed to allocate ExecResponse\n");
        delete_cgroup(cgroup_root, req->id);
        free_exec_request(req);
        return 1;
    }

    resp->id = strdup(req->id);
    resp->exit_code = exit_code;
    resp->term_signal = signal;
    resp->wall_time_us = wall_time;
    resp->cpu_time_us = stats.cpu_usage_us;
    resp->memory_bytes = stats.memory_usage_bytes;
    resp->stdout = stdout_output;
    resp->stderr = stderr_output;

    char *out_json; 
    if(create_response_json(resp, &out_json) != 0) {
        fprintf(stderr, "Failed to create response JSON\n");
        free_exec_response(resp);
        delete_cgroup(cgroup_root, req->id);
        free_exec_request(req);
        return 1;
    }

    printf("%s\n", out_json);

    free(out_json);
    delete_cgroup(cgroup_root, req->id);
    free_exec_request(req);
    free_exec_response(resp);

    return 0;
}

static ExecRequest* read_exec_request_from_stdin() {
    char *input = read_all_from_stdin();
    if (!input) {
        fprintf(stderr, "Failed to read ExecRequest from stdin\n");
        free(input);
        return NULL;
    }

    cJSON *json = cJSON_Parse(input);
    free(input);

    if(!json) {
        fprintf(stderr, "Failed to parse ExecRequest JSON from stdin\n");
        return NULL;
    }

    if(!cJSON_IsObject(json)) {
        fprintf(stderr, "ExecRequest JSON is not an object\n");
        cJSON_Delete(json);
        return NULL;
    }

    ExecRequest *req = parse_exec_request_from_json(json);
    cJSON_Delete(json);

    return req;
}

static ExecRequest* parse_exec_request_from_json(cJSON *json) {
    if(!cJSON_IsObject(json)) {
        fprintf(stderr, "ExecRequest JSON is not an object\n");
        return NULL;
    }

    ExecRequest *req = calloc(1, sizeof(ExecRequest));
    if (!req) {
        fprintf(stderr, "Failed to allocate ExecRequest\n");
        return NULL;
    }

    cJSON *id = cJSON_GetObjectItemCaseSensitive(json, "id");
    if(!cJSON_IsString(id)) {
        fprintf(stderr, "ExecRequest.id is not a string\n");
        free_exec_request(req);
        return NULL;
    }
    req->id = strdup(id->valuestring);
    if (!req->id) {
        fprintf(stderr, "Failed to allocate ExecRequest.id\n");
        free_exec_request(req);
        return NULL;
    }

    cJSON *args = cJSON_GetObjectItemCaseSensitive(json, "args");
    if (!cJSON_IsArray(args)) {
        fprintf(stderr, "ExecRequest.args is not an array\n");
        free_exec_request(req);
        return NULL;
    }

    int args_c = cJSON_GetArraySize(args);
    req->args = calloc(args_c + 1,  sizeof(char*));
    if (!req->args) {
        fprintf(stderr, "Failed to allocate ExecRequest.args\n");
        free_exec_request(req);
        return NULL;
    }

    for (int i = 0; i < args_c; i++) {
        cJSON *arg = cJSON_GetArrayItem(args, i);
        if (!cJSON_IsString(arg)) {
            fprintf(stderr, "ExecRequest.args[%d] is not a string\n", i);
            free_exec_request(req);
            return NULL;
        }
        req->args[i] = strdup(arg->valuestring);
        if (!req->args[i]) {
            fprintf(stderr, "Failed to allocate ExecRequest.args[%d]\n", i);
            free_exec_request(req);
            return NULL;
        }
    }
    req->args[args_c] = NULL;
    req->args_c = args_c;

    cJSON *envp = cJSON_GetObjectItemCaseSensitive(json, "envp");
    if (!cJSON_IsArray(envp)) {
        fprintf(stderr, "ExecRequest.envp is not an array\n");
        free_exec_request(req);
        return NULL;
    }

    int envp_c = cJSON_GetArraySize(envp);
    req->envp = calloc(envp_c + 1,  sizeof(char*));
    if (!req->envp) {
        fprintf(stderr, "Failed to allocate ExecRequest.envp\n");
        free_exec_request(req);
        return NULL;
    }

    for(int i = 0; i < envp_c; i++) {
        cJSON *env = cJSON_GetArrayItem(envp, i);
        if (!cJSON_IsString(env)) {
            fprintf(stderr, "ExecRequest.envp[%d] is not a string\n", i);
            free_exec_request(req);
            return NULL;
        }
        req->envp[i] = strdup(env->valuestring);
        if (!req->envp[i]) {
            fprintf(stderr, "Failed to allocate ExecRequest.envp[%d]\n", i);
            free_exec_request(req);
            return NULL;
        }
    }
    req->envp[envp_c] = NULL;
    req->envp_c = envp_c;

    cJSON *cpu_time_limit_us = cJSON_GetObjectItemCaseSensitive(json, "cpu_time_limit_us");
    if(!cJSON_IsNumber(cpu_time_limit_us)) {
        fprintf(stderr, "ExecRequest.cpu_time_limit_us is not a number\n");
        free_exec_request(req);
        return NULL;
    }
    if(cpu_time_limit_us->valuedouble < 0) {
        fprintf(stderr, "ExecRequest.cpu_time_limit_us is negative\n");
        free_exec_request(req);
        return NULL;
    }
    req->cpu_time_limit_us = (uint64_t)cpu_time_limit_us->valuedouble;

    cJSON *wall_time_limit_us = cJSON_GetObjectItemCaseSensitive(json, "wall_time_limit_us");
    if(!cJSON_IsNumber(wall_time_limit_us)) {
        fprintf(stderr, "ExecRequest.wall_time_limit_us is not a number\n");
        free_exec_request(req);
        return NULL;
    }
    if(wall_time_limit_us->valuedouble < 0) {
        fprintf(stderr, "ExecRequest.wall_time_limit_us is negative\n");
        free_exec_request(req);
        return NULL;
    }
    req->wall_time_limit_us = (uint64_t)wall_time_limit_us->valuedouble;

    cJSON *memory_limit_bytes = cJSON_GetObjectItemCaseSensitive(json, "memory_limit_bytes");
    if(!cJSON_IsNumber(memory_limit_bytes)) {
        fprintf(stderr, "ExecRequest.memory_limit_bytes is not a number\n");
        free_exec_request(req);
        return NULL;
    }
    if(memory_limit_bytes->valuedouble < 0) {
        fprintf(stderr, "ExecRequest.memory_limit_bytes is negative\n");
        free_exec_request(req);
        return NULL;
    }
    req->memory_limit_bytes = (uint64_t)memory_limit_bytes->valuedouble;

    cJSON *max_processes = cJSON_GetObjectItemCaseSensitive(json, "max_processes");
    if(!cJSON_IsNumber(max_processes)) {
        fprintf(stderr, "ExecRequest.max_processes is not a number\n");
        free_exec_request(req);
        return NULL;
    }
    if(max_processes->valuedouble < 0) {
        fprintf(stderr, "ExecRequest.max_processes is negative\n");
        free_exec_request(req);
        return NULL;
    }
    req->max_processes = (uint32_t)max_processes->valuedouble;

    cJSON *output_limit_bytes = cJSON_GetObjectItemCaseSensitive(json, "output_limit_bytes");
    if(!cJSON_IsNumber(output_limit_bytes)) {
        fprintf(stderr, "ExecRequest.output_limit_bytes is not a number\n");
        free_exec_request(req);
        return NULL;
    }
    if(output_limit_bytes->valuedouble < 0) {
        fprintf(stderr, "ExecRequest.output_limit_bytes is negative\n");
        free_exec_request(req);
        return NULL;
    }
    req->output_limit_bytes = (uint64_t)output_limit_bytes->valuedouble;

    cJSON *max_open_files = cJSON_GetObjectItemCaseSensitive(json, "max_open_files");
    if(!cJSON_IsNumber(max_open_files)) {
        fprintf(stderr, "ExecRequest.max_open_files is not a number\n");
        free_exec_request(req);
        return NULL;
    }
    if(max_open_files->valuedouble < 0) {
        fprintf(stderr, "ExecRequest.max_open_files is negative\n");
        free_exec_request(req);
        return NULL;
    }
    req->max_open_files = (uint32_t)max_open_files->valuedouble;

    cJSON *stack_limit_bytes = cJSON_GetObjectItemCaseSensitive(json, "stack_limit_bytes");
    if(!cJSON_IsNumber(stack_limit_bytes)) {
        fprintf(stderr, "ExecRequest.stack_limit_bytes is not a number\n");
        free_exec_request(req);
        return NULL;
    }
    if(stack_limit_bytes->valuedouble < 0) {
        fprintf(stderr, "ExecRequest.stack_limit_bytes is negative\n");
        free_exec_request(req);
        return NULL;
    }
    req->stack_limit_bytes = (uint64_t)stack_limit_bytes->valuedouble;

    cJSON *use_cpus = cJSON_GetObjectItemCaseSensitive(json, "use_cpus");
    if(!cJSON_IsString(use_cpus)) {
        fprintf(stderr, "ExecRequest.use_cpus is not a string\n");
        free_exec_request(req);
        return NULL;
    }
    req->use_cpus = strdup(use_cpus->valuestring);
    if (!req->use_cpus) {
        fprintf(stderr, "Failed to allocate ExecRequest.use_cpus\n");
        free_exec_request(req);
        return NULL;
    }

    cJSON *use_mems = cJSON_GetObjectItemCaseSensitive(json, "use_mems");
    if(!cJSON_IsString(use_mems)) {
        fprintf(stderr, "ExecRequest.use_mems is not a string\n");
        free_exec_request(req);
        return NULL;
    }
    req->use_mems = strdup(use_mems->valuestring);
    if (!req->use_mems) {
        fprintf(stderr, "Failed to allocate ExecRequest.use_mems\n");
        free_exec_request(req);
        return NULL;
    }

    cJSON *stdin = cJSON_GetObjectItemCaseSensitive(json, "stdin");
    if(!cJSON_IsString(stdin)) {
        fprintf(stderr, "ExecRequest.stdin is not a string\n");
        free_exec_request(req);
        return NULL;
    }
    req->stdin = strdup(stdin->valuestring);
    if (!req->stdin) {
        fprintf(stderr, "Failed to allocate ExecRequest.stdin\n");
        free_exec_request(req);
        return NULL;
    }

    cJSON *rootfs_path = cJSON_GetObjectItemCaseSensitive(json, "rootfs_path");
    if(!cJSON_IsString(rootfs_path)) {
        fprintf(stderr, "ExecRequest.rootfs_path is not a string\n");
        free_exec_request(req);
        return NULL;
    }
    req->rootfs_path = strdup(rootfs_path->valuestring);
    if (!req->rootfs_path) {
        fprintf(stderr, "Failed to allocate ExecRequest.rootfs_path\n");
        free_exec_request(req);
        return NULL;
    }

    cJSON *bind_mounts = cJSON_GetObjectItemCaseSensitive(json, "bind_mounts");
    if (!cJSON_IsArray(bind_mounts)) {
        fprintf(stderr, "ExecRequest.bind_mounts is not an array\n");
        free_exec_request(req);
        return NULL;
    }

    int bind_mounts_c = cJSON_GetArraySize(bind_mounts);
    req->bind_mounts = calloc(bind_mounts_c + 1,  sizeof(char*));
    if (!req->bind_mounts) {
        fprintf(stderr, "Failed to allocate ExecRequest.bind_mounts\n");
        free_exec_request(req);
        return NULL;
    }

    for(int i = 0; i < bind_mounts_c; i++) {
        cJSON *bind = cJSON_GetArrayItem(bind_mounts, i);
        if (!cJSON_IsString(bind)) {
            fprintf(stderr, "ExecRequest.bind_mounts[%d] is not a string\n", i);
            free_exec_request(req);
            return NULL;
        }
        req->bind_mounts[i] = strdup(bind->valuestring);
        if (!req->bind_mounts[i]) {
            fprintf(stderr, "Failed to allocate ExecRequest.bind_mounts[%d]\n", i);
            free_exec_request(req);
            return NULL;
        }
    }
    req->bind_mounts[bind_mounts_c] = NULL;
    req->bind_mounts_c = bind_mounts_c;

    cJSON *use_overlayfs = cJSON_GetObjectItemCaseSensitive(json, "use_overlayfs");
    if(!cJSON_IsBool(use_overlayfs)) {
        fprintf(stderr, "ExecRequest.use_overlayfs is not a boolean\n");
        free_exec_request(req);
        return NULL;
    }
    req->use_overlayfs = cJSON_IsTrue(use_overlayfs) ? 1 : 0;

    return req;
}

static int setup_uid_gid_maps(pid_t pid) {
    char pid_s[32];
    snprintf(pid_s, sizeof(pid_s), "%d", pid);
    
    char *path = join_paths(join_paths("/proc", pid_s), "setgroups");
    if(!path) {
        fprintf(stderr, "Failed to allocate path for setgroups\n");
        return -1;
    }
    write_to_file(path, "deny");
    free(path);

    char ruid_s[32];
    snprintf(ruid_s, sizeof(ruid_s), "%d", getuid());
    char rgid_s[32];
    snprintf(rgid_s, sizeof(rgid_s), "%d", getgid());

    /* Set 0 -> ruid */

    char *set_uid_map_argv[] = {
        "newuidmap",
        pid_s,
        "0",
        ruid_s,
        "1",
        NULL
    };
    if(run_wait(set_uid_map_argv) != 0) {
        fprintf(stderr, "Failed to run newuidmap\n");
        return -1;
    }

    char *set_gid_map_argv[] = {
        "newgidmap",
        pid_s,
        "0",
        rgid_s,
        "1",
        NULL
    };
    if(run_wait(set_gid_map_argv) != 0) {
        fprintf(stderr, "Failed to run newgidmap\n");
        return -1;
    }

    return 0;
}

static inline uint64_t now_us(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint64_t)ts.tv_sec * 1000000ULL + (uint64_t)ts.tv_nsec / 1000ULL;
}

static int waitpid_with_timeout(pid_t pid, int *exit_code, int *signal, uint64_t *wall_time, uint64_t timeout_us) {
    if (!exit_code || !signal || !wall_time) {
        errno = EINVAL;
        return -1;
    }

    uint64_t start_time = now_us();

    int pfd = (int)syscall(SYS_pidfd_open, pid, 0);
    if (pfd == -1) {
        if (errno == ESRCH) {
            int st;
            pid_t r = waitpid(pid, &st, WNOHANG);
            uint64_t end_time = now_us();
            if (r == pid) {
                if(WIFEXITED(st)) {
                    *exit_code = WEXITSTATUS(st);
                    *signal = 0;
                    *wall_time = end_time - start_time;
                } else if (WIFSIGNALED(st)) {
                    *exit_code = 0;
                    *signal = WTERMSIG(st);
                    *wall_time = end_time - start_time;
                } else {
                    errno = ECHILD;
                    return -1;
                }
                return 0;
            }
        }
        perror("pidfd_open");
        return -1;
    }

    struct pollfd fds = {
        .fd = pfd,
        .events = POLLIN,
        .revents = 0
    };

    int timeout_ms;
    if (timeout_us >= (uint64_t)INT32_MAX * 1000ULL) timeout_ms = INT32_MAX;
    else timeout_ms = (int)((timeout_us + 999) / 1000); // use ceil

    int ready;
    do {
        if(now_us() - start_time >= timeout_us) {
            ready = 0;
            break;
        }
        ready = poll(&fds, 1, timeout_ms);
    } while (ready == -1 && errno == EINTR);

    if (ready == -1) {
        perror("poll");
        close(pfd);
        return -1;
    }

    if (ready == 0) {
        kill(pid, SIGKILL);
        int st;
        // wait to finish reap
        while (waitpid(pid, &st, 0) == -1 && errno == EINTR) {}
        uint64_t end_time = now_us();
        close(pfd);
        errno = ETIMEDOUT;
        if(WIFEXITED(st)) {
            *exit_code = WEXITSTATUS(st);
            *signal = 0;
            *wall_time = end_time - start_time;
        } else if (WIFSIGNALED(st)) {
            *exit_code = 0;
            *signal = WTERMSIG(st);
            *wall_time = end_time - start_time;
        } else {
            errno = ECHILD;
        }
        return -1;
    }

    if (!(fds.revents & (POLLIN | POLLHUP))) {
        // Something odd (POLLNVAL, etc.)
        close(pfd);
        errno = EIO;
        return -1;
    }

    int st;
    pid_t r;
    do {
        r = waitpid(pid, &st, 0);
    } while (r == -1 && errno == EINTR);
    uint64_t end_time = now_us();

    close(pfd);

    if (r == -1) return -1;

    if(WIFEXITED(st)) {
        *exit_code = WEXITSTATUS(st);
        *signal = 0;
        *wall_time = end_time - start_time;
    } else if (WIFSIGNALED(st)) {
        *exit_code = 0;
        *signal = WTERMSIG(st);
        *wall_time = end_time - start_time;
    } else {
        errno = ECHILD;
        return -1;
    }
    return 0;
}

static int create_response_json(const ExecResponse *resp, char **out_json) {
    if (!resp || !out_json) {
        return -1;
    }

    cJSON *json = cJSON_CreateObject();
    if (!json) {
        return -1;
    }

    cJSON_AddStringToObject(json, "id", resp->id);
    cJSON_AddNumberToObject(json, "exit_code", resp->exit_code);
    cJSON_AddNumberToObject(json, "term_signal", resp->term_signal);
    cJSON_AddNumberToObject(json, "cpu_time_us", resp->cpu_time_us);
    cJSON_AddNumberToObject(json, "wall_time_us", resp->wall_time_us);
    cJSON_AddNumberToObject(json, "memory_bytes", resp->memory_bytes);
    cJSON_AddStringToObject(json, "stdout", resp->stdout);
    cJSON_AddStringToObject(json, "stderr", resp->stderr);

    char *json_str = cJSON_PrintUnformatted(json);
    cJSON_Delete(json);

    if (!json_str) {
        return -1;
    }

    *out_json = json_str;
    return 0;
}

// Child fn and helpers

static int child_fn(void *arg) {
    struct child_args *args = (struct child_args *)arg;

    int sync_fd = args->sync_fd;
    ExecRequest *cfg = args->cfg;

    // set mount propagation to private
    if(mount(NULL, "/", NULL, MS_REC | MS_PRIVATE, NULL) != 0) {
        perror("mount private");
        write_byte(sync_fd, 'X');
        _exit(1);
    }
    
    // wait for parent to setup uid/gid maps

    char b = 0;
    if (read_byte(sync_fd, &b) != 0 || b != 'A') {
        perror("read_byte A");
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    if (setresuid(0, 0, 0) != 0) { 
        perror("setresuid"); 
        write_byte(sync_fd, 'X');
        _exit(1);
    }
    if (setresgid(0, 0, 0) != 0) { 
        perror("setresgid");
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    char *dir = join_paths("/tmp/lime", cfg->id);
    if(!dir) {
        perror("join_paths");
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    if(create_directory_if_not_exists("/tmp/lime") != 0) {
        perror("create_directory_if_not_exists /tmp/lime");
        free(dir);
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    if(create_directory_if_not_exists(dir) != 0) {
        perror("create_directory_if_not_exists ctr dir");
        free(dir);
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    // setup rootfs

    if(cfg->use_overlayfs) {
        if(setup_rootfs_with_overlayfs(cfg, dir) != 0) {
            fprintf(stderr, "Failed to setup rootfs with overlayfs\n");
            free(dir);
            write_byte(sync_fd, 'X');
            _exit(1);
        }
    } else {
        if(setup_rootfs_without_overlayfs(cfg, dir) != 0) {
            fprintf(stderr, "Failed to setup rootfs\n");
            free(dir);
            write_byte(sync_fd, 'X');
            _exit(1);
        }
    }

    struct stat sb;
    char *root_path = join_paths(dir, "root");
    if (!root_path) {
        fprintf(stderr, "Failed to allocate root_path\n");
        free(dir);
        write_byte(sync_fd, 'X');
        _exit(1);
    }
    if (stat(root_path, &sb) != 0 || !S_ISDIR(sb.st_mode)) {
        fprintf(stderr, "Rootfs directory does not exist or is not a directory\n");
        free(root_path);
        free(dir);
        write_byte(sync_fd, 'X');
        _exit(1);
    }
    free(root_path);
    // notify parent that bootstrapping is done

    if(write_byte(sync_fd, 'B') != 0) {
        perror("write_byte B");
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    b = 0;
    if (read_byte(sync_fd, &b) != 0 || b != 'C') {
        perror("read_byte C");
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    // setrlimits
    set_limit(RLIMIT_NOFILE, cfg->max_open_files);
    set_limit(RLIMIT_STACK, cfg->stack_limit_bytes);
    set_limit(RLIMIT_AS, cfg->memory_limit_bytes);
    set_limit(RLIMIT_FSIZE, cfg->output_limit_bytes);
    set_limit(RLIMIT_CPU, (cfg->cpu_time_limit_us + 999999) / 1000000);

    // change root
    char *new_root = join_paths(dir, "root");
    if(!new_root) {
        fprintf(stderr, "Failed to allocate new_root\n");
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    char *put_old = join_paths(new_root, "old_root");
    if(!put_old) {
        fprintf(stderr, "Failed to allocate put_old\n");
        free(new_root);
        write_byte(sync_fd, 'X');
        _exit(1);
    }
    if (mkdir(put_old, 0755) != 0) {
        fprintf(stderr, "Failed to create put_old dir %s: %m\n", put_old);
        free(new_root);
        free(put_old);
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    if(syscall(SYS_pivot_root, new_root, put_old) != 0) {
        fprintf(stderr, "Failed to pivot_root to %s: %m\n", new_root);
        free(new_root);
        free(put_old);
        write_byte(sync_fd, 'X');
        _exit(1);
    }
    if(chdir("/") != 0) {
        fprintf(stderr, "Failed to chdir to new root: %m\n");
        free(new_root);
        free(put_old);
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    if(umount2("/old_root", MNT_DETACH) != 0) {
        fprintf(stderr, "Failed to umount old_root: %m\n");
        free(new_root);
        free(put_old);
        write_byte(sync_fd, 'X'); 
        _exit(1);
    }
    if(rmdir("/old_root") != 0) {
        fprintf(stderr, "Failed to rmdir old_root: %m\n");
        free(new_root);
        free(put_old);
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    free(new_root);
    free(put_old);

    if(args->in_fd) {
        if(dup2(args->in_fd, STDIN_FILENO) < 0) {
            perror("dup2 stdin");
            write_byte(sync_fd, 'X');
            _exit(1);
        }
        close(args->in_fd);
    }

    if(args->out_fd) {
        if(dup2(args->out_fd, STDOUT_FILENO) < 0) {
            perror("dup2 stdout");
            write_byte(sync_fd, 'X');
            _exit(1);
        }
        close(args->out_fd);
    }

    if(args->err_fd) {
        if(dup2(args->err_fd, STDERR_FILENO) < 0) {
            perror("dup2 stderr");
            write_byte(sync_fd, 'X');
            _exit(1);
        }
        close(args->err_fd);
    }

    if (!cfg->use_overlayfs) {
        if (mount(NULL, "/", NULL, MS_REMOUNT | MS_RDONLY, NULL) != 0) {
            fprintf(stderr, "Failed to remount / read-only: %m\n");
            write_byte(sync_fd, 'X');
            _exit(1);
        }
    }

    if(drop_all_caps() != 0) {
        fprintf(stderr, "Failed to drop capabilities\n");
        write_byte(sync_fd, 'X');
        _exit(1);
    }

    close(sync_fd);

    // exec the requested process
    if (execve(cfg->args[0], cfg->args, cfg->envp) != 0) {
        fprintf(stderr, "Failed to execve %s: %m\n", cfg->args[0]);
        _exit(1);
    }

    return 0;
}

static int setup_rootfs_without_overlayfs(const ExecRequest *cfg, const char *ctr_dir) {
    // just do a bind mount to ctr_dir/root
    char *root_path = join_paths(ctr_dir, "root");
    if(!root_path) {
        fprintf(stderr, "Failed to allocate root_path\n");
        return -1;
    }
    
    if(create_directory_if_not_exists(root_path) != 0) {
        fprintf(stderr, "Failed to create rootfs directory %s\n", root_path);
        free(root_path);
        return -1;
    }

    if(mount(cfg->rootfs_path, root_path, NULL, MS_BIND | MS_REC, NULL) != 0) {
        fprintf(stderr, "Failed to bind mount rootfs from %s to %s: %m\n", cfg->rootfs_path, root_path);
        free(root_path);
        return -1;
    }

    for(size_t i = 0; i < cfg->bind_mounts_c; i++) {
        char *bind = cfg->bind_mounts[i];
        char *src = strtok(bind, ":");
        char *dst = strtok(NULL, ":");
        char *options = strtok(NULL, ":");

        if(!src || !dst) {
            fprintf(stderr, "Invalid bind mount specification: %s\n", bind);
            free(root_path);
            return -1;
        }

        char *full_dst = join_paths(root_path, dst);
        if(!full_dst) {
            fprintf(stderr, "Failed to allocate full_dst for bind mount\n");
            free(root_path);
            return -1;
        }

        if(create_directory_if_not_exists(full_dst) != 0) {
            fprintf(stderr, "Failed to create bind mount destination %s\n", full_dst);
            free(full_dst);
            free(root_path);
            return -1;
        }

        unsigned long mount_flags = MS_BIND | MS_REC;
        if(options && strcmp(options, "ro") == 0) {
            mount_flags |= MS_RDONLY;
        }

        if(mount(src, full_dst, NULL, mount_flags, NULL) != 0) {
            fprintf(stderr, "Failed to bind mount from %s to %s: %m\n", src, full_dst);
            free(full_dst);
            free(root_path);
            return -1;
        }

        free(full_dst);
    }

    // check if root directory is mounted

    free(root_path);

    return 0;
}

static int setup_rootfs_with_overlayfs(const ExecRequest *cfg, const char *ctr_dir) {
    char *root_path = join_paths(ctr_dir, "root");;
    if(!root_path) {
        fprintf(stderr, "Failed to allocate root_path\n");
        return -1;
    }

    if(create_directory_if_not_exists(root_path) != 0) {
        fprintf(stderr, "Failed to create rootfs directory %s\n", root_path);
        free(root_path);
        return -1;
    }

    char *upper_dir = join_paths(ctr_dir, "overlay_upper");
    if(!upper_dir) {
        fprintf(stderr, "Failed to allocate upper_dir\n");
        free(root_path);
        return -1;
    }

    if(create_directory_if_not_exists(upper_dir) != 0) {
        fprintf(stderr, "Failed to create overlay upper directory %s\n", upper_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }

    char *work_dir = join_paths(ctr_dir, "overlay_work");
    if(!work_dir) {
        fprintf(stderr, "Failed to allocate work_dir\n");
        free(upper_dir);
        free(root_path);
        return -1;
    }
    if(create_directory_if_not_exists(work_dir) != 0) {
        fprintf(stderr, "Failed to create overlay work directory %s\n", work_dir);
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }

    char mount_data[PATH_MAX * 3 + 64];
    snprintf(mount_data, sizeof(mount_data), "lowerdir=%s,upperdir=%s,workdir=%s", cfg->rootfs_path, upper_dir, work_dir);
    if(mount("overlay", root_path, "overlay", 0, mount_data) != 0) {
        fprintf(stderr, "Failed to mount overlayfs at %s: %m\n", root_path);
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }

    // mount /proc, /sys, /dev/null, etc.
    char *proc_path = join_paths(root_path, "proc");
    if(!proc_path) {
        fprintf(stderr, "Failed to allocate proc_path\n");
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }
    if(create_directory_if_not_exists(proc_path) != 0) {
        fprintf(stderr, "Failed to create proc directory %s\n", proc_path);
        free(proc_path);
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }
    if(mount("proc", proc_path, "proc", MS_NOSUID | MS_NOEXEC | MS_NODEV, NULL) != 0) {
        fprintf(stderr, "Failed to mount proc at %s: %m\n", proc_path);
        free(proc_path);
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }
    free(proc_path);

    char *sys_path = join_paths(root_path, "sys");
    if(!sys_path) {
        fprintf(stderr, "Failed to allocate sys_path\n");
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }
    if(create_directory_if_not_exists(sys_path) != 0) {
        fprintf(stderr, "Failed to create sys directory %s\n", sys_path);
        free(sys_path);
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }
    if(mount("sysfs", sys_path, "sysfs", MS_NOSUID | MS_NOEXEC | MS_NODEV, NULL) != 0) {
        fprintf(stderr, "Failed to mount sysfs at %s: %m\n", sys_path);
        free(sys_path);
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }
    free(sys_path);

    char *dev_path = join_paths(root_path, "dev");
    if(!dev_path) {
        fprintf(stderr, "Failed to allocate dev_path\n");
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }
    if(create_directory_if_not_exists(dev_path) != 0) {
        fprintf(stderr, "Failed to create dev directory %s\n", dev_path);
        free(dev_path);
        free(work_dir);
        free(upper_dir);
        free(root_path);
        return -1;
    }

    const char *dev_nodes[] = { "null", "zero", "random", "urandom", NULL };
    for (int i = 0; dev_nodes[i] != NULL; i++) {
        char *dev_dst = join_paths(dev_path, dev_nodes[i]);
        if (!dev_dst) {
            fprintf(stderr, "Failed to allocate dev dst path\n");
            free(dev_path);
            free(work_dir);
            free(upper_dir);
            free(root_path);
            return -1;
        }
        int fd = open(dev_dst, O_CREAT | O_CLOEXEC, 0666);
        if (fd != -1) {
            close(fd);
        }
        char dev_src[64];
        snprintf(dev_src, sizeof(dev_src), "/dev/%s", dev_nodes[i]);
        if (mount(dev_src, dev_dst, NULL, MS_BIND, NULL) != 0) {
            fprintf(stderr, "Failed to bind mount %s to %s: %m\n", dev_src, dev_dst);
            free(dev_dst);
            free(dev_path);
            free(work_dir);
            free(upper_dir);
            free(root_path);
            return -1;
        }
        free(dev_dst);
    }
    free(dev_path);

    for(size_t i = 0; i < cfg->bind_mounts_c; i++) {
        char *bind = cfg->bind_mounts[i];
        char *src = strtok(bind, ":");
        char *dst = strtok(NULL, ":");
        char *options = strtok(NULL, ":");

        if(!src || !dst) {
            fprintf(stderr, "Invalid bind mount specification: %s\n", bind);
            free(root_path);
            return -1;
        }

        char *full_dst = join_paths(root_path, dst);
        if(!full_dst) {
            fprintf(stderr, "Failed to allocate full_dst for bind mount\n");
            free(root_path);
            return -1;
        }

        if(create_directory_if_not_exists(full_dst) != 0) {
            fprintf(stderr, "Failed to create bind mount destination %s\n", full_dst);
            free(full_dst);
            free(root_path);
            return -1;
        }

        unsigned long mount_flags = MS_BIND | MS_REC;
        if(options && strcmp(options, "ro") == 0) {
            mount_flags |= MS_RDONLY;
        }

        if(mount(src, full_dst, NULL, mount_flags, NULL) != 0) {
            fprintf(stderr, "Failed to bind mount from %s to %s: %m\n", src, full_dst);
            free(full_dst);
            free(root_path);
            return -1;
        }

        free(full_dst);
    }

    free(work_dir);
    free(upper_dir);
    free(root_path);

    return 0;
}

static int drop_all_caps(void) {
    if (prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0) < 0) {
        perror("prctl(PR_CAP_AMBIENT_CLEAR_ALL)");
        return -1;
    }

    for (int cap = 0; cap <= CAP_LAST_CAP; cap++) {
        if (prctl(PR_CAPBSET_DROP, cap, 0, 0, 0) < 0) {
            if (errno != EINVAL) { // EINVAL = cap not supported
                perror("prctl(PR_CAPBSET_DROP)");
                return -1;
            }
        }
    }

    return 0;
}

static void set_limit(int resource, rlim_t lim) {
    struct rlimit rl = {
        .rlim_cur = lim,
        .rlim_max = lim * 2,
    };
    setrlimit(resource, &rl);
}

// Socket IPC helpers

static int write_byte(int fd, char b) {
    for (;;) {
        ssize_t n = write(fd, &b, 1);
        if (n == 1) return 0;
        if (n < 0 && errno == EINTR) continue;
        return -1;
    }
}

static int read_byte(int fd, char *out) {
    for (;;) {
        ssize_t n = read(fd, out, 1);
        if (n == 1) return 0;
        if (n == 0) { errno = EPIPE; return -1; }
        if (n < 0 && errno == EINTR) continue;
        return -1;
    }
}

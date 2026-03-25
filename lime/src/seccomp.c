#define _GNU_SOURCE

#include "seccomp.h"

#include <errno.h>
#include <linux/audit.h>
#include <linux/filter.h>
#include <linux/seccomp.h>
#include <stddef.h>
#include <stdio.h>
#include <sys/prctl.h>
#include <sys/syscall.h>

#define ALLOW_SYSCALL(nr) BPF_JUMP(BPF_JMP | BPF_JEQ | BPF_K, (nr), 0, 1), BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_ALLOW)

int apply_seccomp_filter(void) {
    if (prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) != 0) {
        perror("prctl(PR_SET_NO_NEW_PRIVS)");
        return -1;
    }

    struct sock_filter filter[] = {
        BPF_STMT(BPF_LD | BPF_W | BPF_ABS,
                 offsetof(struct seccomp_data, arch)),
        BPF_JUMP(BPF_JMP | BPF_JEQ | BPF_K, AUDIT_ARCH_X86_64, 1, 0),
        BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_KILL_PROCESS),

        BPF_STMT(BPF_LD | BPF_W | BPF_ABS,
                 offsetof(struct seccomp_data, nr)),

        // I/O
        ALLOW_SYSCALL(__NR_read),
        ALLOW_SYSCALL(__NR_write),
        ALLOW_SYSCALL(__NR_readv),
        ALLOW_SYSCALL(__NR_writev),
        ALLOW_SYSCALL(__NR_pread64),
        ALLOW_SYSCALL(__NR_pwrite64),
        ALLOW_SYSCALL(__NR_open),
        ALLOW_SYSCALL(__NR_openat),
        ALLOW_SYSCALL(__NR_close),
        ALLOW_SYSCALL(__NR_lseek),
        ALLOW_SYSCALL(__NR_stat),
        ALLOW_SYSCALL(__NR_fstat),
        ALLOW_SYSCALL(__NR_lstat),
        ALLOW_SYSCALL(__NR_newfstatat),
        ALLOW_SYSCALL(__NR_access),
        ALLOW_SYSCALL(__NR_faccessat),
        ALLOW_SYSCALL(__NR_getcwd),
        ALLOW_SYSCALL(__NR_readlink),
        ALLOW_SYSCALL(__NR_readlinkat),
        ALLOW_SYSCALL(__NR_dup),
        ALLOW_SYSCALL(__NR_dup2),
        ALLOW_SYSCALL(__NR_dup3),
        ALLOW_SYSCALL(__NR_fcntl),
        ALLOW_SYSCALL(__NR_ioctl),
        ALLOW_SYSCALL(__NR_pipe),
        ALLOW_SYSCALL(__NR_pipe2),
        ALLOW_SYSCALL(__NR_getdents),
        ALLOW_SYSCALL(__NR_getdents64),
        ALLOW_SYSCALL(__NR_truncate),
        ALLOW_SYSCALL(__NR_ftruncate),
        ALLOW_SYSCALL(__NR_rename),
        ALLOW_SYSCALL(__NR_renameat),
        ALLOW_SYSCALL(__NR_mkdir),
        ALLOW_SYSCALL(__NR_mkdirat),
        ALLOW_SYSCALL(__NR_rmdir),
        ALLOW_SYSCALL(__NR_unlink),
        ALLOW_SYSCALL(__NR_unlinkat),
        ALLOW_SYSCALL(__NR_symlink),
        ALLOW_SYSCALL(__NR_symlinkat),
        ALLOW_SYSCALL(__NR_link),
        ALLOW_SYSCALL(__NR_linkat),
        ALLOW_SYSCALL(__NR_chmod),
        ALLOW_SYSCALL(__NR_fchmod),
        ALLOW_SYSCALL(__NR_fchmodat),
        ALLOW_SYSCALL(__NR_sendfile),
        ALLOW_SYSCALL(__NR_chdir),
        ALLOW_SYSCALL(__NR_umask),

        // Memory management
        ALLOW_SYSCALL(__NR_brk),
        ALLOW_SYSCALL(__NR_mmap),
        ALLOW_SYSCALL(__NR_munmap),
        ALLOW_SYSCALL(__NR_mprotect),
        ALLOW_SYSCALL(__NR_mremap),
        ALLOW_SYSCALL(__NR_madvise),
        ALLOW_SYSCALL(__NR_msync),
        ALLOW_SYSCALL(__NR_mincore),
        ALLOW_SYSCALL(__NR_memfd_create),
        ALLOW_SYSCALL(__NR_mlock),
        ALLOW_SYSCALL(__NR_munlock),

        // Process and threads
        ALLOW_SYSCALL(__NR_execve),
        ALLOW_SYSCALL(__NR_execveat),
        ALLOW_SYSCALL(__NR_clone),
        ALLOW_SYSCALL(__NR_fork),
        ALLOW_SYSCALL(__NR_vfork),
        ALLOW_SYSCALL(__NR_exit),
        ALLOW_SYSCALL(__NR_exit_group),
        ALLOW_SYSCALL(__NR_wait4),
        ALLOW_SYSCALL(__NR_waitid),
        ALLOW_SYSCALL(__NR_getpid),
        ALLOW_SYSCALL(__NR_gettid),
        ALLOW_SYSCALL(__NR_getuid),
        ALLOW_SYSCALL(__NR_getgid),
        ALLOW_SYSCALL(__NR_geteuid),
        ALLOW_SYSCALL(__NR_getegid),
        ALLOW_SYSCALL(__NR_getppid),
        ALLOW_SYSCALL(__NR_getpgrp),
        ALLOW_SYSCALL(__NR_getpgid),
        ALLOW_SYSCALL(__NR_setpgid),
        ALLOW_SYSCALL(__NR_setsid),
        ALLOW_SYSCALL(__NR_getsid),
        ALLOW_SYSCALL(__NR_getgroups),
        ALLOW_SYSCALL(__NR_getresuid),
        ALLOW_SYSCALL(__NR_getresgid),

        ALLOW_SYSCALL(__NR_set_tid_address),
        ALLOW_SYSCALL(__NR_set_robust_list),
        ALLOW_SYSCALL(__NR_get_robust_list),
        ALLOW_SYSCALL(__NR_arch_prctl),

        // Futex
        ALLOW_SYSCALL(__NR_futex),

        // Signal handling
        ALLOW_SYSCALL(__NR_rt_sigaction),
        ALLOW_SYSCALL(__NR_rt_sigprocmask),
        ALLOW_SYSCALL(__NR_rt_sigreturn),
        ALLOW_SYSCALL(__NR_rt_sigpending),
        ALLOW_SYSCALL(__NR_rt_sigsuspend),
        ALLOW_SYSCALL(__NR_rt_sigtimedwait),
        ALLOW_SYSCALL(__NR_sigaltstack),
        ALLOW_SYSCALL(__NR_kill),
        ALLOW_SYSCALL(__NR_tgkill),
        ALLOW_SYSCALL(__NR_tkill),

        // Time
        ALLOW_SYSCALL(__NR_clock_gettime),
        ALLOW_SYSCALL(__NR_clock_getres),
        ALLOW_SYSCALL(__NR_clock_nanosleep),
        ALLOW_SYSCALL(__NR_gettimeofday),
        ALLOW_SYSCALL(__NR_time),
        ALLOW_SYSCALL(__NR_nanosleep),

        ALLOW_SYSCALL(__NR_prctl),
        ALLOW_SYSCALL(__NR_uname),
        ALLOW_SYSCALL(__NR_sysinfo),
        ALLOW_SYSCALL(__NR_getrlimit),
        ALLOW_SYSCALL(__NR_setrlimit),
        ALLOW_SYSCALL(__NR_prlimit64),

        ALLOW_SYSCALL(__NR_getrandom),

        ALLOW_SYSCALL(__NR_poll),
        ALLOW_SYSCALL(__NR_ppoll),
        ALLOW_SYSCALL(__NR_select),
        ALLOW_SYSCALL(__NR_pselect6),
        ALLOW_SYSCALL(__NR_epoll_create),
        ALLOW_SYSCALL(__NR_epoll_create1),
        ALLOW_SYSCALL(__NR_epoll_ctl),
        ALLOW_SYSCALL(__NR_epoll_wait),
        ALLOW_SYSCALL(__NR_epoll_pwait),

        ALLOW_SYSCALL(__NR_sched_yield),
        ALLOW_SYSCALL(__NR_sched_getaffinity),
        ALLOW_SYSCALL(__NR_sched_getscheduler),
        ALLOW_SYSCALL(__NR_sched_getparam),

        ALLOW_SYSCALL(__NR_eventfd),
        ALLOW_SYSCALL(__NR_eventfd2),
        ALLOW_SYSCALL(__NR_timerfd_create),
        ALLOW_SYSCALL(__NR_timerfd_settime),
        ALLOW_SYSCALL(__NR_timerfd_gettime),

        // Sockets — Python import machinery touches these even without
        // explicit networking code (e.g. ssl, hashlib, locale, site.py)
        ALLOW_SYSCALL(__NR_socket),
        ALLOW_SYSCALL(__NR_connect),
        ALLOW_SYSCALL(__NR_bind),
        ALLOW_SYSCALL(__NR_listen),
        ALLOW_SYSCALL(__NR_accept),
        ALLOW_SYSCALL(__NR_accept4),
        ALLOW_SYSCALL(__NR_shutdown),
        ALLOW_SYSCALL(__NR_socketpair),
        ALLOW_SYSCALL(__NR_getsockname),
        ALLOW_SYSCALL(__NR_getpeername),
        ALLOW_SYSCALL(__NR_getsockopt),
        ALLOW_SYSCALL(__NR_setsockopt),
        ALLOW_SYSCALL(__NR_sendto),
        ALLOW_SYSCALL(__NR_sendmsg),
        ALLOW_SYSCALL(__NR_recvfrom),
        ALLOW_SYSCALL(__NR_recvmsg),

        // Newer syscalls
#ifdef __NR_clone3
        ALLOW_SYSCALL(__NR_clone3),
#endif
#ifdef __NR_futex_waitv
        ALLOW_SYSCALL(__NR_futex_waitv),
#endif
#ifdef __NR_faccessat2
        ALLOW_SYSCALL(__NR_faccessat2),
#endif
#ifdef __NR_statx
        ALLOW_SYSCALL(__NR_statx),
#endif
#ifdef __NR_copy_file_range
        ALLOW_SYSCALL(__NR_copy_file_range),
#endif
#ifdef __NR_openat2
        ALLOW_SYSCALL(__NR_openat2),
#endif
#ifdef __NR_rseq
        ALLOW_SYSCALL(__NR_rseq),
#endif

        // unknown syscall -> kill
        BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_KILL_PROCESS),
    };

    struct sock_fprog prog = {
        .len = (unsigned short)(sizeof(filter) / sizeof(filter[0])),
        .filter = filter,
    };

    if (prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER, &prog) != 0) {
        perror("prctl(PR_SET_SECCOMP)");
        return -1;
    }

    return 0;
}

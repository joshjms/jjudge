#include "api.h"

#include <stdlib.h>

void free_exec_request(ExecRequest *req) {
    if (!req) return;

    free(req->id);

    if (req->args) {
        for (char **arg = req->args; *arg != NULL; arg++) {
            free(*arg);
        }
        free(req->args);
    }

    if (req->envp) {
        for (char **env = req->envp; *env != NULL; env++) {
            free(*env);
        }
        free(req->envp);
    }

    free(req->stdin);
    free(req->rootfs_path);

    if (req->bind_mounts) {
        for (char **bm = req->bind_mounts; *bm != NULL; bm++) {
            free(*bm);
        }
        free(req->bind_mounts);
    }

    free(req);
}

void free_exec_response(ExecResponse *resp) {
    if (!resp) return;

    free(resp->id);
    free(resp->stdout);
    free(resp->stderr);

    free(resp);
}

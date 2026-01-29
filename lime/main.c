#include <stdio.h>
#include <string.h>

#include "run.h"

static void print_usage(const char *prog) {
    fprintf(stderr, "Lime - A lightweight rootless container runtime\n");
}

int main(int argc, char **argv) {
    if (argc > 1 && strcmp(argv[1], "run") == 0) {
        return handle_run(argc, argv);
    }

    print_usage(argv[0]);
    return 1;
}

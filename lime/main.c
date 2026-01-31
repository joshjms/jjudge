#include <stdio.h>
#include <string.h>

#include "run.h"

const char *VERSION = "0.1.0";
const char *AUTHORS = "Joshua James";

static void print_usage(const char *prog) {
    fprintf(stderr, "Lime - A lightweight rootless container runtime\n");
}

int main(int argc, char **argv) {
    if (argc > 1 && strcmp(argv[1], "run") == 0) {
        return handle_run(argc, argv);
    } else if(argc > 1 && strcmp(argv[1], "version") == 0) {
        printf("Lime v%s by %s\n", VERSION, AUTHORS);
        return 0;
    }

    print_usage(argv[0]);
    return 1;
}

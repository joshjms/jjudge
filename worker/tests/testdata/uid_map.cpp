// Reads /proc/self/uid_map and writes it to stdout.
// Inside a lime user-namespace sandbox the output is:
//   "         0   <slot_uid>          1"
// The middle field is the outer (host) uid that lime was exec'd as.
#include <cstdio>

int main() {
    FILE* f = fopen("/proc/self/uid_map", "r");
    if (!f) { perror("open uid_map"); return 1; }
    char buf[256];
    while (fgets(buf, sizeof(buf), f)) {
        fputs(buf, stdout);
    }
    fclose(f);
    return 0;
}

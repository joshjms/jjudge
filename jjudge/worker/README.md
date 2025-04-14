
# 🤖 worker

**`worker`** is a lightweight runner agent designed to execute untrusted code as a Kubernetes [job](). It is spawned by the `controller` service as part of a secure code execution platform.

> **NOTE: `worker` does NOT provide isolation.**

---

## 🚩 Flag Configurations

| Flag         | Shorthand | Type     | Default   | Description                                  |
|--------------|-----------|----------|-----------|----------------------------------------------|
| `--dir`      | `-d`      | string   | `/data/c` | Directory of the source code                 |
| `--stdin`    | `-i`      | string   | `/data/in`| Input file                                   |
| `--lang`     |           | string   | `cpp`     | Language of the code                         |
| `--ml`       |           | int64    | `256`     | Memory limit of the running code (in MB)     |
| `--tl`       |           | int64    | `1`       | Time limit of the running code (in seconds)  |
| `--proc`     |           | int      | `1`       | Maximum number of processes                  |
| `--uid`      | `-u`      | uint32   | `65534`   | UID of the process to monitor                |
| `--gid`      | `-g`      | uint32   | `65534`   | GID of the process to monitor                |

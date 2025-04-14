package config

type Config struct {
	Dir          string // Code to be executed (e.g., /tmp/code.cpp)
	Stdin        string // Input to be passed to the program (e.g., /tmp/input.txt)
	Language     string // Language of the code (e.g., cpp, python)
	MemoryLimit  int64  // Memory limit of the running code (in MB)
	TimeLimit    int64  // Time limit of the running code (in seconds)
	MaxProcesses int    // Maximum number of processes
	UID          uint32 // UID of the process
	GID          uint32 // GID of the process
}

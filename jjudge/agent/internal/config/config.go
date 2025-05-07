package config

type Config struct {
	AgentID  string `mapstructure:"agent_id"`
	LogLevel string `mapstructure:"log_level"`

	MQ MQConfig `mapstructure:"mq"`

	Sandbox SandboxConfig `mapstructure:"sandbox"`

	ResourceLimits ResourceLimits `mapstructure:"resources"`

	Cgroup CgroupConfig `mapstructure:"cgroups"`
}

type MQConfig struct {
	URL      string `mapstructure:"url"`
	Queue    string `mapstructure:"queue"`
	Prefetch int    `mapstructure:"prefetch"`
}

type SandboxConfig struct {
	RuntimePath            string `mapstructure:"runtime_path"` // e.g., /usr/local/bin/runsc
	BaseRootFS             string `mapstructure:"base_rootfs"`  // Path to unpacked Alpine/etc
	WorkingDir             string `mapstructure:"working_dir"`  // e.g., /tmp/jobs
	MaxConcurrentSandboxes int    `mapstructure:"max_concurrent_sandboxes"`
	TimeoutSec             int    `mapstructure:"timeout_sec"`
	CleanupAfterJob        bool   `mapstructure:"cleanup"`
}

// For hard limits only
type ResourceLimits struct {
	MemoryMB int64 `mapstructure:"memory_mb"` // Max memory in MB (cgroup or RLIMIT_AS)

	CPUMicros int64 `mapstructure:"cpu_micros"` // CPU time in microseconds (cgroup or RLIMIT_CPU)

	Processes int `mapstructure:"max_processes"` // Max number of processes (cgroup or RLIMIT_NPROC)

	OpenFileLimit int   `mapstructure:"open_file_limit"`  // Max number of open files (cgroup or RLIMIT_NOFILE)
	FSSizeLimitMB int64 `mapstructure:"fs_size_limit_mb"` // Total disk space quota in MB

	StdoutLimitKB int64 `mapstructure:"stdout_limit_kb"` // Max stdout size before truncation
	StderrLimitKB int64 `mapstructure:"stderr_limit_kb"` // Max stderr size before truncation

	HardTimeoutSec int `mapstructure:"hard_timeout_sec"` // Max job execution time in seconds
}

type CgroupConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	UseSystemd   bool   `mapstructure:"use_systemd"`
	CgroupRoot   string `mapstructure:"cgroup_root"` // e.g., /sys/fs/cgroup/unified
	CgroupPrefix string `mapstructure:"prefix"`      // e.g., sandbox-job
}

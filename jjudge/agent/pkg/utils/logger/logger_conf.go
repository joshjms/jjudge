package logger

// Logger configuration
type Config struct {
	// Level can be "debug", "info", "warn", or "error".
	Level string

	// Format can be "json" or "console".
	Format string

	// OutputPath can be a file path or "stdout" for standard output.
	// If empty, it defaults to "stdout".
	OutputPath string
}

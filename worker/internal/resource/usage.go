package resource

type Usage struct {
	Memory        int64   // in bytes
	UserCpuTime   float64 // in seconds
	SystemCpuTime float64 // in seconds
}

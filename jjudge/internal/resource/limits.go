package resource

type Limits struct {
	Memory    int64 // in MB
	Time      int64 // in seconds
	Processes int
	OpenFiles int
	Filesize  int64
	Stack     int64
}

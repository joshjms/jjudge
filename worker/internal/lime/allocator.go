package lime

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
)

type Slot struct {
	CPUs string
	Mems string
}

type SlotPool struct {
	ch chan Slot
}

type SlotPoolOption func(*SlotPool)

func NewSlotPool(opts ...SlotPoolOption) *SlotPool {
	sp := &SlotPool{}
	for _, opt := range opts {
		opt(sp)
	}

	if sp.ch == nil {
		WithCPUs("")(sp)
	}

	return sp
}

// WithCPUs configures the slot pool from a CPU set string (e.g. "0-3", "0,2,4").
// One slot is created per CPU in the set, each pinned to its CPU.
// An empty string means no pinning; the number of slots equals runtime.NumCPU().
func WithCPUs(cpuSet string) SlotPoolOption {
	return func(sp *SlotPool) {
		mems := parseCPUSet(strings.TrimSpace(os.Getenv("CASTLETOWN_MEMS")))
		if len(mems) == 0 {
			mems = []string{"0"}
		}

		cpus := parseCPUSet(cpuSet)
		n := len(cpus)
		if n == 0 {
			n = runtime.NumCPU()
		}

		sp.ch = make(chan Slot, n)
		for i := 0; i < n; i++ {
			cpu := ""
			if len(cpus) > 0 {
				cpu = cpus[i]
			}
			sp.ch <- Slot{
				CPUs: cpu,
				Mems: mems[i%len(mems)],
			}
		}
	}
}

// CPUCount returns the number of CPUs described by cpuSet.
// An empty string returns runtime.NumCPU().
func CPUCount(cpuSet string) int {
	cpus := parseCPUSet(cpuSet)
	if len(cpus) == 0 {
		return runtime.NumCPU()
	}
	return len(cpus)
}

type Allocation struct {
	pool     *SlotPool
	slot     Slot
	released atomic.Bool
}

func (sp *SlotPool) Allocate(ctx context.Context) (*Allocation, error) {
	select {
	case r := <-sp.ch:
		return &Allocation{pool: sp, slot: r}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (a *Allocation) Release() {
	if a == nil || a.pool == nil {
		return
	}
	if !a.released.CompareAndSwap(false, true) {
		return
	}
	a.pool.ch <- a.slot
}

func parseCPUSet(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				return nil
			}
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil
			}
			if start > end {
				return nil
			}
			for i := start; i <= end; i++ {
				out = append(out, strconv.Itoa(i))
			}
			continue
		}

		if _, err := strconv.Atoi(part); err != nil {
			return nil
		}
		out = append(out, part)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

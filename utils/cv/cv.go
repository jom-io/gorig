package cv

import "time"

func Str(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func Int(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func Int32(i int32) *int32 {
	if i == int32(0) {
		return nil
	}
	return &i
}

func Int64(i int64) *int64 {
	if i == int64(0) {
		return nil
	}
	return &i
}

func Bool(b bool) *bool {
	return &b
}

func Float32(f float32) *float32 {
	if f == float32(0) {
		return nil
	}
	return &f
}

func Float64(f float64) *float64 {
	if f == float64(0) {
		return nil
	}
	return &f
}

func Time(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func Duration(d time.Duration) *time.Duration {
	if d == 0 {
		return nil
	}
	return &d
}

func N[T any](o T) *T {
	return &o
}

// Package concat compares different ways of joining strings in Go.
package concat

import (
	"bytes"
	"fmt"
	"strings"
)

// Plus joins parts with the + operator. Every iteration copies the whole
// result so far into a new string so the cost grows quadratically
func Plus(parts []string) string {
	var s string
	for _, p := range parts {
		s += p
	}
	return s
}

// Sprintf joins parts with fmt.Sprintf. It has the same quadratic copying
// as Plus, plus the overhead of format parsing and interface boxing
func Sprintf(parts []string) string {
	var s string
	for _, p := range parts {
		s = fmt.Sprintf("%s%s", s, p)
	}
	return s
}

// Builder joins parts with a strings.Builder that grows on demand
func Builder(parts []string) string {
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

// BuilderGrow joins parts with a strings.Builder that is sized up front
// so it allocates exactly once
func BuilderGrow(parts []string) string {
	n := 0
	for _, p := range parts {
		n += len(p)
	}

	var sb strings.Builder
	sb.Grow(n)
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

// BytesBuffer joins parts with a bytes.Buffer. Converting the buffer to a
// string at the end costs an extra copy that strings.Builder avoids
func BytesBuffer(parts []string) string {
	var buf bytes.Buffer
	for _, p := range parts {
		buf.WriteString(p)
	}
	return buf.String()
}

// Join joins parts with strings.Join which presizes its buffer internally
func Join(parts []string) string {
	return strings.Join(parts, "")
}

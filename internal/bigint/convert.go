package bigint

import (
	"math"
	"math/big"
	"strings"
	"unicode/utf8"
)

const maxUint = ^uint(0)
const maxInt = int(maxUint >> 1)
const minInt = -maxInt - 1

// ToInt converts a *big.Int x to an int and returns whether x can be
// contained within int.
func ToInt(x *big.Int) (int, bool) {
	i64 := x.Int64()
	return int(i64), x.IsInt64() && int64(minInt) <= i64 && i64 <= int64(maxInt)
}

// ToUint converts a *big.Int x to a uint and returns whether x can be
// contained within uint.
func ToUint(x *big.Int) (uint, bool) {
	u64 := x.Uint64()
	return uint(u64), x.IsUint64() && u64 <= uint64(maxUint)
}

// ToInt64 converts a *big.Int to an int64 and returns whether x can be
// contained within int64.
func ToInt64(x *big.Int) (int64, bool) {
	return x.Int64(), x.IsInt64()
}

// ToUint64 converts a *big.Int to a uint64 and returns whether x can be
// contained within uint64.
func ToUint64(x *big.Int) (uint64, bool) {
	return x.Uint64(), x.IsUint64()
}

// ToInt32 converts a *big.Int to an int32 and returns whether x can be
// contained within int32.
func ToInt32(x *big.Int) (int32, bool) {
	i64 := x.Int64()
	return int32(i64), x.IsInt64() && math.MinInt32 <= i64 && i64 <= math.MaxInt32
}

// ToUint32 converts a *big.Int to a uint32 and returns whether x can be
// contained within uint32.
func ToUint32(x *big.Int) (uint32, bool) {
	u64 := x.Uint64()
	return uint32(u64), x.IsUint64() && u64 <= math.MaxUint32
}

// ToRune converts a *big.Int x to a rune. When x is not a valid UTF-8
// codepoint, U+FFFD � replacement character is returned.
func ToRune(x *big.Int) rune {
	i32, ok := ToInt32(x)
	if ok && utf8.ValidRune(i32) {
		return i32
	}
	return '\uFFFD'
}

// FormatSlice formats a slice of *big.Int to a space separated string.
func FormatSlice(s []*big.Int) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range s {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteString(x.String())
	}
	b.WriteByte(']')
	return b.String()
}

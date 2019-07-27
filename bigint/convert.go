package bigint

import (
	"math"
	"math/big"
	"unicode/utf8"
)

const maxInt int = int(^uint(0) >> 1)

// ToInt converts a *big.Int x to an int and returns whether x can be
// contained within int.
func ToInt(x *big.Int) (int, bool) {
	if !x.IsInt64() {
		return 0, false
	}
	i64 := x.Int64()
	if i64 > int64(maxInt) {
		return 0, false
	}
	return int(i64), true
}

// ToInt64 converts a *big.Int to an int64 and returns whether x can be
// contained within int64.
func ToInt64(x *big.Int) (int64, bool) {
	if !x.IsInt64() {
		return 0, false
	}
	return x.Int64(), true
}

// ToInt32 converts a *big.Int to an int32 and returns whether x can be
// contained within int32.
func ToInt32(x *big.Int) (int32, bool) {
	if !x.IsInt64() {
		return 0, false
	}
	i32 := x.Int64()
	if i32 > math.MaxInt32 {
		return 0, false
	}
	return int32(i32), true
}

// ToRune converts a *big.Int x to a rune. When x is not a valid UTF-8
// codepoint, U+FFFD � replacement character is returned.
func ToRune(x *big.Int) rune {
	i32, ok := ToInt32(x)
	if !ok || !utf8.ValidRune(i32) {
		return '\uFFFD'
	}
	return i32
}

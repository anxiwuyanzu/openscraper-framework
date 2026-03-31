package atof

func lower(c byte) byte {
	return c | ('x' - 'X')
}

// underscoreOK reports whether the underscores in s are allowed.
// Checking them in this one function lets all the parsers skip over them simply.
// Underscore must appear only between digits or between a base prefix and a digit.
func underscoreOK(s string) bool {
	// saw tracks the last character (class) we saw:
	// ^ for beginning of number,
	// 0 for a digit or base prefix,
	// _ for an underscore,
	// ! for none of the above.
	saw := '^'
	i := 0

	// Optional sign.
	if len(s) >= 1 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}

	// Optional base prefix.
	hex := false
	if len(s) >= 2 && s[0] == '0' && (lower(s[1]) == 'b' || lower(s[1]) == 'o' || lower(s[1]) == 'x') {
		i = 2
		saw = '0' // base prefix counts as a digit for "underscore as digit separator"
		hex = lower(s[1]) == 'x'
	}

	// Number proper.
	for ; i < len(s); i++ {
		// Digits are always okay.
		if '0' <= s[i] && s[i] <= '9' || hex && 'a' <= lower(s[i]) && lower(s[i]) <= 'f' {
			saw = '0'
			continue
		}
		// Underscore must follow digit.
		if s[i] == '_' {
			if saw != '0' {
				return false
			}
			saw = '_'
			continue
		}
		// Underscore must also be followed by digit.
		if saw == '_' {
			return false
		}
		// Saw non-digit, non-underscore.
		saw = '!'
	}
	return saw != '_'
}

func readFloat(s string) (mantissa uint64, exp int, neg, trunc, hex bool, i int, ok bool) {
	underscores := false

	// optional sign
	if i >= len(s) {
		return
	}
	switch {
	case s[i] == '+':
		i++
	case s[i] == '-':
		neg = true
		i++
	}

	// digits
	base := uint64(10)
	maxMantDigits := 19 // 10^19 fits in uint64
	expChar := byte('e')
	if i+2 < len(s) && s[i] == '0' && lower(s[i+1]) == 'x' {
		base = 16
		maxMantDigits = 16 // 16^16 fits in uint64
		i += 2
		expChar = 'p'
		hex = true
	}
	sawdot := false
	sawdigits := false
	nd := 0
	ndMant := 0
	dp := 0
loop:
	for ; i < len(s); i++ {
		switch c := s[i]; true {
		case c == '_':
			underscores = true
			continue

		case c == '.':
			if sawdot {
				break loop
			}
			sawdot = true
			dp = nd
			continue

		case '0' <= c && c <= '9':
			sawdigits = true
			if c == '0' && nd == 0 { // ignore leading zeros
				dp--
				continue
			}
			nd++
			if ndMant < maxMantDigits {
				mantissa *= base
				mantissa += uint64(c - '0')
				ndMant++
			} else if c != '0' {
				trunc = true
			}
			continue

		case base == 16 && 'a' <= lower(c) && lower(c) <= 'f':
			sawdigits = true
			nd++
			if ndMant < maxMantDigits {
				mantissa *= 16
				mantissa += uint64(lower(c) - 'a' + 10)
				ndMant++
			} else {
				trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		return
	}
	if !sawdot {
		dp = nd
	}

	if base == 16 {
		dp *= 4
		ndMant *= 4
	}

	// optional exponent moves decimal point.
	// if we read a very large, very long number,
	// just be sure to move the decimal point by
	// a lot (say, 100000).  it doesn't matter if it's
	// not the exact number.
	if i < len(s) && lower(s[i]) == expChar {
		i++
		if i >= len(s) {
			return
		}
		esign := 1
		if s[i] == '+' {
			i++
		} else if s[i] == '-' {
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				underscores = true
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		dp += e * esign
	} else if base == 16 {
		// Must have exponent.
		return
	}

	if mantissa != 0 {
		exp = dp - ndMant
	}

	if underscores && !underscoreOK(s[:i]) {
		return
	}

	ok = true
	return
}

// Exact powers of 10.
var float64pow10 = []float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
	1e20, 1e21, 1e22,
}

func Atof64(s string) (f float64, n int) {
	mantissa, exp, neg, _, _, n, _ := readFloat(s)
	f = float64(mantissa)
	if neg {
		f = -f
	}
	switch {
	case exp == 0:
		// an integer.
		return f, n
	// Exact integers are <= 10^15.
	// Exact powers of ten are <= 10^22.
	case exp > 0 && exp <= 15+22: // int * 10^k
		// If exponent is big but number of digits is not,
		// can move a few zeros into the integer part.
		if exp > 22 {
			f *= float64pow10[exp-22]
			exp = 22
		}
		if f > 1e15 || f < -1e15 {
			// the exponent was really too large.
			return
		}
		return f * float64pow10[exp], n
	case exp < 0 && exp >= -22: // int / 10^k
		return f / float64pow10[-exp], n
	}
	return
}

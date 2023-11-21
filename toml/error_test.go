//go:build go1.16
// +build go1.16

package toml_test

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/bloom42/stdx/toml"
	tomltest "github.com/bloom42/stdx/toml/internal/toml-test"
)

func TestErrorPosition(t *testing.T) {
	// Note: take care to use leading spaces (not tabs).
	tests := []struct {
		test, err string
	}{
		{"array/missing-separator.toml", `
toml: error: expected a comma (',') or array terminator (']'), but got '2'

At line 1, column 13:

      1 | wrong = [ 1 2 3 ]
                      ^`},

		{"array/no-close-2.toml", `
toml: error: expected a comma (',') or array terminator (']'), but got end of file

At line 1, column 10:

      1 | x = [42 #
                   ^`},

		{"array/tables-2.toml", `
toml: error: Key 'fruit.variety' has already been defined.

At line 9, column 3-8:

      7 |
      8 |   # This table conflicts with the previous table
      9 |   [fruit.variety]
             ^^^^^`},
		{"datetime/trailing-t.toml", `
toml: error: Invalid TOML Datetime: "2006-01-30T".

At line 2, column 4-15:

      1 | # Date cannot end with trailing T
      2 | d = 2006-01-30T
              ^^^^^^^^^^^`},
	}

	fsys := tomltest.EmbeddedTests()
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			input, err := fs.ReadFile(fsys, "invalid/"+tt.test)
			if err != nil {
				t.Fatal(err)
			}

			var x interface{}
			_, err = toml.Decode(string(input), &x)
			if err == nil {
				t.Fatal("err is nil")
			}

			var pErr toml.ParseError
			if !errors.As(err, &pErr) {
				t.Errorf("err is not a ParseError: %T %[1]v", err)
			}

			tt.err = tt.err[1:] + "\n" // Remove first newline, and add trailing.
			want := pErr.ErrorWithUsage()

			if !strings.Contains(want, tt.err) {
				t.Fatalf("\nwant:\n%s\nhave:\n%s", tt.err, want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		in        interface{}
		toml, err string
	}{
		{
			&struct{ Int int8 }{},
			"Int = 200",
			`| toml: error: 200 is out of range for int8
			 |
			 | At line 1, column 6-9:
			 |
			 |       1 | Int = 200
			 |                 ^^^
			 | Error help:
             |
			 |     This number is too large; this may be an error in the TOML, but it can also be a
			 |     bug in the program that uses too small of an integer.
             |
			 |     The maximum and minimum values are:
             |
			 |         size   │ lowest         │ highest
			 |         ───────┼────────────────┼──────────
			 |         int8   │ -128           │ 127
			 |         int16  │ -32,768        │ 32,767
			 |         int32  │ -2,147,483,648 │ 2,147,483,647
			 |         int64  │ -9.2 × 10¹⁷    │ 9.2 × 10¹⁷
			 |         uint8  │ 0              │ 255
			 |         uint16 │ 0              │ 65535
			 |         uint32 │ 0              │ 4294967295
			 |         uint64 │ 0              │ 1.8 × 10¹⁸
             |
			 |     int refers to int32 on 32-bit systems and int64 on 64-bit systems.
			`,
		},
		{
			&struct{ Int int }{},
			fmt.Sprintf("Int = %d", uint64(math.MaxInt64+1)),
			`| toml: error: 9223372036854775808 is out of range for int64
			 |
			 | At line 1, column 6-25:
			 |
			 |       1 | Int = 9223372036854775808
			 |                 ^^^^^^^^^^^^^^^^^^^
			 | Error help:
             |
			 |     This number is too large; this may be an error in the TOML, but it can also be a
			 |     bug in the program that uses too small of an integer.
             |
			 |     The maximum and minimum values are:
             |
			 |         size   │ lowest         │ highest
			 |         ───────┼────────────────┼──────────
			 |         int8   │ -128           │ 127
			 |         int16  │ -32,768        │ 32,767
			 |         int32  │ -2,147,483,648 │ 2,147,483,647
			 |         int64  │ -9.2 × 10¹⁷    │ 9.2 × 10¹⁷
			 |         uint8  │ 0              │ 255
			 |         uint16 │ 0              │ 65535
			 |         uint32 │ 0              │ 4294967295
			 |         uint64 │ 0              │ 1.8 × 10¹⁸
             |
			 |     int refers to int32 on 32-bit systems and int64 on 64-bit systems.
			`,
		},
		{
			&struct{ Float float32 }{},
			"Float = 1.1e99",
			`
            | toml: error: 1.1e+99 is out of range for float32
            |
            | At line 1, column 8-14:
            |
            |       1 | Float = 1.1e99
            |                   ^^^^^^
            | Error help:
            |
            |     This number is too large; this may be an error in the TOML, but it can also be a
            |     bug in the program that uses too small of an integer.
            |
            |     The maximum and minimum values are:
            |
            |         size   │ lowest         │ highest
            |         ───────┼────────────────┼──────────
            |         int8   │ -128           │ 127
            |         int16  │ -32,768        │ 32,767
            |         int32  │ -2,147,483,648 │ 2,147,483,647
            |         int64  │ -9.2 × 10¹⁷    │ 9.2 × 10¹⁷
            |         uint8  │ 0              │ 255
            |         uint16 │ 0              │ 65535
            |         uint32 │ 0              │ 4294967295
            |         uint64 │ 0              │ 1.8 × 10¹⁸
            |
            |     int refers to int32 on 32-bit systems and int64 on 64-bit systems.
			`,
		},

		{
			&struct{ D time.Duration }{},
			`D = "99 bottles"`,
			`
			| toml: error: invalid duration: "99 bottles"
			|
			| At line 1, column 5-15:
			|
			|       1 | D = "99 bottles"
			|                ^^^^^^^^^^
			| Error help:
			|
			|     A duration must be as "number<unit>", without any spaces. Valid units are:
			|
			|         ns         nanoseconds (billionth of a second)
			|         us, µs     microseconds (millionth of a second)
			|         ms         milliseconds (thousands of a second)
			|         s          seconds
			|         m          minutes
			|         h          hours
			|
			|     You can combine multiple units; for example "5m10s" for 5 minutes and 10
			|     seconds.
			`,
		},
	}

	prep := func(s string) string {
		lines := strings.Split(strings.TrimSpace(s), "\n")
		for i := range lines {
			if j := strings.IndexByte(lines[i], '|'); j >= 0 {
				lines[i] = lines[i][j+1:]
				lines[i] = strings.Replace(lines[i], " ", "", 1)
			}
		}
		return strings.Join(lines, "\n") + "\n"
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			_, err := toml.Decode(tt.toml, tt.in)
			if err == nil {
				t.Fatalf("err is nil; decoded: %#v", tt.in)
			}

			var pErr toml.ParseError
			if !errors.As(err, &pErr) {
				t.Fatalf("err is not a ParseError: %#v", err)
			}

			tt.err = prep(tt.err)
			have := pErr.ErrorWithUsage()

			// have = strings.ReplaceAll(have, " ", "·")
			// tt.err = strings.ReplaceAll(tt.err, " ", "·")
			if have != tt.err {
				t.Fatalf("\nwant:\n%s\nhave:\n%s", tt.err, have)
			}
		})
	}
}

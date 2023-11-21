package countries

import (
	"database/sql/driver"
	"fmt"
)

// CountryInt32 is a tpye that store a 2-letters country code as an int32.
// The goal is to have more efficient database operations than using the a basic TEXT
type CountryInt32 string

// Scan implements sql.Scanner so CountryInt32 can be read from databases transparently.
func (country *CountryInt32) Scan(src interface{}) (err error) {
	switch src := src.(type) {
	case int32:
		value := make([]rune, 2)
		value[0] = rune(src >> 8)
		value[1] = rune(int8(src))
		*country = CountryInt32(string(value))

	default:
		return fmt.Errorf("Scan: unable to scan type %T into CountryInt32", src)
	}
	return
}

// Value implements sql.Valuer so that CountryInt32 can be written to databases
func (country CountryInt32) Value() (driver.Value, error) {
	var value int32

	value |= int32(country[0]) << 8
	value |= int32(country[1])

	return value, nil
}

package countries

import "testing"

func TestCountryInt32Value(t *testing.T) {
	data := []CountryInt32{
		"AA",
		"AB",
		"ZY",
		"ZZ",
	}
	expected := []int32{
		0x4141,
		0x4142,
		0x5A59,
		0x5A5A,
	}

	for i, elem := range data {
		v, _ := elem.Value()
		if v != expected[i] {
			t.Errorf("value (0x%x) != expected (0x%x)", v, expected[i])
		}
	}
}

func TestCountryInt32Scan(t *testing.T) {
	data := []int32{
		0x4141,
		0x4142,
		0x5A59,
		0x5A5A,
	}
	expected := []CountryInt32{
		"AA",
		"AB",
		"ZY",
		"ZZ",
	}

	for i, elem := range data {
		var country CountryInt32
		err := country.Scan(elem)
		if err != nil {
			t.Errorf("Scanning: %x: %s", elem, err)
		}
		if country != expected[i] {
			t.Errorf("value (%s) != expected (%s)", country, expected[i])
		}
	}
}

package countries

import (
	_ "embed"
	"encoding/json"
	"errors"
)

//go:embed countries.json
var Bytes []byte

const (
	Unknown     = "Unknown"
	CodeUnknown = "XX"
)

type Country struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

var countriesMap map[string]Country
var countriesList []Country

var (
	ErrCountryNotFound = errors.New("Country not found")
)

func GetMap() (map[string]Country, error) {
	if countriesMap == nil {
		list, err := GetList()
		if err != nil {
			return countriesMap, err
		}

		countriesMap = map[string]Country{}
		for _, country := range list {
			countriesMap[country.Code] = country
		}
	}

	return countriesMap, nil
}

func GetList() ([]Country, error) {
	var err error

	if countriesList == nil {
		countriesList = []Country{}

		err = json.Unmarshal(Bytes, &countriesList)
		if err != nil {

			return countriesList, err
		}
	}

	return countriesList, nil
}

func Name(countryCode string) (countryName string, err error) {
	countryName = Unknown

	if countriesMap == nil {
		_, err = GetMap()
		if err != nil {
			return
		}
	}

	country, exists := countriesMap[countryCode]
	if !exists {
		err = ErrCountryNotFound
		return
	}

	countryName = country.Name
	return
}

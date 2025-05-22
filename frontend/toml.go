package frontend

import (
	"github.com/BurntSushi/toml"
	"github.com/go-playground/validator/v10"
)

type GluaxToml struct {
	Name    string `toml:"name" validate:"required"`
	Version string `toml:"version" validate:"required"`
	Lib     bool   `toml:"lib"`
	Std     bool   `toml:"std"`
}

func HandleGluaxToml(tomlContent string) (GluaxToml, error) {
	var gt GluaxToml
	_, err := toml.Decode(tomlContent, &gt)
	if err != nil {
		return gt, err
	}
	validate := validator.New()
	if err := validate.Struct(gt); err != nil {
		return gt, err
	}
	return gt, nil
}

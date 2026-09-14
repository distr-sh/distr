package main

import (
	"gopkg.in/yaml.v3"
)

func DecodeComposeFile(manifest []byte) (result map[string]any, err error) {
	err = yaml.Unmarshal(manifest, &result)
	return result, err
}

func EncodeComposeFile(compose map[string]any) (result []byte, err error) {
	return yaml.Marshal(compose)
}

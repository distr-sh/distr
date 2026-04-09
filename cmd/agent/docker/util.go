package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func DecodeComposeFile(manifest []byte) (result map[string]any, err error) {
	err = yaml.Unmarshal(manifest, &result)
	return result, err
}

func EncodeComposeFile(compose map[string]any) (result []byte, err error) {
	return yaml.Marshal(compose)
}

type tempFile string

func (f tempFile) Destroy() {
	if err := os.Remove(string(f)); err != nil {
		logger.Warn("failed to destroy temp file", zap.String("fileName", string(f)), zap.Error(err))
	}
}

func WriteTempFile(pattern string, data []byte) (tempFile, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("failed to write data to temp file: %w", err)
	}

	return tempFile(f.Name()), nil
}

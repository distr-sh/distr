package main

import "github.com/distr-sh/distr/internal/controllerenv"

func ScratchDir() string {
	if dir := controllerenv.Get("SCRATCH_DIR"); dir != "" {
		return dir
	}
	return "./scratch"
}

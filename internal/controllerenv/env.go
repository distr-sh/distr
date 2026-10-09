package controllerenv

import (
	"strconv"
	"time"

	"github.com/distr-sh/distr/internal/envparse"
	"github.com/distr-sh/distr/internal/envutil"
)

var (
	ControllerVersionID = envutil.GetEnv("DISTR_CONTROLLER_VERSION_ID")
	Interval            = envutil.GetEnvParsedOrDefault(
		"DISTR_CONTROLLER_INTERVAL", envparse.PositiveDuration, 5*time.Second,
	)
	ProgressingInterval    = 3 * Interval
	DistrRegistryHost      = envutil.GetEnv("DISTR_REGISTRY_HOST")
	DistrRegistryPlainHTTP = envutil.GetEnvParsedOrDefault("DISTR_REGISTRY_PLAIN_HTTP", strconv.ParseBool, false)
)

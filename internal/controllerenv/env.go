package controllerenv

import (
	"cmp"
	"os"
	"strconv"
	"time"

	"github.com/distr-sh/distr/internal/envparse"
	"github.com/distr-sh/distr/internal/envutil"
)

var (
	ControllerVersionID    = Get("VERSION_ID")
	Interval               = envutil.GetEnvParsedOrDefault("DISTR_INTERVAL", envparse.PositiveDuration, 5*time.Second)
	DistrRegistryHost      = envutil.GetEnv("DISTR_REGISTRY_HOST")
	DistrRegistryPlainHTTP = envutil.GetEnvParsedOrDefault("DISTR_REGISTRY_PLAIN_HTTP", strconv.ParseBool, false)
)

// Get returns DISTR_CONTROLLER_<name>, or DISTR_AGENT_<name> when the former is empty. Manifests of targets
// created before the rename to controller only ever set the agent name.
func Get(name string) string {
	return cmp.Or(os.Getenv("DISTR_CONTROLLER_"+name), os.Getenv("DISTR_AGENT_"+name))
}

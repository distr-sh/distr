package types

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseDockerEndpoint(t *testing.T) {
	valid := map[string]string{
		"unix:///var/run/docker.sock":                  "/var/run/docker.sock",
		"unix:///Users/you/.socktainer/container.sock": "/Users/you/.socktainer/container.sock",
		"unix:///run/user/1000/docker.sock":            "/run/user/1000/docker.sock",
	}
	for endpoint, expected := range valid {
		t.Run(endpoint, func(t *testing.T) {
			g := NewWithT(t)
			socketPath, err := ParseDockerEndpoint(endpoint)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(socketPath).To(Equal(expected))
		})
	}

	invalid := []string{
		"",
		"/var/run/docker.sock",
		"tcp://localhost:2375",
		"ssh://user@host",
		"unix://var/run/docker.sock",
		"unix://",
		"unix:///tmp/docker:desktop.sock",
		"unix:///tmp/docker%3Adesktop.sock",
		"unix:///tmp/docker desktop.sock",
	}
	for _, endpoint := range invalid {
		t.Run(endpoint, func(t *testing.T) {
			g := NewWithT(t)
			_, err := ParseDockerEndpoint(endpoint)
			g.Expect(err).To(HaveOccurred())
		})
	}
}

func TestDockerSocketPath(t *testing.T) {
	g := NewWithT(t)

	dt := DeploymentTarget{Type: DeploymentTypeDocker}
	g.Expect(dt.DockerSocketPath()).To(Equal(DefaultDockerSocketPath))

	dt.DockerEndpoint = new("unix:///run/user/1000/docker.sock")
	g.Expect(dt.DockerSocketPath()).To(Equal("/run/user/1000/docker.sock"))
}

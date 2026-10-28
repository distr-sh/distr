package supportbundle_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/supportbundle"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

const (
	collectScriptHeredocStart = "<< 'DISTR_COLLECT_EOF'\n"
	collectScriptHeredocEnd   = "\nDISTR_COLLECT_EOF\n"
)

// The collector doing the actual work sits inside a quoted heredoc, so nothing parses it until a
// customer runs it on their host. Render the collector of every deployment type both with and
// without custom scripts and check that each one is valid shell.
func TestGenerateCollectScript_syntax(t *testing.T) {
	for _, deploymentType := range []types.DeploymentType{types.DeploymentTypeDocker, types.DeploymentTypeKubernetes} {
		for name, scripts := range map[string][]types.SupportBundleConfigurationScript{
			"without custom scripts": nil,
			"with custom scripts": {
				{Name: "disk usage", Description: new("what is on disk"), Content: "df -h\n"},
				{Name: "plain", Content: "echo hello\n"},
			},
		} {
			t.Run(string(deploymentType)+" "+name, func(t *testing.T) {
				g := NewWithT(t)
				script, err := supportbundle.GenerateCollectScript(supportbundle.CollectScriptParams{
					DeploymentType:       deploymentType,
					BaseURL:              "https://app.example.com",
					BundleID:             uuid.New(),
					BundleSecret:         "secret",
					EnvVars:              []types.SupportBundleConfigurationEnvVar{{Name: "DB_HOST"}},
					Scripts:              scripts,
					LogTailLines:         100,
					ResourceMaxBytes:     1024,
					ScriptTimeout:        time.Minute,
					CollectorDownloadURL: "https://github.com/distr-sh/distr/releases/latest/download",
				})
				g.Expect(err).ToNot(HaveOccurred())

				expectValidShellSyntax(t, g, script)
				expectValidShellSyntax(t, g, innerCollectScript(g, script))
			})
		}
	}
}

// The Kubernetes collect script runs a binary it downloads, so it must refuse one whose checksum
// does not match, and leave the bundle open when it does not get to run the collector.
func TestGenerateCollectScript_kubernetesCollector(t *testing.T) {
	platform := runtime.GOOS + "-" + runtime.GOARCH
	if !slices.Contains([]string{"linux-amd64", "linux-arm64", "darwin-amd64", "darwin-arm64"}, platform) {
		t.Skipf("no collector is released for %s", platform)
	}
	for _, tool := range []string{"sh", "curl", "tar"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not available", tool)
		}
	}
	if _, err := exec.LookPath("sha256sum"); err != nil {
		if _, err := exec.LookPath("shasum"); err != nil {
			t.Skip("neither sha256sum nor shasum is available")
		}
	}
	archiveName := "kubernetes-support-bundle-" + platform + ".tar.gz"

	for name, tampered := range map[string]bool{"verified": false, "tampered": true} {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			dir := t.TempDir()
			archive := collectorArchive(g, "#!/bin/sh\n"+
				"printf '%s\\n' \"$@\" > '"+dir+"/args'\n"+
				"printf '%s\\n' \"$DISTR_SUPPORT_BUNDLE_URL\" \"$DISTR_SUPPORT_BUNDLE_SECRET\" > '"+dir+"/env'\n")
			checksum := sha256.Sum256(archive)
			if tampered {
				checksum = sha256.Sum256([]byte("another collector"))
			}

			bundleID := uuid.New()
			var finalized atomic.Bool
			mux := http.NewServeMux()
			mux.HandleFunc("GET /releases/"+archiveName, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(archive)
			})
			mux.HandleFunc("GET /releases/kubernetes-support-bundle-checksums.txt",
				func(w http.ResponseWriter, r *http.Request) {
					_, _ = fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(checksum[:]), archiveName)
				})
			mux.HandleFunc("POST /api/v1/support-bundle-collect/"+bundleID.String()+"/finalize",
				func(w http.ResponseWriter, r *http.Request) { finalized.Store(true) })
			server := httptest.NewServer(mux)
			defer server.Close()

			script, err := supportbundle.GenerateCollectScript(supportbundle.CollectScriptParams{
				DeploymentType:       types.DeploymentTypeKubernetes,
				BaseURL:              server.URL,
				BundleID:             bundleID,
				BundleSecret:         "secret",
				LogTailLines:         100,
				ResourceMaxBytes:     1024,
				CollectorDownloadURL: server.URL + "/releases",
			})
			g.Expect(err).ToNot(HaveOccurred())
			file := filepath.Join(dir, "collect.sh")
			g.Expect(os.WriteFile(file, []byte(innerCollectScript(g, script)), 0o600)).To(Succeed())

			cmd := exec.Command("sh", file, "--context", "prod")
			cmd.Env = append(os.Environ(), "DISTR_SUPPORT_BUNDLE_COLLECTOR=")
			out, err := cmd.CombinedOutput()

			if tampered {
				g.Expect(err).To(HaveOccurred())
				g.Expect(string(out)).To(ContainSubstring("does not match the checksum"))
				g.Expect(filepath.Join(dir, "args")).NotTo(BeAnExistingFile())
				g.Expect(finalized.Load()).To(BeFalse())
				return
			}
			g.Expect(err).NotTo(HaveOccurred(), string(out))
			args, err := os.ReadFile(filepath.Join(dir, "args"))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(strings.Fields(string(args))).To(HaveExactElements(
				"--output", HaveSuffix("/bundle"),
				"--log-tail-lines", "100", "--max-resource-bytes", "1024",
				"--context", "prod",
			))
			g.Expect(os.ReadFile(filepath.Join(dir, "env"))).To(BeEquivalentTo(
				server.URL + "/api/v1/support-bundle-collect/" + bundleID.String() + "\nsecret\n",
			))
			g.Expect(finalized.Load()).To(BeTrue())
		})
	}
}

func collectorArchive(g *WithT, collector string) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	archive := tar.NewWriter(gz)
	g.Expect(archive.WriteHeader(&tar.Header{
		Name: "kubernetes-support-bundle", Mode: 0o755, Size: int64(len(collector)), Typeflag: tar.TypeReg,
	})).To(Succeed())
	_, err := archive.Write([]byte(collector))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(archive.Close()).To(Succeed())
	g.Expect(gz.Close()).To(Succeed())
	return buf.Bytes()
}

// innerCollectScript returns the collector that the outer script writes to a temp file.
func innerCollectScript(g *WithT, script string) string {
	start := strings.Index(script, collectScriptHeredocStart)
	g.Expect(start).ToNot(Equal(-1), "collect script is not written with the expected heredoc")
	body := script[start+len(collectScriptHeredocStart):]
	end := strings.Index(body, collectScriptHeredocEnd)
	g.Expect(end).ToNot(Equal(-1), "heredoc is not terminated")
	return body[:end]
}

func expectValidShellSyntax(t *testing.T, g *WithT, script string) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh is not available")
	}

	file := filepath.Join(t.TempDir(), "collect-script.sh")
	g.Expect(os.WriteFile(file, []byte(script), 0o600)).To(Succeed())

	out, err := exec.Command(sh, "-n", file).CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "sh -n reported: %s", string(out))
}

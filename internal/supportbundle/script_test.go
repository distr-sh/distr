package supportbundle_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
// customer runs it on their host. Render both the branch with and the branch without custom scripts
// and check that each one is valid shell.
func TestGenerateCollectScript_syntax(t *testing.T) {
	for name, scripts := range map[string][]types.SupportBundleConfigurationScript{
		"without custom scripts": nil,
		"with custom scripts": {
			{Name: "disk usage", Description: new("what is on disk"), Content: "df -h\n"},
			{Name: "plain", Content: "echo hello\n"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			script, err := supportbundle.GenerateCollectScript(supportbundle.CollectScriptParams{
				BaseURL:              "https://app.example.com",
				BundleID:             uuid.New(),
				BundleSecret:         "secret",
				EnvVars:              []types.SupportBundleConfigurationEnvVar{{Name: "DB_HOST"}},
				Scripts:              scripts,
				LogTailLines:         100,
				ScriptOutputMaxBytes: 1024,
			})
			g.Expect(err).ToNot(HaveOccurred())

			expectValidShellSyntax(t, g, script)
			expectValidShellSyntax(t, g, innerCollectScript(g, script))
		})
	}
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

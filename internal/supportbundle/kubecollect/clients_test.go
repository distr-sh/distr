package kubecollect

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/clientcmd"
)

const twoContextKubeconfig = `apiVersion: v1
kind: Config
current-context: dev
clusters:
- name: dev
  cluster: {server: "https://dev.example.com"}
- name: prod
  cluster: {server: "https://prod.example.com", certificate-authority: ca.crt}
users:
- name: dev
  user: {token: dev-token}
- name: prod
  user: {token: prod-token}
contexts:
- name: dev
  context: {cluster: dev, user: dev, namespace: default}
- name: prod
  context: {cluster: prod, user: prod, namespace: shop}
`

// The written kubeconfig is all the vendor's scripts get, so it must reach the cluster and
// namespace the collector used without the customer's kubeconfig or the files next to it.
func TestWriteKubeconfig(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "ca.crt"), []byte("prod CA"), 0o600)).To(Succeed())
	kubeconfig := filepath.Join(dir, "config")
	g.Expect(os.WriteFile(kubeconfig, []byte(twoContextKubeconfig), 0o600)).To(Succeed())

	configFlags := NewConfigFlags()
	configFlags.KubeConfig = &kubeconfig
	configFlags.Context = new("prod")
	configFlags.Namespace = new("payments")
	kubeContext, err := LoadKubeContext(configFlags)
	g.Expect(err).NotTo(HaveOccurred())

	written := filepath.Join(dir, "written")
	g.Expect(WriteKubeconfig(configFlags, kubeContext, written)).To(Succeed())

	config, err := clientcmd.LoadFromFile(written)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(config.CurrentContext).To(Equal("prod"))
	g.Expect(config.Contexts).To(HaveLen(1))
	g.Expect(config.Contexts["prod"].Namespace).To(Equal("payments"))
	g.Expect(config.Clusters).To(HaveLen(1))
	g.Expect(config.Clusters["prod"].CertificateAuthority).To(BeEmpty())
	g.Expect(config.Clusters["prod"].CertificateAuthorityData).To(Equal([]byte("prod CA")))
	g.Expect(config.AuthInfos).To(HaveLen(1))
	g.Expect(config.AuthInfos["prod"].Token).To(Equal("prod-token"))
}

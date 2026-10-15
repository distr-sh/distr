// Package apitest serves the API and the OCI registry in-process for integration tests, each with a
// database of its own cloned from a migrated template and the registry's blobs kept in memory. A
// package using it calls [Main] from its TestMain and [New] from every test. Tests are skipped unless
// TEST_DATABASE_URL points at a Postgres server whose user may create roles and databases, which
// `mise run test:integration` provides.
package apitest

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/license"
	"github.com/distr-sh/distr/internal/logstore"
	"github.com/distr-sh/distr/internal/migrations"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/prometheus"
	"github.com/distr-sh/distr/internal/registry"
	"github.com/distr-sh/distr/internal/registry/blob/inmemory"
	"github.com/distr-sh/distr/internal/routing"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/svc"
	"github.com/distr-sh/distr/internal/tracers"
	"github.com/distr-sh/distr/internal/util"
	"github.com/go-mailx/mailx"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const databaseURLEnv = "TEST_DATABASE_URL"

// testEnv is the configuration every test runs with. The database URL is a placeholder because
// [New] connects each test to a database of its own.
var testEnv = map[string]string{
	"DATABASE_URL":                     "postgres://unused",
	"DATABASE_ENCRYPTION_KEY":          "iQyACEaOrUAE0x6nffYgT3eHddyHMG1cClBNYhbDyD0=",
	"JWT_SECRET":                       "H4V7z6mEpe8/k5H/KyogT/iATJhNpEIUMd5cHF6mqF8=",
	"DISTR_HOST":                       "http://localhost:8080",
	"LOKI_URL":                         "http://localhost:3100",
	"USER_EMAIL_VERIFICATION_REQUIRED": "false",
}

var migrator = golangmigrator.New("sql", golangmigrator.WithFS(migrations.FS))

// Main configures the process and runs the tests of a package. env and dbcrypto are read once per
// process, so every test of the package shares this configuration.
func Main(m *testing.M) {
	// mise points DISTR_ENV at the local development settings, which env.Initialize would load.
	util.Must(os.Unsetenv("DISTR_ENV"))
	for key, value := range testEnv {
		util.Must(os.Setenv(key, value))
	}
	util.Must(os.Setenv("LICENSE_KEY_PRIVATE_KEY", generateLicenseKeyPrivateKey()))
	env.Initialize()
	util.Must(dbcrypto.Init(env.DatabaseEncryptionKey()))
	util.Must(license.Initialize())
	// Distr runs in UTC in its images, and pgx writes the wall clock of a non-UTC time into a
	// TIMESTAMP column, so the tests would otherwise depend on the zone of the machine.
	time.Local = time.UTC
	os.Exit(m.Run())
}

func generateLicenseKeyPrivateKey() string {
	_, key, err := ed25519.GenerateKey(nil)
	util.Must(err)
	der := util.Require(x509.MarshalPKCS8PrivateKey(key))
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// Harness is the API of a single test, served from a database nothing else uses.
type Harness struct {
	t        testing.TB
	server   *httptest.Server
	registry *httptest.Server
	mails    *mailRecorder
	// DB is the test's database, for checking what a request wrote.
	DB *pgxpool.Pool
}

func New(t testing.TB) *Harness {
	t.Helper()
	baseURL := os.Getenv(databaseURLEnv)
	if baseURL == "" {
		t.Skipf("%v is not set", databaseURLEnv)
	}

	dbConfig := pgtestdb.Custom(t, pgtestdbConfig(t, baseURL), migrator)
	pool := newPool(t, dbConfig.URL())

	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	prepareDatabase(t, internalctx.WithLogger(internalctx.WithDb(t.Context(), pool), logger))
	oidcer, err := oidc.NewOIDCer(t.Context(), logger)
	if err != nil {
		t.Fatalf("failed to create OIDCer: %v", err)
	}
	noopTracer := noop.NewTracerProvider()
	mails := &mailRecorder{}
	mailer := &mailx.Mailer{
		MailerAdapter: mails,
		Config: &mailx.MailerConfig{
			FromAddressSrc: []mailx.FromAddressFunc{
				mailx.MailOverrideFromAddress(),
				mailx.StaticFromAddress("Distr <noreply@distr.test>"),
			},
		},
	}

	server := httptest.NewServer(routing.NewRouter(
		logger,
		pool,
		nil,
		mailer,
		&tracers.Tracers{DefaultProvider: noopTracer, AlwaysProvider: noopTracer},
		oidcer,
		prometheus.NewDistrCollector(),
		logstore.NewFake(),
	))
	t.Cleanup(server.Close)
	registryServer := httptest.NewServer(
		registry.NewWithBlobHandler(logger, pool, mailer, noopTracer, inmemory.NewBlobHandler()),
	)
	t.Cleanup(registryServer.Close)

	return &Harness{t: t, server: server, registry: registryServer, mails: mails, DB: pool}
}

// QueryValue runs sql, which has to return exactly one row with one column, against the database
// of h.
func QueryValue[T any](h *Harness, sql string, args ...any) T {
	h.t.Helper()
	var value T
	if err := h.DB.QueryRow(h.t.Context(), sql, args...).Scan(&value); err != nil {
		h.t.Fatalf("query %q failed: %v", sql, err)
	}
	return value
}

// Exec runs sql against the database of h, for putting a fixture into a state the API does not
// offer, such as an expired secret.
func (h *Harness) Exec(sql string, args ...any) {
	h.t.Helper()
	if _, err := h.DB.Exec(h.t.Context(), sql, args...); err != nil {
		h.t.Fatalf("exec %q failed: %v", sql, err)
	}
}

// prepareDatabase does what the serve command does on startup to a fresh database.
func prepareDatabase(t testing.TB, ctx context.Context) {
	t.Helper()
	if err := db.CreateControllerVersion(ctx); err != nil {
		t.Fatalf("failed to create controller version: %v", err)
	}
	if err := subscription.ReconcileEditionFeatures(ctx); err != nil {
		t.Fatalf("failed to reconcile edition features: %v", err)
	}
}

func pgtestdbConfig(t testing.TB, rawURL string) pgtestdb.Config {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("%v is not a valid URL: %v", databaseURLEnv, err)
	}
	password, _ := u.User.Password()
	return pgtestdb.Config{
		DriverName:                "pgx",
		Host:                      u.Hostname(),
		Port:                      u.Port(),
		User:                      u.User.Username(),
		Password:                  password,
		Database:                  u.Path[1:],
		Options:                   u.RawQuery,
		ForceTerminateConnections: true,
	}
}

func newPool(t testing.TB, url string) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("failed to parse database URL: %v", err)
	}
	config.AfterConnect = svc.RegisterDBTypes
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatalf("failed to create database pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

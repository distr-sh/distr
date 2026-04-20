package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/distr-sh/distr/internal/svc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type GenerateLicenseKeyOptions struct {
	OrgID         string
	CustomerOrgID string
	Name          string
	Description   string
	Payload       string
	NotBefore     string
	ExpiresAt     string
	ValidPeriod   string
	NoSave        bool
}

func NewGenerateLicenseKeyCommand() *cobra.Command {
	var opts GenerateLicenseKeyOptions
	cmd := &cobra.Command{
		Use:    "generate",
		PreRun: func(cmd *cobra.Command, args []string) { env.Initialize() },
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateLicenseKey(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.OrgID, "organization-id", "o", "", "Organization ID (required without --no-save)")
	cmd.Flags().StringVarP(&opts.CustomerOrgID, "customer-id", "c", "",
		"Customer organization ID (required without --no-save)")
	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "License key name (required without --no-save)")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "License key description")
	cmd.Flags().StringVarP(&opts.Payload, "payload", "p", "{}", "License key JSON payload")
	cmd.Flags().StringVar(&opts.NotBefore, "not-before", "",
		"Date after which the license key is valid (yyyy-mm-dd; default \"time.Now()\")")
	cmd.Flags().StringVar(&opts.ExpiresAt, "expires-at", "",
		"Date until the license key is valid (yyyy-mm-dd)")
	cmd.Flags().StringVar(&opts.ValidPeriod, "valid-period", "8760h", "Validity period")
	cmd.Flags().BoolVar(&opts.NoSave, "no-save", false, "Skip saving the license key to the database")
	cmd.MarkFlagsMutuallyExclusive("expires-at", "valid-period")
	cmd.MarkFlagsRequiredTogether("organization-id", "customer-id", "name")
	for _, f := range []string{"organization-id", "customer-id", "name"} {
		cmd.MarkFlagsOneRequired("no-save", f)
		cmd.MarkFlagsMutuallyExclusive("no-save", f)
	}

	return cmd
}

func runGenerateLicenseKey(ctx context.Context, opts GenerateLicenseKeyOptions) error {
	var notBefore time.Time
	if opts.NotBefore != "" {
		if t, err := time.Parse(time.DateOnly, opts.NotBefore); err != nil {
			return fmt.Errorf("invalid not-before: %w", err)
		} else {
			notBefore = t
		}
	} else {
		notBefore = time.Now()
	}

	var expiresAt time.Time
	if opts.ExpiresAt != "" {
		if t, err := time.Parse(time.DateOnly, opts.ExpiresAt); err != nil {
			return fmt.Errorf("invalid expires-at: %w", err)
		} else {
			expiresAt = t
		}
	} else {
		if d, err := time.ParseDuration(opts.ValidPeriod); err != nil {
			return fmt.Errorf("invalid valid-period: %w", err)
		} else {
			expiresAt = notBefore.Add(d)
		}
	}

	if opts.NoSave {
		data := licensekey.LicenseKeyData{
			LicenseKeyID: uuid.New(),
			IssuedAt:     time.Now(),
			NotBefore:    notBefore,
			ExpiresAt:    expiresAt,
			Payload:      json.RawMessage(opts.Payload),
		}
		token, err := licensekey.GenerateToken(data, env.Host())
		if err != nil {
			return fmt.Errorf("token creation error: %w", err)
		}
		fmt.Println(token)
		return nil
	}

	registry := util.Require(svc.NewDefault(ctx))
	defer func() { util.Must(registry.Shutdown(ctx)) }()
	log := registry.GetLogger()

	license := types.LicenseKey{
		Name:      opts.Name,
		Payload:   json.RawMessage(opts.Payload),
		NotBefore: &notBefore,
		ExpiresAt: &expiresAt,
	}

	if opts.Description != "" {
		license.Description = &opts.Description
	}

	if p, err := uuid.Parse(opts.OrgID); err != nil {
		log.Error("invalid organization-id", zap.Error(err))
		return err
	} else {
		license.OrganizationID = p
	}

	if p, err := uuid.Parse(opts.CustomerOrgID); err != nil {
		log.Error("invalid customer-id", zap.Error(err))
		return err
	} else {
		license.CustomerOrganizationID = &p
	}

	log.Debug("creating license", zap.Any("license", license))

	if err := db.CreateLicenseKey(internalctx.WithDb(ctx, registry.GetDbPool()), &license); err != nil {
		log.Error("license creation error", zap.Error(err))
		return err
	}

	token, err := licensekey.GenerateToken(licensekey.FromLicenseKey(license), env.Host())
	if err != nil {
		log.Error("token creation error", zap.Error(err))
		return err
	}

	log.Info("license created", zap.String("token", token))

	return nil
}

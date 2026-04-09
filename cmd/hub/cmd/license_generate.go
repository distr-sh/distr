package cmd

import (
	"context"
	"encoding/json"
	"os"
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
}

func NewGenerateLicenseKeyCommand() *cobra.Command {
	var opts GenerateLicenseKeyOptions
	cmd := &cobra.Command{
		Use:    "generate",
		PreRun: func(cmd *cobra.Command, args []string) { env.Initialize() },
		Run: func(cmd *cobra.Command, args []string) {
			if err := runGenerateLicenseKey(cmd.Context(), opts); err != nil {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&opts.OrgID, "organization-id", "o", "", "Organization ID (required)")
	util.Must(cmd.MarkFlagRequired("organization-id"))
	cmd.Flags().StringVarP(&opts.CustomerOrgID, "customer-id", "c", "", "Customer organization ID (required)")
	util.Must(cmd.MarkFlagRequired("customer-id"))
	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "License key name (required)")
	util.Must(cmd.MarkFlagRequired("name"))
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "License key description")
	cmd.Flags().StringVarP(&opts.Payload, "payload", "p", "{}", "License key JSON payload")
	cmd.Flags().StringVar(&opts.NotBefore, "not-before", "",
		"Date after which the license key is valid (yyyy-mm-dd; default \"time.Now()\")")
	cmd.Flags().StringVar(&opts.ExpiresAt, "expires-at", "",
		"Date until the license key is valid (yyyy-mm-dd)")
	cmd.Flags().StringVar(&opts.ValidPeriod, "valid-period", "8760h", "Validity period")
	cmd.MarkFlagsMutuallyExclusive("expires-at", "valid-period")

	return cmd
}

func runGenerateLicenseKey(ctx context.Context, opts GenerateLicenseKeyOptions) error {
	registry := util.Require(svc.NewDefault(ctx))
	defer func() { util.Must(registry.Shutdown(ctx)) }()
	log := registry.GetLogger()

	license := types.LicenseKey{Name: opts.Name, Payload: json.RawMessage(opts.Payload)}

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

	if opts.NotBefore != "" {
		if t, err := time.Parse(time.DateOnly, opts.NotBefore); err != nil {
			log.Error("invalid not-before", zap.Error(err))
			return err
		} else {
			license.NotBefore = &t
		}
	} else {
		t := time.Now()
		license.NotBefore = &t
	}

	if opts.ExpiresAt != "" {
		if t, err := time.Parse(time.DateOnly, opts.ExpiresAt); err != nil {
			log.Error("invalid expires-at", zap.Error(err))
			return err
		} else {
			license.ExpiresAt = &t
		}
	} else {
		if d, err := time.ParseDuration(opts.ValidPeriod); err != nil {
			log.Error("invalid valid-period", zap.Error(err))
			return err
		} else {
			t := license.NotBefore.Add(d)
			license.ExpiresAt = &t
		}
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

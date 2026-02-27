package context

import (
	"context"

	"github.com/distr-sh/distr/internal/types"
)

func GetApplication(ctx context.Context) *types.Application {
	val := ctx.Value(ctxKeyApplication)
	if application, ok := val.(*types.Application); ok {
		if application != nil {
			return application
		}
	}
	panic("application not contained in context")
}

func WithApplication(ctx context.Context, application *types.Application) context.Context {
	ctx = context.WithValue(ctx, ctxKeyApplication, application)
	return ctx
}

func GetDeployment(ctx context.Context) *types.Deployment {
	val := ctx.Value(ctxKeyDeployment)
	if deployment, ok := val.(*types.Deployment); ok {
		if deployment != nil {
			return deployment
		}
	}
	panic("deployment not contained in context")
}

func WithDeployment(ctx context.Context, deployment *types.Deployment) context.Context {
	ctx = context.WithValue(ctx, ctxKeyDeployment, deployment)
	return ctx
}

func WithDeploymentTarget(ctx context.Context, dt *types.DeploymentTargetFull) context.Context {
	return context.WithValue(ctx, ctxKeyDeploymentTarget, dt)
}

func GetDeploymentTarget(ctx context.Context) *types.DeploymentTargetFull {
	val := ctx.Value(ctxKeyDeploymentTarget)
	if dt, ok := val.(*types.DeploymentTargetFull); ok {
		if dt != nil {
			return dt
		}
	}
	panic("deployment target not contained in context")
}

func WithFile(ctx context.Context, file *types.File) context.Context {
	return context.WithValue(ctx, ctxKeyFile, file)
}

func GetFile(ctx context.Context) *types.File {
	if file, ok := ctx.Value(ctxKeyFile).(*types.File); ok {
		return file
	}
	panic("no File found in context")
}

func WithUserAccount(ctx context.Context, userAccount *types.UserAccountWithUserRole) context.Context {
	return context.WithValue(ctx, ctxKeyUserAccount, userAccount)
}

func GetUserAccount(ctx context.Context) *types.UserAccountWithUserRole {
	if userAccount, ok := ctx.Value(ctxKeyUserAccount).(*types.UserAccountWithUserRole); ok {
		return userAccount
	}
	panic("no UserAccount found in context")
}

func WithArtifact(ctx context.Context, userAccount *types.ArtifactWithTaggedVersion) context.Context {
	return context.WithValue(ctx, ctxKeyArtifact, userAccount)
}

func GetArtifact(ctx context.Context) *types.ArtifactWithTaggedVersion {
	if userAccount, ok := ctx.Value(ctxKeyArtifact).(*types.ArtifactWithTaggedVersion); ok {
		return userAccount
	}
	panic("no Artifact found in context")
}

func GetApplicationLicense(ctx context.Context) *types.ApplicationLicense {
	val := ctx.Value(ctxKeyApplicationLicense)
	if license, ok := val.(*types.ApplicationLicense); ok {
		if license != nil {
			return license
		}
	}
	panic("license not contained in context")
}

func WithApplicationLicense(ctx context.Context, license *types.ApplicationLicense) context.Context {
	ctx = context.WithValue(ctx, ctxKeyApplicationLicense, license)
	return ctx
}

func GetArtifactLicense(ctx context.Context) *types.ArtifactLicense {
	val := ctx.Value(ctxKeyArtifactLicense)
	if license, ok := val.(*types.ArtifactLicense); ok {
		if license != nil {
			return license
		}
	}
	panic("license not contained in context")
}

func WithArtifactLicense(ctx context.Context, license *types.ArtifactLicense) context.Context {
	ctx = context.WithValue(ctx, ctxKeyArtifactLicense, license)
	return ctx
}

func GetUsageLicense(ctx context.Context) *types.UsageLicense {
	val := ctx.Value(ctxKeyUsageLicense)
	if license, ok := val.(*types.UsageLicense); ok {
		if license != nil {
			return license
		}
	}
	panic("usage license not contained in context")
}

func WithUsageLicense(ctx context.Context, license *types.UsageLicense) context.Context {
	return context.WithValue(ctx, ctxKeyUsageLicense, license)
}

func GetRequestIPAddress(ctx context.Context) string {
	if val, ok := ctx.Value(ctxKeyIPAddress).(string); ok {
		return val
	}
	panic("no IP address in context")
}

func WithRequestIPAddress(ctx context.Context, address string) context.Context {
	return context.WithValue(ctx, ctxKeyIPAddress, address)
}

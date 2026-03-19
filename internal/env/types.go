package env

import (
	"fmt"
	"net/mail"
)

type RegistrationMode string

const (
	RegistrationEnabled  RegistrationMode = "enabled"
	RegistrationHidden   RegistrationMode = "hidden"
	RegistrationDisabled RegistrationMode = "disabled"
)

func parseRegistrationMode(value string) (RegistrationMode, error) {
	switch value {
	case string(RegistrationEnabled), string(RegistrationHidden), string(RegistrationDisabled):
		return RegistrationMode(value), nil
	default:
		return "", fmt.Errorf("invalid RegistrationMode: %v", value)
	}
}

type MailerTypeString string

const (
	MailerTypeSMTP        MailerTypeString = "smtp"
	MailerTypeSES         MailerTypeString = "ses"
	MailerTypeUnspecified MailerTypeString = ""
)

func parseMailerType(value string) (MailerTypeString, error) {
	switch value {
	case string(MailerTypeSES), string(MailerTypeSMTP), string(MailerTypeUnspecified):
		return MailerTypeString(value), nil
	default:
		return "", fmt.Errorf("invalid MailerTypeString: %v", value)
	}
}

type MailerConfig struct {
	Type        MailerTypeString
	FromAddress mail.Address
	SmtpConfig  *MailerSMTPConfig
}

type MailerSMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	ImplicitTLS bool
}

type S3Config struct {
	Bucket                                 string
	Region                                 string
	Endpoint                               *string
	AccessKeyID                            *string
	SecretAccessKey                        *string
	UsePathStyle                           bool
	AllowRedirect                          bool
	CreateBucket                           bool
	RequestChecksumCalculationWhenRequired bool
	ResponseChecksumValidationWhenRequired bool
	ResignForGCP                           bool
}

type SamplerType string

const (
	SamplerAlwaysOn                SamplerType = "always_on"
	SamplerAlwaysOff               SamplerType = "always_off"
	SamplerTraceIDRatio            SamplerType = "traceidratio"
	SamplerParentBasedAlwaysOn     SamplerType = "parentbased_always_on"
	SamplerParsedBasedAlwaysOff    SamplerType = "parentbased_always_off"
	SamplerParentBasedTraceIDRatio SamplerType = "parentbased_traceidratio"
)

func parseSamplerType(value string) (SamplerType, error) {
	switch value {
	case string(SamplerAlwaysOn),
		string(SamplerAlwaysOff),
		string(SamplerTraceIDRatio),
		string(SamplerParentBasedAlwaysOn),
		string(SamplerParsedBasedAlwaysOff),
		string(SamplerParentBasedTraceIDRatio):

		return SamplerType(value), nil
	default:
		return "", fmt.Errorf("invalid SamplerType: %v", value)
	}
}

type SamplerConfig struct {
	Sampler SamplerType
	Arg     float64
}

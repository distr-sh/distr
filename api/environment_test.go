package api

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestCreateUpdateEnvironmentRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request CreateUpdateEnvironmentRequest
		wantErr bool
	}{
		{
			name: "accepts complete environment settings",
			request: CreateUpdateEnvironmentRequest{
				Name:                "Production",
				Description:         "Customer production targets",
				SortOrder:           30,
				IsProduction:        true,
				AllowDynamicTargets: false,
				RetentionPolicyID:   new(uuid.UUID),
			},
			wantErr: false,
		},
		{
			name:    "rejects blank names",
			request: CreateUpdateEnvironmentRequest{Name: "   "},
			wantErr: true,
		},
		{
			name:    "rejects negative sort order",
			request: CreateUpdateEnvironmentRequest{Name: "Development", SortOrder: -1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			err := tt.request.Validate()

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}

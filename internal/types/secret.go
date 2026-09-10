package types

import (
	"time"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

type Secret struct {
	ID                     uuid.UUID       `db:"id"`
	CreatedAt              time.Time       `db:"created_at"`
	UpdatedAt              time.Time       `db:"updated_at"`
	UpdatedByUserAccountID *uuid.UUID      `db:"updated_by_useraccount_id"`
	OrganizationID         uuid.UUID       `db:"organization_id"`
	CustomerOrganizationID *uuid.UUID      `db:"customer_organization_id"`
	Key                    string          `db:"key"`
	Value                  dbcrypto.String `db:"value"`
}

type SecretWithUpdatedBy struct {
	Secret
	UpdatedBy *UserAccount `db:"updated_by"`
}

package types

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	CreatedAt      time.Time  `db:"created_at" json:"createdAt"`
	OrganizationID *uuid.UUID `db:"organization_id" json:"-"`
	ContentType    string     `db:"content_type" json:"contentType"`
	Data           []byte     `db:"data" json:"data"`
	FileName       string     `db:"file_name" json:"fileName"`
	FileSize       int64      `db:"file_size" json:"fileSize"`
	Public         bool       `db:"public" json:"public"`
}

// FileMetadata holds the ownership and visibility of a file without loading its data blob.
type FileMetadata struct {
	OrganizationID *uuid.UUID `db:"organization_id"`
	ContentType    string     `db:"content_type"`
	Public         bool       `db:"public"`
}

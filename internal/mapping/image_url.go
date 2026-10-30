package mapping

import (
	"github.com/google/uuid"
)

func CreateImageURL(imageID *uuid.UUID) *string {
	if imageID == nil || *imageID == uuid.Nil {
		return nil
	}
	return new("/api/v1/files/" + imageID.String())
}

func CreatePublicImageURL(imageID *uuid.UUID) *string {
	if imageID == nil || *imageID == uuid.Nil {
		return nil
	}
	return new("/api/public/v1/files/" + imageID.String())
}

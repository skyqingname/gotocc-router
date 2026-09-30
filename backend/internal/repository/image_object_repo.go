package repository

import (
	"context"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/ent/imageobject"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type imageObjectRepository struct {
	client *dbent.Client
}

func NewImageObjectRepository(client *dbent.Client) service.ImageObjectRepository {
	return &imageObjectRepository{client: client}
}

func (r *imageObjectRepository) CreateMany(ctx context.Context, objects []service.ImageObjectRecord) error {
	if len(objects) == 0 {
		return nil
	}
	builders := make([]*dbent.ImageObjectCreate, 0, len(objects))
	for _, object := range objects {
		builder := clientFromContext(ctx, r.client).ImageObject.Create().
			SetObjectID(object.ObjectID).
			SetUserID(object.UserID).
			SetAPIKeyID(object.APIKeyID).
			SetTaskID(object.TaskID).
			SetStorageKey(object.StorageKey).
			SetContentType(object.ContentType).
			SetByteSize(object.Bytes)
		if !object.CreatedAt.IsZero() {
			builder.SetCreatedAt(object.CreatedAt)
		}
		builders = append(builders, builder)
	}
	return clientFromContext(ctx, r.client).ImageObject.CreateBulk(builders...).Exec(ctx)
}

func (r *imageObjectRepository) GetOwned(ctx context.Context, objectID string, userID int64) (*service.ImageObjectRecord, error) {
	object, err := clientFromContext(ctx, r.client).ImageObject.Query().
		Where(imageobject.ObjectIDEQ(objectID), imageobject.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrImageObjectNotFound
		}
		return nil, err
	}
	return &service.ImageObjectRecord{
		ObjectID: object.ObjectID, UserID: object.UserID, APIKeyID: object.APIKeyID,
		TaskID: object.TaskID, StorageKey: object.StorageKey,
		ContentType: object.ContentType, Bytes: object.ByteSize, CreatedAt: object.CreatedAt,
	}, nil
}

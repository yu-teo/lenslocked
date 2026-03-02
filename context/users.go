package context

import (
	"context"

	"github.com/whyttea/lenslocked/models"
)

type key string

const (
	userKey key = "user"
)

// store a user in the context
func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func User(ctx context.Context) *models.User {
	val := ctx.Value(userKey)
	// perform type assertion to confirm if e received the user type back
	user, ok := val.(*models.User)
	if !ok {
		// this case could be hit if there was nothing stored in the context so we returned a nil, which does not have the type *models.User;
		// it's also possible other code in this package wrote an invalid value using the user key
		return nil
	}
	return user
}

package schema

import (
	"context"
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"fmt"
	entgen "github.com/xiajignge/aihub/internal/ent"
	entapikey "github.com/xiajignge/aihub/internal/ent/apikey"
	entchannel "github.com/xiajignge/aihub/internal/ent/channel"
	entrequest "github.com/xiajignge/aihub/internal/ent/request"
	entrole "github.com/xiajignge/aihub/internal/ent/role"
	entuser "github.com/xiajignge/aihub/internal/ent/user"
	"time"
)

// SoftDeleteMixin implements the soft delete pattern for schemas.
type SoftDeleteMixin struct {
	mixin.Schema
}

// Fields of the SoftDeleteMixin.
// For some databases, the null is distinct, so every row with null will be different.
// So the nullable deleted_at solution is not a good solution.
// For non deleted rows, the deleted_at will be 0.
// For deleted rows, the deleted_at will be a timestamp.
func (SoftDeleteMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Int("deleted_at").Default(0).Annotations(
			entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
		),
	}
}

type softDeleteKey struct{}

// SkipSoftDelete returns a new context that skips the soft-delete interceptor/mutators.
func SkipSoftDelete(parent context.Context) context.Context {
	return context.WithValue(parent, softDeleteKey{}, true)
}

// Interceptors of the SoftDeleteMixin.
func (d SoftDeleteMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		ent.TraverseFunc(func(ctx context.Context, q ent.Query) error {
			// Skip soft-delete, means include soft-deleted entities.
			if skip, _ := ctx.Value(softDeleteKey{}).(bool); skip {
				return nil
			}

			switch query := q.(type) {
			case *entgen.APIKeyQuery:
				query.Where(entapikey.DeletedAtEQ(0))
			case *entgen.ChannelQuery:
				query.Where(entchannel.DeletedAtEQ(0))
			case *entgen.RequestQuery:
				query.Where(entrequest.DeletedAtEQ(0))
			case *entgen.RoleQuery:
				query.Where(entrole.DeletedAtEQ(0))
			case *entgen.UserQuery:
				query.Where(entuser.DeletedAtEQ(0))
			default:
				return fmt.Errorf("unexpected query type %T", q)
			}

			return nil
		}),
	}
}

// Hooks of the SoftDeleteMixin.
func (d SoftDeleteMixin) Hooks() []ent.Hook {
	return []ent.Hook{
		func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				if !m.Op().Is(ent.OpDelete | ent.OpDeleteOne) {
					return next.Mutate(ctx, m)
				}
				// Skip soft-delete, means delete the entity permanently.
				if skip, _ := ctx.Value(softDeleteKey{}).(bool); skip {
					return next.Mutate(ctx, m)
				}

				mx, ok := m.(interface {
					SetOp(ent.Op)
					SetDeletedAt(int)
					WhereP(...func(*sql.Selector))
					Client() interface {
						Mutate(context.Context, ent.Mutation) (ent.Value, error)
					}
				})
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}

				d.P(mx)
				mx.SetOp(ent.OpUpdate)
				mx.SetDeletedAt(int(time.Now().Unix()))

				return mx.Client().Mutate(ctx, m)
			})
		},
	}
}

// P 给当前查询或 mutation 加一个底层 SQL 条件:deleted_at = 0
func (d SoftDeleteMixin) P(w interface{ WhereP(...func(*sql.Selector)) }) {
	w.WhereP(
		sql.FieldEQ("deleted_at", 0),
		// 旧实现（依赖 Fields() 的字段顺序）：
		// sql.FieldEQ(d.Fields()[0].Descriptor().Name, 0),
	)
}

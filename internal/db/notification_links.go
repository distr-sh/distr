package db

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type notificationLink struct {
	table        string
	configColumn string
	valueColumn  string
	valueTable   string
	// valueCondition further restricts which rows may be linked, on top of belonging to the
	// organization. It is applied to the alias `v` of valueTable.
	valueCondition string
}

// replaceNotificationLinks makes the rows of a link table match the given ids exactly. Ids are
// filtered through the table they point at, so that a configuration cannot be linked to a row of
// another organization by passing its id.
func replaceNotificationLinks(
	ctx context.Context,
	link notificationLink,
	configID, organizationID uuid.UUID,
	valueIDs []uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	condition := ""
	if link.valueCondition != "" {
		condition = " AND " + link.valueCondition
	}
	if _, err := db.Exec(
		ctx,
		fmt.Sprintf(`INSERT INTO %v (%v, %v)
		SELECT @configID, v.id FROM %v v
		WHERE v.id = any(@valueIDs) AND v.organization_id = @organizationID%v
		ON CONFLICT DO NOTHING`,
			link.table, link.configColumn, link.valueColumn, link.valueTable, condition),
		pgx.NamedArgs{
			"configID":       configID,
			"organizationID": organizationID,
			"valueIDs":       valueIDs,
		},
	); err != nil {
		return fmt.Errorf("failed to insert into %v: %w", link.table, err)
	}

	if _, err := db.Exec(
		ctx,
		fmt.Sprintf(`DELETE FROM %v WHERE %v = @configID AND NOT %v = any(@valueIDs)`,
			link.table, link.configColumn, link.valueColumn),
		pgx.NamedArgs{"configID": configID, "valueIDs": valueIDs},
	); err != nil {
		return fmt.Errorf("failed to delete from %v: %w", link.table, err)
	}

	return nil
}

// replaceNotificationRecipients makes the recipients of a configuration match the given user
// accounts exactly. Only members of the configuration's organization can be linked.
func replaceNotificationRecipients(
	ctx context.Context,
	table, configColumn string,
	configID, organizationID uuid.UUID,
	userAccountIDs []uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(
		ctx,
		fmt.Sprintf(`INSERT INTO %v (%v, organization_id, user_account_id)
		SELECT @configID, oua.organization_id, oua.user_account_id
		FROM Organization_UserAccount oua
		WHERE oua.user_account_id = any(@userAccountIDs) AND oua.organization_id = @organizationID
		ON CONFLICT DO NOTHING`, table, configColumn),
		pgx.NamedArgs{
			"configID":       configID,
			"organizationID": organizationID,
			"userAccountIDs": userAccountIDs,
		},
	); err != nil {
		return fmt.Errorf("failed to insert into %v: %w", table, err)
	}

	if _, err := db.Exec(
		ctx,
		fmt.Sprintf(`DELETE FROM %v WHERE %v = @configID AND NOT user_account_id = any(@userAccountIDs)`,
			table, configColumn),
		pgx.NamedArgs{"configID": configID, "userAccountIDs": userAccountIDs},
	); err != nil {
		return fmt.Errorf("failed to delete from %v: %w", table, err)
	}

	return nil
}

package sqlx

import (
	"context"

	"code.cloudfoundry.org/lager"
	"github.com/Masterminds/squirrel"
)

func RollbackMigrations(
	ctx context.Context,
	logger lager.Logger,
	conn *DB,
	tableName string,
	migrations []Migration,
	all bool,
) error {
	migrationsLogger := logger.Session("rollback-migrations").WithData(lager.Data{
		"table_name": tableName,
	})

	migrationsLogger.Info("starting")
	if len(migrations) == 0 {
		return nil
	}

	appliedMigrations, err := RetrieveAppliedMigrations(ctx, migrationsLogger, conn, tableName)
	if err != nil {
		return err
	}
	migrationsLogger.Debug(retrievedAppliedMigrations, lager.Data{
		"versions": appliedMigrations,
	})

	for version := len(migrations) - 1; version >= 0; version-- {
		migration := migrations[version]
		_, ok := appliedMigrations[version]

		migrationLogger := logger.WithData(lager.Data{
			"version": version,
			"name":    migration.Name,
		})

		if !ok {
			migrationLogger.Debug("skipping")
			continue
		}

		err = rollbackMigration(ctx, migrationLogger, conn, tableName, version, migration)
		if err != nil {
			return err
		}
		if !all {
			return nil
		}
	}

	return nil
}

func rollbackMigration(
	ctx context.Context,
	logger lager.Logger,
	conn *DB,
	tableName string,
	version int,
	migration Migration,
) (err error) {
	logger.Debug(starting)

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		logger.Error(failedToStartTransaction, err)
		return
	}

	defer func() {
		if err != nil {
			logger.Error(failedToApplyMigration, err)
		}
		err = Commit(logger, tx, err)
	}()

	err = migration.Down(ctx, logger, tx)
	if err != nil {
		return
	}

	_, err = squirrel.Delete(tableName).
		Where(squirrel.Eq{"version": version}).
		RunWith(tx).
		ExecContext(ctx)

	logger.Debug(finished)

	return
}

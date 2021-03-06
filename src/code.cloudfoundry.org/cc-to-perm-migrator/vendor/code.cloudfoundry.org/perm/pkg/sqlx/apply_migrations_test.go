package sqlx_test

import (
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "code.cloudfoundry.org/perm/pkg/sqlx"

	"database/sql"

	"context"

	"errors"

	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("#ApplyMigrations", func() {
	var (
		migrationTableName string

		fakeLogger *lagertest.TestLogger

		fakeConn *sql.DB
		mock     sqlmock.Sqlmock
		err      error

		conn *DB

		ctx context.Context
	)

	BeforeEach(func() {
		migrationTableName = "db_migrations"

		fakeLogger = lagertest.NewTestLogger("perm-sqlx")

		fakeConn, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		conn = &DB{
			Conn: fakeConn,
		}
		ctx = context.Background()
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	It("creates the table if not exists", func() {
		mock.ExpectBegin()
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS `" + migrationTableName +
			"` \\(id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, version INTEGER, name VARCHAR\\(255\\), applied_at DATETIME\\)").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err = ApplyMigrations(ctx, fakeLogger, conn, migrationTableName, []Migration{})

		Expect(err).NotTo(HaveOccurred())
	})

	It("returns the error if the commit fails", func() {
		mock.ExpectBegin()

		mock.ExpectExec("CREATE TABLE").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit().WillReturnError(errors.New("commit-failed"))

		err = ApplyMigrations(ctx, fakeLogger, conn, migrationTableName, []Migration{})

		Expect(err).To(MatchError("commit-failed"))
	})

	It("rolls back if the table creation fails", func() {
		mock.ExpectBegin()

		mock.ExpectExec("CREATE TABLE").
			WillReturnError(errors.New("create-table-failed"))
		mock.ExpectRollback()

		err = ApplyMigrations(ctx, fakeLogger, conn, migrationTableName, []Migration{})

		Expect(err).To(MatchError("create-table-failed"))
	})

	It("returns the create table failure if the rollback fails", func() {
		mock.ExpectBegin()

		mock.ExpectExec("CREATE TABLE").
			WillReturnError(errors.New("create-table-failed"))
		mock.ExpectRollback().WillReturnError(errors.New("rollback-failed"))

		err = ApplyMigrations(ctx, fakeLogger, conn, migrationTableName, []Migration{})

		Expect(err).To(MatchError("create-table-failed"))
	})

	It("applies the migrations", func() {
		migration1 := Migration{
			Name: "migration-1",
			Up: func(ctx context.Context, logger lager.Logger, tx *Tx) error {
				_, err := tx.ExecContext(ctx, "FAKE MIGRATION")
				return err
			},
		}
		migration2 := Migration{
			Name: "migration-2",
			Up: func(ctx context.Context, logger lager.Logger, tx *Tx) error {
				_, err := tx.ExecContext(ctx, "THIS IS A TEST")
				return err
			},
		}
		migrations := []Migration{migration1, migration2}

		mock.ExpectBegin()

		mock.ExpectExec("CREATE TABLE").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		mock.ExpectQuery("SELECT version, name, applied_at FROM " + migrationTableName).
			WillReturnRows(sqlmock.NewRows([]string{"version", "name", "applied_at"}))

		mock.ExpectBegin()
		mock.ExpectExec("FAKE MIGRATION").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO "+migrationTableName+" \\(version,name,applied_at\\)").
			WithArgs(0, "migration-1", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		mock.ExpectBegin()
		mock.ExpectExec("THIS IS A TEST").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO "+migrationTableName+" \\(version,name,applied_at\\)").
			WithArgs(1, "migration-2", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(2, 1))
		mock.ExpectCommit()

		err = ApplyMigrations(ctx, fakeLogger, conn, migrationTableName, migrations)

		Expect(err).NotTo(HaveOccurred())
	})

	It("does not repeat applied migrations", func() {
		migration1 := Migration{
			Name: "migration-1",
			Up: func(ctx context.Context, logger lager.Logger, tx *Tx) error {
				_, err := tx.ExecContext(ctx, "FAKE MIGRATION")
				return err
			},
		}
		migration2 := Migration{
			Name: "migration-2",
			Up: func(ctx context.Context, logger lager.Logger, tx *Tx) error {
				_, err := tx.ExecContext(ctx, "THIS IS A TEST")
				return err
			},
		}

		mock.ExpectBegin()

		mock.ExpectExec("CREATE TABLE").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		mock.ExpectQuery("SELECT version, name, applied_at FROM " + migrationTableName).
			WillReturnRows(sqlmock.NewRows([]string{"version", "name", "applied_at"}).
				AddRow("0", "migration-1", time.Now()))

		mock.ExpectBegin()
		mock.ExpectExec("THIS IS A TEST").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO "+migrationTableName+" \\(version,name,applied_at\\)").
			WithArgs(1, "migration-2", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(2, 1))
		mock.ExpectCommit()

		err = ApplyMigrations(ctx, fakeLogger, conn, migrationTableName, []Migration{migration1, migration2})

		Expect(err).NotTo(HaveOccurred())
	})

	It("does not apply later migrations if a migration fails", func() {
		migration1 := Migration{
			Name: "migration-1",
			Up: func(ctx context.Context, logger lager.Logger, tx *Tx) error {
				_, err := tx.ExecContext(ctx, "FAKE MIGRATION")
				return err
			},
		}
		migration2 := Migration{
			Name: "migration-2",
			Up: func(ctx context.Context, logger lager.Logger, tx *Tx) error {
				_, err := tx.ExecContext(ctx, "SHOULD NOT BE APPLIED")
				return err
			},
		}
		migrations := []Migration{migration1, migration2}

		mock.ExpectBegin()

		mock.ExpectExec("CREATE TABLE").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		mock.ExpectQuery("SELECT version, name, applied_at FROM " + migrationTableName).
			WillReturnRows(sqlmock.NewRows([]string{"version", "name", "applied_at"}))

		mock.ExpectBegin()
		mock.ExpectExec("FAKE MIGRATION").
			WillReturnError(errors.New("migration-failed"))
		mock.ExpectRollback()

		err = ApplyMigrations(ctx, fakeLogger, conn, migrationTableName, migrations)

		Expect(err).To(MatchError("migration-failed"))
	})

	It("does not apply later migrations if a migration commit fails", func() {
		migration1 := Migration{
			Name: "migration-1",
			Up: func(ctx context.Context, logger lager.Logger, tx *Tx) error {
				_, err := tx.ExecContext(ctx, "FAKE MIGRATION")
				return err
			},
		}
		migration2 := Migration{
			Name: "migration-2",
			Up: func(ctx context.Context, logger lager.Logger, tx *Tx) error {
				_, err := tx.ExecContext(ctx, "SHOULD NOT BE APPLIED")
				return err
			},
		}
		migrations := []Migration{migration1, migration2}

		mock.ExpectBegin()

		mock.ExpectExec("CREATE TABLE").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		mock.ExpectQuery("SELECT version, name, applied_at FROM " + migrationTableName).
			WillReturnRows(sqlmock.NewRows([]string{"version", "name", "applied_at"}))

		mock.ExpectBegin()
		mock.ExpectExec("FAKE MIGRATION").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit().
			WillReturnError(errors.New("commit-failed"))

		err = ApplyMigrations(ctx, fakeLogger, conn, migrationTableName, migrations)

		Expect(err).To(MatchError("commit-failed"))
	})
})

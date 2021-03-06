package integration_test

import (
	"code.cloudfoundry.org/perm/pkg/api/db"
	"code.cloudfoundry.org/perm/pkg/sqlx/sqlxtest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var testDB *sqlxtest.TestMySQLDB

var _ = BeforeSuite(func() {
	var err error

	testDB = sqlxtest.NewTestMySQLDB()
	err = testDB.Create(db.Migrations...)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := testDB.Drop()
	Expect(err).NotTo(HaveOccurred())
})

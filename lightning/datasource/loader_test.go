package datasource

import (
	"testing"

	. "github.com/pingcap/check"
	"github.com/pingcap/tidb-lightning/lightning/config"
)

var _ = Suite(&testMydumpLoaderSuite{})

func TestMydumps(t *testing.T) {
	TestingT(t)
}

type testMydumpLoaderSuite struct{}

func (s *testMydumpLoaderSuite) SetUpSuite(c *C)    {}
func (s *testMydumpLoaderSuite) TearDownSuite(c *C) {}

func (s *testMydumpLoaderSuite) TestLoader(c *C) {
	cfg := &config.Config{DataSource: config.DataSource{SourceDir: "./not-exists"}}
	mdl, err := NewMyDumpLoader(cfg)
	c.Assert(err, NotNil)

	cfg = &config.Config{DataSource: config.DataSource{SourceDir: "./examples"}}
	mdl, err = NewMyDumpLoader(cfg)
	c.Assert(err, IsNil)

	dbMeta := mdl.GetDatabase()
	c.Assert(dbMeta.Name, Equals, "mocker_test")
	c.Assert(len(dbMeta.Tables), Equals, 2)

	for _, table := range []string{"tbl_multi_index", "tbl_autoid"} {
		c.Assert(dbMeta.Tables[table].Name, Equals, table)
	}

	c.Assert(len(dbMeta.Tables["tbl_autoid"].DataFiles), Equals, 1)
	c.Assert(len(dbMeta.Tables["tbl_multi_index"].DataFiles), Equals, 1)
}

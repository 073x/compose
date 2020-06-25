package mobycli

import (
	"testing"

	"github.com/docker/api/tests/framework"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
)

type MobyExecSuite struct {
	framework.CliSuite
}

func (sut *MobyExecSuite) TestDelegateContextTypeToMoby() {
	Expect(mustDelegateToMoby("moby")).To(BeTrue())
	Expect(mustDelegateToMoby("aws")).To(BeTrue())
	Expect(mustDelegateToMoby("aci")).To(BeFalse())
}

func TestExec(t *testing.T) {
	RegisterTestingT(t)
	suite.Run(t, new(MobyExecSuite))
}

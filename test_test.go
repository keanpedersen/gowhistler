package gowhistler

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStuff(t *testing.T) {
	url := `http://test1.ekstern-test.nspop.dk:8080/stamdata-cpr-ws/service/DetGodeCPROpslag-1.0.4.1a?wsdl`

	ret, err := Parse(url)
	require.NoError(t, err)
	err = ret.Build()
	require.NoError(t, err)
	// urn:oio:medcom:cprservice:1.0.4.1a:internal_15
}

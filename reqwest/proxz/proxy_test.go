package proxz

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseProxyFromString(t *testing.T) {
	assert := require.New(t)
	raw := "http://username:pwd@127.0.0.1"
	config, err := ParseConfigFromString(raw)
	assert.Nil(err)
	assert.Equal(config.Proxy, "username:pwd@127.0.0.1")
	assert.Equal(config.Protocol, ProtocolHttp)

	config, err = ParseConfigFromString("socks://127.0.0.1:8900")
	assert.Nil(err)
	assert.Equal(config.Protocol, ProtocolSocks)
	assert.Equal(config.Proxy, "127.0.0.1:8900")

	config, err = ParseConfigFromString("127.0.0.1:8900")
	assert.NotNil(err)

	providerBuilders["zhima"] = nil
	config, err = ParseConfigFromString("http://zhima?chan=zl&topic=default&size=100")
	assert.Nil(err)
	assert.Equal(config.Provider, "zhima")
	assert.Equal(config.Params.Get("chan"), "zl")
	assert.Equal(config.GetSizeOrDefault(30), 100)

}

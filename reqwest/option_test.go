package reqwest

import (
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestOption(t *testing.T) {
	g := dot.Reqwest{}
	defaults.SetDefaults(&g)

	option := &Option{
		Reqwest: g,
	}

	assert := require.New(t)
	assert.Equal(option.Idle, 5*time.Second)

	ops := WithIdle(10 * time.Second)
	ops(option)

	assert.Equal(option.Idle, 10*time.Second)
	assert.Equal(g.Idle, 5*time.Second)
}

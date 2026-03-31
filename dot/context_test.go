package dot

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestWithDefaultSinker(t *testing.T) {
	assert := require.New(t)
	ctx := NewContext(true, log.NewEntry(log.New()))
	ctx.WithSinker()
	assert.NotNil(ctx.Sinker())
	assert.IsType(&DebugSinker{}, ctx.Sinker())
}

func TestWithKafkaSinker(t *testing.T) {
	assert := require.New(t)
	ctx := NewContext(false, log.NewEntry(log.New()))
	ctx.WithSinker()
	assert.NotNil(ctx.Sinker())
	assert.IsType(&KafkaSinker{}, ctx.Sinker())
}

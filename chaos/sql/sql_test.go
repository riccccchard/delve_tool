package sql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInvade(t *testing.T) {
	h, err := NewSQLHacker(69410)
	assert.Nil(t, err)
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	assert.Nil(t, h.Invade(ctx, 10*time.Second))
}

package logger

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromCtx(t *testing.T) {
	dl := NewLogger(envDev)
	_ = NewLogger(envProd)

	dl.Error("test", ErrorAttr(errors.New("error")))

	lctx := ToCtx(context.Background(), dl)

	assert.IsType(t, &slog.Logger{}, FromCtx(lctx))
	assert.IsType(t, &slog.Logger{}, FromCtx(context.Background()))
}

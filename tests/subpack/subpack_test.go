package subpack

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFailBuildExample(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "pass test",
		},
		{
			name: "pass test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			time.Sleep(3 * time.Second)
			assert.Equal(t, tt.name, "pass test")
		})
	}
}

package tests

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExample(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "pass test",
		},
		{
			name: "fail test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.name, "pass test")
		})
	}
}

func TestExampleNumer(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "new pass test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.name, "new pass test")
		})
	}
}

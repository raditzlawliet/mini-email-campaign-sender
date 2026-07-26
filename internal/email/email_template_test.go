package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]string
		want     string
	}{
		{
			name:     "single placeholder",
			template: "Hello {name}!",
			data:     map[string]string{"name": "Alice"},
			want:     "Hello Alice!",
		},
		{
			name:     "multiple placeholders",
			template: "{greeting} {name}, your email is {email}",
			data:     map[string]string{"greeting": "Hi", "name": "Bob", "email": "bob@example.com"},
			want:     "Hi Bob, your email is bob@example.com",
		},
		{
			name:     "no placeholders",
			template: "Hello world",
			data:     map[string]string{},
			want:     "Hello world",
		},
		{
			name:     "unknown placeholder left as-is",
			template: "Hello {unknown}!",
			data:     map[string]string{"name": "Alice"},
			want:     "Hello {unknown}!",
		},
		{
			name:     "empty data map",
			template: "Hi {name}",
			data:     map[string]string{},
			want:     "Hi {name}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got := Render(tt.template, tt.data)
			assert.Equal(tt.want, got)
		})
	}
}

func TestNewSender(t *testing.T) {
	t.Run("unknown provider", func(t *testing.T) {
		assert := assert.New(t)
		_, err := NewSender(SenderConfig{Provider: "unknown"})
		assert.Error(err)
		assert.Contains(err.Error(), "unknown email provider")
	})
}

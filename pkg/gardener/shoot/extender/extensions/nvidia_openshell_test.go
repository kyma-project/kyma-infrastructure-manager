package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNvidiaOpenshellExtension(t *testing.T) {
	t.Run("Should create NVIDIA OpenShell extension", func(t *testing.T) {
		ext, err := NewNvidiaOpenshellExtension()

		require.NoError(t, err)
		require.NotNil(t, ext)
		assert.Equal(t, NvidiaOpenshellExtensionType, ext.Type)
		require.NotNil(t, ext.Disabled)
		assert.Equal(t, false, *ext.Disabled)
		assert.Nil(t, ext.ProviderConfig)
	})
}

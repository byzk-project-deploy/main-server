package security

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDataEnvelope(t *testing.T) {
	rawData := []byte("123abc你好啊")

	a := assert.New(t)

	encData, err := Instance.DataEnvelope(rawData)
	if !a.NoError(err) {
		return
	}

	decryptData, err := Instance.DataUnEnvelope(encData)
	if !a.NoError(err) {
		return
	}

	a.Equal(rawData, decryptData)

}

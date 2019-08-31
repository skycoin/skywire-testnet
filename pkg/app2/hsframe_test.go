package app2

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHSFrame(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		body := struct {
			Test string `json:"test"`
		}{
			Test: "some string",
		}

		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err)

		procID := ProcID(1)
		frameType := HSFrameTypeDMSGListen
		bodyLen := len(bodyBytes)

		hsFrame, err := NewHSFrame(procID, frameType, body)
		require.NoError(t, err)

		require.Equal(t, len(hsFrame), HSFrameHeaderLen+len(bodyBytes))

		gotProcID := ProcID(binary.BigEndian.Uint16(hsFrame))
		require.Equal(t, gotProcID, procID)

		gotFrameType := HSFrameType(hsFrame[HSFrameProcIDLen])
		require.Equal(t, gotFrameType, frameType)

		gotBodyLen := int(binary.BigEndian.Uint16(hsFrame[HSFrameProcIDLen+HSFrameTypeLen:]))
		require.Equal(t, gotBodyLen, bodyLen)

		require.Equal(t, bodyBytes, []byte(hsFrame[HSFrameProcIDLen+HSFrameTypeLen+HSFrameBodyLenLen:]))

		gotProcID = hsFrame.ProcID()
		require.Equal(t, gotProcID, procID)

		gotFrameType = hsFrame.FrameType()
		require.Equal(t, gotFrameType, frameType)

		gotBodyLen = hsFrame.BodyLen()
		require.Equal(t, gotBodyLen, bodyLen)
	})

	t.Run("fail - too large body", func(t *testing.T) {
		body := struct {
			Test string `json:"test"`
		}{
			Test: "some string",
		}

		for len(body.Test) <= HSFrameMaxBodyLen {
			body.Test += body.Test
		}

		procID := ProcID(1)
		frameType := HSFrameTypeDMSGListen

		_, err := NewHSFrame(procID, frameType, body)
		require.Equal(t, err, ErrHSFrameBodyTooLarge)
	})
}

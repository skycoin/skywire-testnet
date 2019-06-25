package dmsg

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skywire/internal/ioutil"
	"github.com/skycoin/skywire/pkg/cipher"
)

func Test_isInitiatorID(t *testing.T) {
	type args struct {
		tpID uint16
	}

	cases := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Initiator ID",
			args: args{tpID: 2},
			want: true,
		},
		{
			name: "Not initiator ID",
			args: args{tpID: 1},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isInitiatorID(tc.args.tpID)
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_randID(t *testing.T) {
	type args struct {
		initiator bool
	}

	cases := []struct {
		name   string
		args   args
		isEven bool
	}{
		{
			name:   "Even number",
			args:   args{initiator: true},
			isEven: true,
		},
		{
			name:   "Odd number",
			args:   args{initiator: false},
			isEven: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := randID(tc.args.initiator)
			isEven := got%2 == 0

			assert.Equal(t, tc.isEven, isEven)
		})
	}
}

func TestFrameType_String(t *testing.T) {
	cases := []struct {
		name string
		ft   FrameType
		want string
	}{
		{
			name: "Request type",
			ft:   RequestType,
			want: "REQUEST",
		},
		{
			name: "Accept type",
			ft:   AcceptType,
			want: "ACCEPT",
		},
		{
			name: "Close type",
			ft:   CloseType,
			want: "CLOSE",
		},
		{
			name: "Fwd type",
			ft:   FwdType,
			want: "FWD",
		},
		{
			name: "Ack type",
			ft:   AckType,
			want: "ACK",
		},
		{
			name: "Ok type",
			ft:   OkType,
			want: "OK",
		},
		{
			name: "Unknown type",
			ft:   255,
			want: "UNKNOWN:255",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ft.String()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMakeFrame(t *testing.T) {
	type args struct {
		ft   FrameType
		chID uint16
		pay  []byte
	}

	cases := []struct {
		name string
		args args
		want Frame
	}{
		{
			name: "Example 1",
			args: args{
				ft:   RequestType,
				chID: 2,
				pay:  []byte{0x03, 0x04, 0x05},
			},
			want: Frame{0x01, 0x00, 0x02, 0x00, 0x03, 0x03, 0x04, 0x05},
		},
		{
			name: "Example 2",
			args: args{
				ft:   0xFF,
				chID: 0xABCD,
				pay:  []byte{0x10, 0x20, 0x30},
			},
			want: Frame{0xFF, 0xAB, 0xCD, 0x00, 0x03, 0x10, 0x20, 0x30},
		},
		{
			name: "Payload length > 65535 is not supported",
			args: args{
				ft:   0xAB,
				chID: 0xCDEF,
				pay:  make([]byte, 65536),
			},
			want: append([]byte{0xAB, 0xCD, 0xEF, 0, 0x00}, make([]byte, 65536)...),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MakeFrame(tc.args.ft, tc.args.chID, tc.args.pay)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFrame_Type(t *testing.T) {
	cases := []struct {
		name string
		f    Frame
		want FrameType
	}{
		{
			name: "Request type",
			f:    Frame{1},
			want: RequestType,
		},
		{
			name: "Accept type",
			f:    Frame{2},
			want: AcceptType,
		},
		{
			name: "Close type",
			f:    Frame{3},
			want: CloseType,
		},
		{
			name: "Fwd type",
			f:    Frame{10},
			want: FwdType,
		},
		{
			name: "Ack type",
			f:    Frame{11},
			want: AckType,
		},
		{
			name: "Unknown type",
			f:    Frame{255},
			want: FrameType(255),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f.Type()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFrame_TpID(t *testing.T) {
	cases := []struct {
		name string
		f    Frame
		want uint16
	}{
		{
			name: "Example 1",
			f:    Frame{0, 0x00, 0x01},
			want: 0x01,
		},
		{
			name: "Example 2",
			f:    Frame{0, 0xAB, 0xCD},
			want: 0xABCD,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f.TpID()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFrame_PayLen(t *testing.T) {
	cases := []struct {
		name string
		f    Frame
		want int
	}{
		{
			name: "Example 1",
			f:    Frame{0, 0x00, 0x00, 0x00, 0x01},
			want: 0x01,
		},
		{
			name: "Example 2",
			f:    Frame{0, 0x00, 0x00, 0xAB, 0xCD},
			want: 0xABCD,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f.PayLen()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFrame_Pay(t *testing.T) {
	cases := []struct {
		name string
		f    Frame
		want []byte
	}{
		{
			name: "Empty payload",
			f:    Frame{0, 0x00, 0x00, 0x00, 0x00},
			want: []byte{},
		},
		{
			name: "Two-byte payload",
			f:    Frame{0, 0x00, 0x00, 0x00, 0x01, 0xAB, 0xCD},
			want: []byte{0xAB, 0xCD},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f.Pay()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFrame_Disassemble(t *testing.T) {
	cases := []struct {
		name   string
		f      Frame
		wantFT FrameType
		wantID uint16
		wantP  []byte
	}{
		{
			name:   "Example 1",
			f:      Frame{0x01, 0x00, 0x02, 0x00, 0x03, 0x03, 0x04, 0x05},
			wantFT: RequestType,
			wantID: 2,
			wantP:  []byte{0x03, 0x04, 0x05},
		},
		{
			name:   "Example 2",
			f:      Frame{0xFF, 0xAB, 0xCD, 0x00, 0x03, 0x10, 0x20, 0x30},
			wantFT: 0xFF,
			wantID: 0xABCD,
			wantP:  []byte{0x10, 0x20, 0x30},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotFT, gotID, gotP := tc.f.Disassemble()

			assert.Equal(t, tc.wantFT, gotFT)
			assert.Equal(t, tc.wantID, gotID)
			assert.Equal(t, tc.wantP, gotP)
		})
	}
}

func Test_readFrame(t *testing.T) {
	type args struct {
		r io.Reader
	}

	cases := []struct {
		name    string
		args    args
		want    Frame
		wantErr error
	}{
		{
			name:    "Payload length equals to required",
			args:    args{r: bytes.NewReader([]byte{0x01, 0x00, 0x02, 0x00, 0x03, 0x03, 0x04, 0x05})},
			want:    Frame{0x01, 0x00, 0x02, 0x00, 0x03, 0x03, 0x04, 0x05},
			wantErr: nil,
		},
		{
			name:    "Payload longer than required",
			args:    args{r: bytes.NewReader(append([]byte{0x01, 0x00, 0x02, 0x00, 0x03, 0x03, 0x04, 0x05}, make([]byte, 10)...))},
			want:    Frame{0x01, 0x00, 0x02, 0x00, 0x03, 0x03, 0x04, 0x05},
			wantErr: nil,
		},
		{
			name:    "Payload shorter than required",
			args:    args{r: bytes.NewReader(append([]byte{0x01, 0x00, 0x02, 0x00, 0x03}))},
			want:    Frame{0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x00, 0x00},
			wantErr: io.EOF,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readFrame(tc.args.r)

			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_writeFrame(t *testing.T) {
	type args struct {
		f Frame
	}

	cases := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
	}{
		{
			name:    "Example 1",
			args:    args{f: Frame{0xFF, 0xAB, 0xCD, 0x00, 0x03, 0x10, 0x20, 0x30}},
			want:    []byte{0xFF, 0xAB, 0xCD, 0x00, 0x03, 0x10, 0x20, 0x30},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &bytes.Buffer{}

			err := writeFrame(w, tc.args.f)
			assert.Equal(t, tc.wantErr, err)

			got := w.Bytes()
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_writeCloseFrame(t *testing.T) {
	type args struct {
		id     uint16
		reason byte
	}

	cases := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
	}{
		{
			name: "Example 1",
			args: args{
				id:     0xABCD,
				reason: 0xEF,
			},
			want:    []byte{0x03, 0xAB, 0xCD, 0x00, 0x01, 0xEF},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &bytes.Buffer{}

			err := writeCloseFrame(w, tc.args.id, tc.args.reason)
			assert.Equal(t, tc.wantErr, err)

			got := w.Bytes()
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_writeFwdFrame(t *testing.T) {
	type args struct {
		id  uint16
		seq ioutil.Uint16Seq
		p   []byte
	}

	cases := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
	}{
		{
			name: "Example 1",
			args: args{
				id:  0xABCD,
				seq: 0xEF01,
				p:   []byte{0x23, 0x45, 0x67},
			},
			want:    []byte{0x0A, 0xAB, 0xCD, 0x00, 0x05, 0xEF, 0x01, 0x23, 0x45, 0x67},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &bytes.Buffer{}

			err := writeFwdFrame(w, tc.args.id, tc.args.seq, tc.args.p)
			assert.Equal(t, tc.wantErr, err)

			got := w.Bytes()
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_combinePKs(t *testing.T) {
	type args struct {
		initPK string
		respPK string
	}

	cases := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Example 1",
			args: args{
				initPK: "024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7",
				respPK: "031b80cd5773143a39d940dc0710b93dcccc262a85108018a7a95ab9af734f8055",
			},
			want: "024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7031b80cd5773143a39d940dc0710b93dcccc262a85108018a7a95ab9af734f8055",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var initPK, respPK cipher.PubKey

			err := initPK.Set(tc.args.initPK)
			assert.NoError(t, err)

			err = respPK.Set(tc.args.respPK)
			assert.NoError(t, err)

			got := combinePKs(initPK, respPK)
			assert.Equal(t, tc.want, hex.EncodeToString(got))
		})
	}
}

func Test_splitPKs(t *testing.T) {
	type args struct {
		s string
	}

	cases := []struct {
		name       string
		args       args
		wantInitPK string
		wantRespPK string
		wantOk     bool
	}{
		{
			name:       "OK",
			args:       args{s: "024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7031b80cd5773143a39d940dc0710b93dcccc262a85108018a7a95ab9af734f8055"},
			wantInitPK: "024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7",
			wantRespPK: "031b80cd5773143a39d940dc0710b93dcccc262a85108018a7a95ab9af734f8055",
			wantOk:     true,
		},
		{
			name:       "Not OK",
			args:       args{s: "024ec47420176680816e0406250e7156465e4531f5b26057c9f6297bb0303558c7"},
			wantInitPK: "000000000000000000000000000000000000000000000000000000000000000000",
			wantRespPK: "000000000000000000000000000000000000000000000000000000000000000000",
			wantOk:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pks, err := hex.DecodeString(tc.args.s)
			assert.NoError(t, err)

			gotInitPK, gotRespPK, gotOk := splitPKs(pks)
			assert.Equal(t, tc.wantOk, gotOk)
			assert.Equal(t, tc.wantInitPK, gotInitPK.Hex())
			assert.Equal(t, tc.wantRespPK, gotRespPK.Hex())
		})
	}
}

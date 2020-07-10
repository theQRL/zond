package crypto

import (
	"github.com/theQRL/qrllib/goqrllib/dilithium"
	"github.com/theQRL/zond/misc"
	"runtime"
)

type DilithiumInterface interface {
	Sign(message []byte) []byte
	Verify(signature []byte, pk []byte, message []byte) bool
}

type Dilithium struct {
	d dilithium.Dilithium
}

func NewDilithium(d dilithium.Dilithium) *Dilithium {
	dilith := &Dilithium{d}

	// Finalizer to clean up memory allocated by C++ when object becomes unreachable
	runtime.SetFinalizer(d,
		func(d *Dilithium) {
			dilithium.DeleteDilithium(d.d)
		})
	return dilith
}

func (d *Dilithium) Sign(message []byte) []byte {
	msg := misc.UcharVector{}
	msg.New(d.d.Sign(misc.BytesToUCharVector(message)))
	return msg.GetBytes()
}

func Verify(signature []byte, pk []byte, message []byte) bool {
	dataOut := misc.Int64ToUCharVector(int64(len(message)))
	dilithium.DilithiumSign_open(dataOut, misc.BytesToUCharVector(signature),
		misc.BytesToUCharVector(pk))

	return dataOut.Size() == int64(len(message))
}

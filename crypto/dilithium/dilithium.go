package dilithium

import (
	"github.com/theQRL/qrllib/goqrllib/dilithium"
	"github.com/theQRL/zond/misc"
	"reflect"
	"runtime"
)

type DilithiumInterface interface {
	Sign(message []byte) []byte
}

type Dilithium struct {
	d dilithium.Dilithium
}

func (d *Dilithium) PK() []byte {
	return misc.UCharVectorToBytes(d.d.GetPK())
}

func (d *Dilithium) SK() []byte {
	return misc.UCharVectorToBytes(d.d.GetSK())
}

func (d *Dilithium) Sign(message []byte) []byte {
	msg := misc.UcharVector{}
	msg.New(d.d.Sign(misc.BytesToUCharVector(message)))
	return msg.GetBytes()
}

func DilithiumVerify(signature []byte, pk []byte, message []byte) bool {
	dataOut := misc.Int64ToUCharVector(int64(len(message)))
	dilithium.DilithiumSign_open(dataOut, misc.BytesToUCharVector(signature),
		misc.BytesToUCharVector(pk))

	bytesData := misc.UCharVectorToBytes(dilithium.DilithiumExtract_message(dataOut))
	return reflect.DeepEqual(bytesData, message)
}

func RecoverDilithium(pk []byte, sk []byte) *Dilithium {
	d := dilithium.NewDilithium__SWIG_1(misc.BytesToUCharVector(pk), misc.BytesToUCharVector(sk))
	dilith := &Dilithium{d}

	// Finalizer to clean up memory allocated by C++ when object becomes unreachable
	runtime.SetFinalizer(dilith,
		func(d *Dilithium) {
			dilithium.DeleteDilithium(d.d)
		})
	return dilith
}

func NewDilithium() *Dilithium {
	d := dilithium.NewDilithium__SWIG_0()
	dilith := &Dilithium{d}

	// Finalizer to clean up memory allocated by C++ when object becomes unreachable
	runtime.SetFinalizer(dilith,
		func(d *Dilithium) {
			dilithium.DeleteDilithium(d.d)
		})
	return dilith
}

package crypto

import (
	"encoding/hex"
	"fmt"
	"github.com/theQRL/qrllib/goqrllib/goqrllib"
	"github.com/theQRL/zond/misc"
	"runtime"
)

var hashFunctionsReverse = map[goqrllib.EHashFunction]string{
	goqrllib.SHAKE_128: "shake128",
	goqrllib.SHAKE_256: "shake256",
	goqrllib.SHA2_256:  "sha2_256",
}

type XMSSInterface interface {
	HashFunction() string

	SignatureType() goqrllib.ESignatureType

	Height() uint64

	sk() []byte

	pk() []byte

	NumberSignatures() uint64

	RemainingSignatures() uint64

	Mnemonic() string

	Address() []byte

	QAddress() string

	OTSIndex() uint64

	SetOTSIndex(newIndex uint)

	HexSeed() string

	ExtendedSeed() string

	Seed() string

	Sign(message []byte) []byte
}

type XMSS struct {
	xmss goqrllib.XmssFast
}

func NewXMSS(xmssFast goqrllib.XmssFast) *XMSS {
	x := &XMSS{xmssFast}

	// Finalizer to clean up memory allocated by C++ when object becomes unreachable
	runtime.SetFinalizer(x,
		func(x *XMSS) {
			goqrllib.DeleteXmssFast(x.xmss)
		})
	return x
}

func FromExtendedSeed(extendedSeedBytes []byte) *XMSS {
	extendedSeed := misc.BytesToUCharVector(extendedSeedBytes)
	moddedExtendedSeed := misc.NewUCharVector()
	moddedExtendedSeed.New(extendedSeed.GetData())
	if extendedSeed.GetData().Size() != 51 {
		message := fmt.Sprintf("Extended seed size not equals to 51 %v", extendedSeed.GetData().Size())
		panic(message)
	}

	tmp := misc.NewUCharVector()
	tmp.AddBytes(moddedExtendedSeed.GetBytes()[0:3])
	descr := goqrllib.QRLDescriptorFromBytes(tmp.GetData())

	if descr.GetSignatureType() != goqrllib.XMSS {
		message := fmt.Sprintf("Signature Type not supported %v", descr.GetSignatureType())
		panic(message)
	}

	height := descr.GetHeight()
	hashFunction := descr.GetHashFunction()
	tmp = misc.NewUCharVector()
	tmp.AddBytes(moddedExtendedSeed.GetBytes()[3:])

	return NewXMSS(goqrllib.NewXmssFast__SWIG_1(tmp.GetData(), height, hashFunction))
}

func FromHeight(treeHeight uint64, hashFunction goqrllib.EHashFunction) *XMSS {
	seed := goqrllib.GetRandomSeed(48, "")
	return NewXMSS(goqrllib.NewXmssFast__SWIG_1(seed, byte(treeHeight), hashFunction))
}

func (x *XMSS) HashFunction() string {
	descr := x.xmss.GetDescriptor()
	eHashFunction := descr.GetHashFunction()
	functionName, ok := hashFunctionsReverse[eHashFunction]
	if !ok {
		message := fmt.Sprintf("Invalid eHashFunction %v", eHashFunction)
		panic(message)
	}
	return functionName

}

func (x *XMSS) SignatureType() goqrllib.ESignatureType {
	descr := x.xmss.GetDescriptor()
	return descr.GetSignatureType()
}

func (x *XMSS) Height() uint64 {
	return uint64(x.xmss.GetHeight())
}

func (x *XMSS) sk() []byte {
	return misc.UCharVectorToBytes(x.xmss.GetSK())
}

func (x *XMSS) PK() []byte {
	return misc.UCharVectorToBytes(x.xmss.GetPK())
}

func (x *XMSS) NumberSignatures() uint64 {
	return uint64(x.xmss.GetNumberSignatures())
}

func (x *XMSS) RemainingSignatures() uint64 {
	return uint64(x.xmss.GetRemainingSignatures())
}

func (x *XMSS) Mnemonic() string {
	return goqrllib.Bin2mnemonic(x.xmss.GetExtendedSeed())
}

func (x *XMSS) Address() []byte {
	return misc.UCharVectorToBytes(x.xmss.GetAddress())
}

func (x *XMSS) StrAddress() string {
	return fmt.Sprintf("%s", hex.EncodeToString(x.Address()))
}

func (x *XMSS) OTSIndex() uint64 {
	return uint64(x.xmss.GetIndex())
}

func (x *XMSS) SetOTSIndex(newIndex uint) {
	x.xmss.SetIndex(newIndex)
}

func (x *XMSS) HexSeed() string {
	u := misc.NewUCharVector()
	u.New(x.xmss.GetExtendedSeed())
	return hex.EncodeToString(u.GetBytes())
}

func (x *XMSS) ExtendedSeed() *misc.UcharVector {
	u := misc.NewUCharVector()
	u.New(x.xmss.GetExtendedSeed())
	return u
}

func (x *XMSS) Seed() *misc.UcharVector {
	u := misc.NewUCharVector()
	u.New(x.xmss.GetSeed())
	return u
}

func (x *XMSS) Sign(message []byte) []byte {
	msg := misc.NewUCharVector()
	msg.New(x.xmss.Sign(misc.BytesToUCharVector(message).GetData()))
	return msg.GetBytes()
}

func XMSSVerify(message []byte, signature []byte, pk []byte) bool {
	return goqrllib.XmssFastVerify(misc.BytesToUCharVector(message).GetData(),
		misc.BytesToUCharVector(signature).GetData(), misc.BytesToUCharVector(pk).GetData())
}

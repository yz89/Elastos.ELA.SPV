package code

import (
	"io"

	. "github.com/elastos/Elastos.ELA.SPV/core/contract"
	"github.com/elastos/Elastos.ELA.SPV/common/serialization"
)

type FunctionCode struct {
	// Contract Code
	Code []byte

	// Contract parameter type list
	ParameterTypes []ContractParameterType

	// Contract return type list
	ReturnTypes []ContractParameterType
}

// method of SerializableData
func (fc *FunctionCode) Serialize(w io.Writer) error {
	err := serialization.WriteVarBytes(w, ContractParameterTypeToByte(fc.ParameterTypes))
	if err != nil {
		return err
	}

	err = serialization.WriteVarBytes(w, fc.Code)
	if err != nil {
		return err
	}

	return nil
}

// method of SerializableData
func (fc *FunctionCode) Deserialize(r io.Reader) error {
	p, err := serialization.ReadVarBytes(r)
	if err != nil {
		return err
	}
	fc.ParameterTypes = ByteToContractParameterType(p)

	fc.Code, err = serialization.ReadVarBytes(r)
	if err != nil {
		return err
	}

	return nil
}

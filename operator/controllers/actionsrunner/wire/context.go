package wire

import (
	"encoding/json"
	"fmt"
)

type Flattener interface {
	Flatten() (interface{}, error)
}

type PipelineContextDataType int32

const (
	PipelineContextDataTypeString                  PipelineContextDataType = 0
	PipelineContextDataTypeArray                   PipelineContextDataType = 1
	PipelineContextDataTypeDictionary              PipelineContextDataType = 2
	PipelineContextDataTypeBoolean                 PipelineContextDataType = 3
	PipelineContextDataTypeNumber                  PipelineContextDataType = 4
	PipelineContextDataTypeCaseSensitiveDictionary PipelineContextDataType = 5
)

type PipelineContextData struct {
	Type PipelineContextDataType `json:"t"`
	JSON json.RawMessage
}

func (pcd *PipelineContextData) Flatten() (interface{}, error) {
	switch t := pcd.Type; t {
	case PipelineContextDataTypeString:
		scd, err := pcd.ToStringContextData()
		if err != nil {
			return nil, err
		}

		return scd.Flatten()

	case PipelineContextDataTypeArray:
		acd, err := pcd.ToArrayContextData()
		if err != nil {
			return nil, err
		}

		return acd.Flatten()

	case PipelineContextDataTypeDictionary, PipelineContextDataTypeCaseSensitiveDictionary:
		dcd, err := pcd.ToDictionaryContextData()
		if err != nil {
			return nil, err
		}

		return dcd.Flatten()

	case PipelineContextDataTypeBoolean:
		bcd, err := pcd.ToBooleanContextData()
		if err != nil {
			return nil, err
		}

		return bcd.Flatten()

	case PipelineContextDataTypeNumber:
		ncd, err := pcd.ToNumberContextData()
		if err != nil {
			return nil, err
		}

		return ncd.Flatten()

	default:
		return nil, fmt.Errorf("unknown pipeline context data type: %d", t)
	}
}

func (pcd *PipelineContextData) ToStringContextData() (*StringContextData, error) {
	var scd StringContextData
	if err := json.Unmarshal(pcd.JSON, &scd); err != nil {
		return nil, err
	}

	return &scd, nil
}

func (pcd *PipelineContextData) ToArrayContextData() (*ArrayContextData, error) {
	var acd ArrayContextData
	if err := json.Unmarshal(pcd.JSON, &acd); err != nil {
		return nil, err
	}

	return &acd, nil
}

func (pcd *PipelineContextData) ToDictionaryContextData() (*DictionaryContextData, error) {
	var dcd DictionaryContextData
	if err := json.Unmarshal(pcd.JSON, &dcd); err != nil {
		return nil, err
	}

	return &dcd, nil
}

func (pcd *PipelineContextData) ToBooleanContextData() (*BooleanContextData, error) {
	var bcd BooleanContextData
	if err := json.Unmarshal(pcd.JSON, &bcd); err != nil {
		return nil, err
	}

	return &bcd, nil
}

func (pcd *PipelineContextData) ToNumberContextData() (*NumberContextData, error) {
	var ncd NumberContextData
	if err := json.Unmarshal(pcd.JSON, &ncd); err != nil {
		return nil, err
	}

	return &ncd, nil
}

type StringContextData struct {
	String bool `json:"s"`
}

var _ Flattener = (*StringContextData)(nil)

func (scd *StringContextData) Flatten() (interface{}, error) {
	return scd.String, nil
}

type ArrayContextData struct {
	Array []PipelineContextData `json:"a"`
}

var _ Flattener = (*ArrayContextData)(nil)

func (acd *ArrayContextData) Flatten() (interface{}, error) {
	is := make([]interface{}, len(acd.Array))
	for i, data := range acd.Array {
		d, err := data.Flatten()
		if err != nil {
			return nil, err
		}

		is[i] = d
	}

	return is, nil
}

type DictionaryContextDataPair struct {
	Key string              `json:"k"`
	Val PipelineContextData `json:"v"`
}

type DictionaryContextData struct {
	Dictionary []DictionaryContextDataPair `json:"d"`
}

var _ Flattener = (*DictionaryContextData)(nil)

func (dcd *DictionaryContextData) Flatten() (interface{}, error) {
	msi := make(map[string]interface{}, len(dcd.Dictionary))
	for _, pair := range dcd.Dictionary {
		v, err := pair.Val.Flatten()
		if err != nil {
			return nil, err
		}

		msi[pair.Key] = v
	}

	return msi, nil
}

type BooleanContextData struct {
	Boolean bool `json:"b"`
}

var _ Flattener = (*BooleanContextData)(nil)

func (bcd *BooleanContextData) Flatten() (interface{}, error) {
	return bcd.Boolean, nil
}

type NumberContextData struct {
	Number float64 `json:"n"`
}

var _ Flattener = (*NumberContextData)(nil)

func (ncd *NumberContextData) Flatten() (interface{}, error) {
	return ncd.Number, nil
}

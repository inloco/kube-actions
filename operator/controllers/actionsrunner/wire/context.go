package wire

import (
	"encoding/json"
	"errors"
	"fmt"
)

type PipelineContextDataType int32

const (
	PipelineContextDataTypeString                  PipelineContextDataType = 0
	PipelineContextDataTypeArray                   PipelineContextDataType = 1
	PipelineContextDataTypeDictionary              PipelineContextDataType = 2
	PipelineContextDataTypeBoolean                 PipelineContextDataType = 3
	PipelineContextDataTypeNumber                  PipelineContextDataType = 4
	PipelineContextDataTypeCaseSensitiveDictionary PipelineContextDataType = 5
)

type DictionaryContextDataPair struct {
	Key string              `json:"k"`
	Val PipelineContextData `json:"v"`
}

var _ json.Unmarshaler = (*DictionaryContextDataPair)(nil)

func (dcdp *DictionaryContextDataPair) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	ki, ok := m["k"]
	if !ok {
		return errors.New(`m["k"] == nil`)
	}

	ks, ok := ki.(string)
	if !ok {
		return errors.New(`ki.(string) == nil`)
	}

	vi, ok := m["v"]
	if !ok {
		return errors.New(`m["v"] == nil`)
	}

	var pcd PipelineContextData
	switch vi.(type) {
	case string:
		pcd.Type = PipelineContextDataTypeString
		pcd.String = vi.(string)

	case bool:
		pcd.Type = PipelineContextDataTypeBoolean
		pcd.Boolean = vi.(bool)

	case float64:
		pcd.Type = PipelineContextDataTypeNumber
		pcd.Number = vi.(float64)

	default:
		i, err := json.Marshal(vi)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(i, &pcd); err != nil {
			return err
		}
	}

	dcdp.Key = ks
	dcdp.Val = pcd

	return nil
}

type PipelineContextData struct {
	Type       PipelineContextDataType     `json:"t"`
	String     string                      `json:"s"`
	Array      []PipelineContextData       `json:"a"`
	Dictionary []DictionaryContextDataPair `json:"d"`
	Boolean    bool                        `json:"b"`
	Number     float64                     `json:"n"`
}

func (pcd *PipelineContextData) Flattened() (interface{}, error) {
	switch t := pcd.Type; t {
	case PipelineContextDataTypeString:
		return pcd.String, nil

	case PipelineContextDataTypeArray:
		is := make([]interface{}, len(pcd.Array))
		for i, data := range pcd.Array {
			d, err := data.Flattened()
			if err != nil {
				return nil, err
			}

			is[i] = d
		}

		return is, nil

	case PipelineContextDataTypeDictionary, PipelineContextDataTypeCaseSensitiveDictionary:
		msi := make(map[string]interface{}, len(pcd.Dictionary))
		for _, pair := range pcd.Dictionary {
			v, err := pair.Val.Flattened()
			if err != nil {
				return nil, err
			}

			msi[pair.Key] = v
		}

		return msi, nil

	case PipelineContextDataTypeBoolean:
		return pcd.Boolean, nil

	case PipelineContextDataTypeNumber:
		return pcd.Number, nil

	default:
		return nil, fmt.Errorf("unknown pipeline context data type: %d", t)
	}
}

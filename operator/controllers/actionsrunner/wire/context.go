package wire

import (
	"encoding/json"
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

type ArrayContextData []PipelineContextData

type DictionaryContextDataPair struct {
	Key string              `json:"k"`
	Val PipelineContextData `json:"v"`
}

type DictionaryContextData []DictionaryContextDataPair

type pipelineContextData struct {
	Type       PipelineContextDataType `json:"t"`
	String     string                  `json:"s"`
	Array      ArrayContextData        `json:"a"`
	Dictionary DictionaryContextData   `json:"d"`
	Boolean    bool                    `json:"b"`
	Number     float64                 `json:"n"`
}

type PipelineContextData struct {
	pipelineContextData `json:",inline"`
}

var _ json.Unmarshaler = (*PipelineContextData)(nil)

func (pcd *PipelineContextData) UnmarshalJSON(data []byte) error {
	var i interface{}
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}

	switch i.(type) {
	case string:
		pcd.Type = PipelineContextDataTypeString
		pcd.String = i.(string)

	case bool:
		pcd.Type = PipelineContextDataTypeBoolean
		pcd.Boolean = i.(bool)

	case float64:
		pcd.Type = PipelineContextDataTypeNumber
		pcd.Number = i.(float64)

	default:
		if err := json.Unmarshal(data, &pcd.pipelineContextData); err != nil {
			return err
		}
	}

	return nil
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

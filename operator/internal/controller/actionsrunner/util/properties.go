package util

func GetPropertyValue(properties interface{}, key string) interface{} {
	mProperties, ok := properties.(map[string]interface{})
	if !ok {
		return nil
	}

	iProperty, ok := mProperties[key]
	if !ok {
		return nil
	}

	mProperty, ok := iProperty.(map[string]interface{})
	if !ok {
		return nil
	}

	iValue, ok := mProperty["$value"]
	if !ok {
		return nil
	}

	return iValue
}

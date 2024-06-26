package blockregistry

import "encoding/json"

func DeserializeBlockRegistry[R BlockRegistry](data []byte) (R, error) {
	var registry R
	if err := json.Unmarshal(data, &registry); err != nil {
		return registry, err
	}

	return registry, nil
}

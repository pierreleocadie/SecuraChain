package fileregistry

import "encoding/json"

func DeserializeFileRegistry[R FileRegistry](data []byte) (R, error) {
	var registry R
	if err := json.Unmarshal(data, &registry); err != nil {
		return registry, err
	}

	return registry, nil
}

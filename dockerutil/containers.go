package dockerutil

import "encoding/json"

func ExtractContainers(res interface{}) []map[string]interface{} {
	b, _ := json.Marshal(res)
	var m interface{}
	json.Unmarshal(b, &m)

	if list, ok := m.([]interface{}); ok {
		var ret []map[string]interface{}
		for _, item := range list {
			if mm, ok := item.(map[string]interface{}); ok {
				ret = append(ret, mm)
			}
		}
		return ret
	}
	if mm, ok := m.(map[string]interface{}); ok {
		for _, val := range mm {
			if list, ok := val.([]interface{}); ok {
				var ret []map[string]interface{}
				for _, item := range list {
					if mmm, ok := item.(map[string]interface{}); ok {
						ret = append(ret, mmm)
					}
				}
				return ret
			}
		}
	}
	return nil
}

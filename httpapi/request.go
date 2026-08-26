package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxJSONBody 单个 JSON 请求体上限（1 MiB），防止异常客户端拖垮服务。
const maxJSONBody = 1 << 20

// decodeJSON 从请求体解析 JSON，限制体积并拒绝「空体/多值」等非法输入。
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("请求体只能包含一个 JSON 对象")
	}
	return nil
}

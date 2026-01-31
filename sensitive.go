package main

import (
	"encoding/json"
	"net/url"
	"strings"
)

var sensitiveKeys = map[string]struct{}{
    "password": {},
    "pass": {},
    "token": {},
    "access_token": {},
    "refresh_token": {},
    "authorization": {},
    "secret": {},
}

func isSensitiveKey(k string) bool {
    k = strings.ToLower(k)
    _, ok := sensitiveKeys[k]
    return ok
}

func maskJSON(v any) any {
    switch val := v.(type) {

    case map[string]any:
        for k, v2 := range val {
            if isSensitiveKey(k) {
                val[k] = "***"
            } else {
                val[k] = maskJSON(v2)
            }
        }
        return val

    case []any:
        for i, v2 := range val {
            val[i] = maskJSON(v2)
        }
        return val

    default:
        return val
    }
}

func maskBody(body string, contentType string) string {

    // JSON
    if strings.Contains(contentType, "application/json") {
        var obj any
        if json.Unmarshal([]byte(body), &obj) == nil {
            masked := maskJSON(obj)
            out, _ := json.Marshal(masked)
            return string(out)
        }
    }

    // FORM URLENCODED
    if strings.Contains(contentType, "application/x-www-form-urlencoded") {
        vals, err := url.ParseQuery(body)
        if err == nil {
            for k := range vals {
                if isSensitiveKey(k) {
                    vals.Set(k, "***")
                }
            }
            return vals.Encode()
        }
    }

    return body
}

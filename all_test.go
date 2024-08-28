package bbfs

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseGitTimestamp(t *testing.T) {
	res := struct {
		Time int
	}{}
	const text = `{ "Time" : 1 } `

	if err := json.Unmarshal([]byte(text), &res); err != nil {
		t.Fatalf("error: %s", err.Error())
	}

}

func TestParseGitTimestampNum(t *testing.T) {
	tt := time.Unix(1722850024, 0)
	t.Errorf("%s", tt.String())
}

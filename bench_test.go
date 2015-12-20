package crowd

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
	"testing"
	"time"
)

var sess = StoredSession{
	Expires:    time.Now(),
	LastAccess: time.Now(),
	ID:         "j4haf8hlahj4haf8hlahj4haf8hlahh4",
	LoggedIn:   true,
	User:       "longestusernameever",
}

var jsonBuffer []byte
var gobBuffer []byte

func init() {
	var err error
	jsonBuffer, err = json.Marshal(sess)
	if err != nil {
		log.Println(err)
	}
	log.Println("JSON output is", len(jsonBuffer), "bytes long.")
	var out2 bytes.Buffer // Stand-in for a network connection
	enc := gob.NewEncoder(&out2)
	err = enc.Encode(sess)
	if err != nil {
		log.Println(err)
	}
	log.Println("Gob output is", out2.Len(), "bytes long.")
	gobBuffer = out2.Bytes()
}

func BenchmarkJSONSerialize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(sess)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkGobSerialize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var out bytes.Buffer
		enc := gob.NewEncoder(&out)
		err := enc.Encode(sess)
		if err != nil {
			b.Error(err)
		}
		_ = out.Bytes()
	}
}

func BenchmarkJSONDeserialize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var s StoredSession
		err := json.Unmarshal(jsonBuffer, &s)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkGobDeserialize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		in := bytes.NewBuffer(gobBuffer)
		dec := gob.NewDecoder(in)
		var s StoredSession
		err := dec.Decode(&s)
		if err != nil {
			b.Error(err)
		}
	}
}

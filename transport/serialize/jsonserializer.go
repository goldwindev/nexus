package serialize

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/goldwindev/nexus/wamp"
	"github.com/ugorji/go/codec"
)

var jh *codec.JsonHandle

func init() {
	jh = &codec.JsonHandle{}
	jh.MapType = reflect.TypeOf(map[string]interface{}(nil))
}

// JSONSerializer is an implementation of Serializer that handles
// serializing and deserializing json encoded payloads.
type JSONSerializer struct{}

// Serialize encodes a Message into a json payload.
func (s *JSONSerializer) Serialize(msg wamp.Message) ([]byte, error) {
	return json.Marshal(msgToList(msg))
}

// Deserialize decodes a json payload into a Message.
func (s *JSONSerializer) Deserialize(data []byte) (wamp.Message, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber() // Preserve number types

	var v []interface{}
	err := d.Decode(&v)
	if err != nil {
		return nil, err
	}

	convertedValue := wamp.ConvertTypeFromInterface(v, "json.Number", "intOrFloat64")
	v, ok := convertedValue.([]interface{})
	if !ok {
		return nil, errors.New("couldn't convert to []interface{}")
	}

	if len(v) == 0 {
		return nil, errors.New("invalid message")
	}

	// json deserializer gives us an float64 instead of an int64, whyever it
	// doesn't matter here, because valid values are only within an 8bit range.
	typ, ok := wamp.AsInt64(v[0])
	if !ok {
		return nil, errors.New("unsupported message format")
	}

	return listToMsg(wamp.MessageType(typ), v)
}

// Binary data follows a convention for conversion to JSON strings.
//
// A byte array is converted to a JSON string as follows:
//
// 1. convert the byte array to a Base64 encoded (host language) string
// 2. prepend the string with a \0 character
// 3. serialize the string to a JSON string
type BinaryData []byte

func (b BinaryData) MarshalJSON() ([]byte, error) {
	s := base64.StdEncoding.EncodeToString([]byte(b))
	var out []byte
	return out, codec.NewEncoderBytes(&out, jh).Encode("\x00" + s)
}

func (b *BinaryData) UnmarshalJSON(v []byte) error {
	var s string
	err := codec.NewDecoderBytes(v, jh).Decode(&s)
	if err != nil {
		return nil
	}
	if s[0] != '\x00' {
		return errors.New("binary string does not start with NUL")
	}
	*b, err = base64.StdEncoding.DecodeString(s[1:])
	return err
}

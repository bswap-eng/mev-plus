package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"
)

const defaultWriteTimeout = 10 * time.Second

var null = json.RawMessage("null")

// parseMessage parses raw bytes as a (batch of) JSON-RPC message(s). There are no error
// checks in this function because the raw message has already been syntax-checked when it
// is called. Any non-JSON-RPC messages in the input return the zero value of
// JsonRPCMessage.
func parseMessage(raw json.RawMessage) (*JsonRPCMessage, bool) {
		msg := JsonRPCMessage{}
		json.Unmarshal(raw, &msg)
		return &msg, false
}

// parsePositionalArguments tries to parse the given args to an array of values with the
// given types. It returns the parsed values or an error when the args could not be
// parsed. Missing optional arguments are returned as reflect.Zero values.
func parsePositionalArguments(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	var args []reflect.Value
	tok, err := dec.Token()
	switch {
	case err == io.EOF || tok == nil && err == nil:
		// "params" is optional and may be empty. Also allow "params":null even though it's
		// not in the spec because our own client used to send it.
	case err != nil:
		return nil, err
	case tok == json.Delim('['):
		// Read argument array.
		if args, err = parseArgumentArray(dec, types); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("non-array args")
	}
	// Set any missing args to nil.
	for i := len(args); i < len(types); i++ {
		if types[i].Kind() != reflect.Ptr {
			return nil, fmt.Errorf("missing value for required argument %d", i)
		}
		args = append(args, reflect.Zero(types[i]))
	}
	return args, nil
}

func parseArgumentArray(dec *json.Decoder, types []reflect.Type) ([]reflect.Value, error) {
	args := make([]reflect.Value, 0, len(types))
	for i := 0; dec.More(); i++ {
		if i >= len(types) {
			return args, fmt.Errorf("too many arguments, want at most %d", len(types))
		}
		argval := reflect.New(types[i])
		if err := dec.Decode(argval.Interface()); err != nil {
			return args, fmt.Errorf("invalid argument %d: %v", i, err)
		}
		if argval.IsNil() && types[i].Kind() != reflect.Ptr {
			return args, fmt.Errorf("missing value for required argument %d", i)
		}
		args = append(args, argval.Elem())
	}
	// Read end of args array.
	_, err := dec.Token()
	return args, err
}

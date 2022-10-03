package remote

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func (rc *RemoteConfig) Decode(in any) error {
	val := reflect.ValueOf(in)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("value passed needs to be a pointer")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("value passed needs to be a struct")
	}
	return rc.doDecode(val, "", 0)
}

func (rc *RemoteConfig) doDecode(val reflect.Value, group string, depth int) error {
	for i := 0; i < val.NumField(); i++ {
		fieldType := val.Type().Field(i)
		isGroup := false
		key := fieldType.Name
		if tag, ok := fieldType.Tag.Lookup("config"); ok {
			s := strings.Split(tag, ",")
			if len(s) > 1 && s[1] == "group" {
				key = s[0]
				isGroup = true
			} else {
				sn := strings.SplitN(s[0], ".", 2)
				if len(sn) > 1 {
					group = sn[0]
					key = sn[1]
				} else {
					key = sn[0]
				}
			}
		}
		field := val.Field(i)
		if isGroup {
			if field.Kind() != reflect.Struct {
				return fmt.Errorf("to decode a group, the field must be a struct")
			}
			if depth == 100 {
				return fmt.Errorf("structure too deep")
			}
			err := rc.doDecode(field, key, depth+1)
			if err != nil {
				return err
			}
			continue
		}
		value, vType, err := rc.getValue(group, key)
		if err != nil {
			return err
		}
		if value == "" {
			continue
		}
		switch vType {
		case ValueTypeString:
			if field.Kind() != reflect.String {
				return expectType("string", fieldType)
			}
			field.SetString(value)
		case ValueTypeBoolean:
			if field.Kind() != reflect.Bool {
				return expectType("bool", fieldType)
			}
			if value == "true" {
				field.SetBool(true)
			} else if value != "false" {
				return invalidConfigType(group, key, value, vType)
			}
		case ValueTypeNumber:
			switch field.Kind() {
			case reflect.Int,
				reflect.Int8,
				reflect.Int16,
				reflect.Int32,
				reflect.Int64,
				reflect.Uint,
				reflect.Uint8,
				reflect.Uint16,
				reflect.Uint32,
				reflect.Uint64,
				reflect.Float32,
				reflect.Float64:
				break
			default:
				return expectType("uint/int/float", fieldType)
			}
			u, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				i, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					f, err := strconv.ParseFloat(value, 64)
					if err != nil {
						return invalidConfigType(group, key, value, vType)
					}
					field.SetFloat(f)
					break
				}
				field.SetInt(i)
				break
			}
			field.SetUint(u)
		case ValueTypeJSON:
			err := json.Unmarshal([]byte(value), (interface{})(field.Pointer())) // TODO: Test
			if err != nil {
				// TODO: better error handling
				return invalidConfigType(group, key, value, vType)
			}
		default:
			return unspecifiedType(group, key)
		}
	}
	return nil
}

func (rc *RemoteConfig) getValue(group, key string) (string, ValueType, error) {
	para := rc.Parameters
	if group != "" {
		g, ok := rc.ParameterGroups[group]
		if !ok {
			return "", "", fmt.Errorf("group not found")
		}
		para = g.Parameters
	}
	p, ok := para[key]
	if ok {
		return p.Value().GetValue(), p.ValueType, nil
	}
	return "", "", fmt.Errorf("parameter not found")
}

func expectType(expect string, field reflect.StructField) error {
	return fmt.Errorf("expected field '%s' to be a %s, got %s", field.Name, expect, field.Type)
}

func invalidConfigType(group, key, value string, vType ValueType) error {
	if group != "" {
		group += "."
	}
	return fmt.Errorf("config parameter %s%s has invalid value. expected a value of type %s, got '%s'", group, key, vType, value)
}

func unspecifiedType(group, key string) error {
	if group != "" {
		group += "."
	}
	return fmt.Errorf("config parameter %s%s has an unspecified type", group, key)
}

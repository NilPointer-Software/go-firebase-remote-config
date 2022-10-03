package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	ordered "github.com/wk8/go-ordered-map/v2"
	"io"
	"time"
)

type RemoteConfig struct {
	client          *RemoteConfigClient
	ETag            string                    `json:"-"`
	Version         Version                   `json:"version"`
	Conditions      []Condition               `json:"conditions"`
	Parameters      map[string]Parameter      `json:"parameters"`
	ParameterGroups map[string]ParameterGroup `json:"parameterGroups"`
}

func (rc *RemoteConfig) Refresh() error {
	if rc.client != nil {
		var err error
		newConfig, err := rc.client.Get()
		if err != nil {
			return err
		}
		*rc = *newConfig
		return nil
	}
	return fmt.Errorf("no client attached")
}

func (rc *RemoteConfig) Update() error {
	if rc.client != nil {
		newConfig, err := rc.client.Update(rc)
		if err != nil {
			return err
		}
		*rc = *newConfig
		return nil
	}
	return fmt.Errorf("no client attached")
}

type Version struct {
	IsLegacy       bool         `json:"isLegacy"`
	VersionNumber  int64        `json:"versionNumber,string"`
	UpdateTime     time.Time    `json:"updateTime"`
	UpdateUser     User         `json:"updateUser"`
	Description    string       `json:"description"`
	UpdateOrigin   UpdateOrigin `json:"updateOrigin"`
	UpdateType     UpdateType   `json:"updateType"`
	RollbackSource int64        `json:"rollbackSource,string"`
}

type User struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	ImageURL string `json:"imageUrl"`
}

type UpdateOrigin string

const (
	UpdateOriginUnspecified  UpdateOrigin = "REMOTE_CONFIG_UPDATE_ORIGIN_UNSPECIFIED"
	UpdateOriginConsole      UpdateOrigin = "CONSOLE"
	UpdateOriginRESTAPI      UpdateOrigin = "REST_API"
	UpdateOriginAdminSDKNode UpdateOrigin = "ADMIN_SDK_NODE"
)

type UpdateType string

const (
	UpdateTypeUnspecified UpdateType = "REMOTE_CONFIG_UPDATE_TYPE_UNSPECIFIED"
	UpdateTypeIncremental UpdateType = "INCREMENTAL_UPDATE"
	UpdateTypeForced      UpdateType = "FORCED_UPDATE"
	UpdateTypeRollback    UpdateType = "ROLLBACK"
)

type Condition struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
	TagColor   string `json:"tagColor"`
}

type Parameter struct {
	Description       string             `json:"description"`
	ValueType         ValueType          `json:"valueType"`
	DefaultValue      *ParameterValue    `json:"defaultValue"`
	ConditionalValues *ConditionalValues `json:"conditionalValues"`
}

func (p *Parameter) Value() ParameterValue {
	if p.DefaultValue != nil {
		return *p.DefaultValue
	}
	if p.ConditionalValues != nil {
		// TODO: Process conditionals
		return p.ConditionalValues.Inner().Oldest().Value
	}
	panic("parameter has no value")
}

type ValueType string

const (
	ValueTypeUnspecified ValueType = "PARAMETER_VALUE_TYPE_UNSPECIFIED"
	ValueTypeString      ValueType = "STRING"
	ValueTypeBoolean     ValueType = "BOOLEAN"
	ValueTypeNumber      ValueType = "NUMBER"
	ValueTypeJSON        ValueType = "JSON"
)

type ParameterValue struct {
	Value                *string               `json:"value"`
	UseInAppDefault      *bool                 `json:"useInAppDefault"`
	PersonalizationValue *PersonalizationValue `json:"personalizationValue"`
}

func (dv ParameterValue) GetValue() string {
	if dv.Value != nil {
		return *dv.Value
	}
	return ""
}

type PersonalizationValue struct {
	PersonalizationId string `json:"personalizationId"`
}

type ConditionalValues ordered.OrderedMap[string, ParameterValue]

func (cv *ConditionalValues) Inner() *ordered.OrderedMap[string, ParameterValue] {
	return (*ordered.OrderedMap[string, ParameterValue])(cv)
}

func (cv *ConditionalValues) MarshalJSON() (res []byte, err error) {
	res = append(res, '{')
	for pair := cv.Inner().Oldest(); pair != nil; pair = pair.Next() {
		res = append(res, fmt.Sprintf("\"%s\":", pair.Key)...)
		var data []byte
		data, err = json.Marshal(pair.Value)
		if err != nil {
			return nil, err
		}
		res = append(res, data...)
		res = append(res, ',')
	}
	if len(res) > 1 {
		res = append(res[:len(res)-1], '}')
	} else {
		res = append(res, '}')
	}
	return
}

func (cv *ConditionalValues) UnmarshalJSON(data []byte) error {
	m := ordered.New[string, ParameterValue]()
	dec := json.NewDecoder(bytes.NewReader(data))

	token, err := dec.Token()
	if err != nil {
		return err
	} else if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected object, got %s", delim)
	}

	for dec.More() {
		token, err = dec.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T: %v", token, token)
		}

		var value ParameterValue
		err = dec.Decode(&value)
		if err != nil {
			return err
		}
		m.Set(key, value)
	}

	token, err = dec.Token()
	if err != nil {
		return err
	} else if delim, ok := token.(json.Delim); !ok || delim != '}' {
		return fmt.Errorf("expected end of object, got %T %v %v", token, token, err)
	}
	token, err = dec.Token()
	if err != io.EOF {
		return fmt.Errorf("expected EOF, got %T %v %v", token, token, err)
	}
	*cv = (ConditionalValues)(*m)
	return nil
}

type ParameterGroup struct {
	Description string               `json:"description"`
	Parameters  map[string]Parameter `json:"parameters"`
}

type Rollback struct {
	VersionNumber int64 `json:"versionNumber,string"`
}

type VersionPage struct {
	Versions      []Version `json:"versions"`
	NextPageToken string    `json:"nextPageToken"`
}

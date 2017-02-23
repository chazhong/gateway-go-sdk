package deviot

type PropertyType int

const (
	PROPERTY_TYPE_NUMBER PropertyType = 0
	PROPERTY_TYPE_STRING PropertyType = 1
	PROPERTY_TYPE_BOOL   PropertyType = 2
	PROPERTY_TYPE_COLOR  PropertyType = 3
)

type Thing struct {
	Id          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Kind        string          `json:"kind"`
	Properties  []ThingProperty `json:"properties,omitempty"`
	Actions     []ThingAction   `json:"actions,omitempty"`
}

type ThingProperty struct {
	Name  string       `json:"name"`
	Type  PropertyType `json:"type"`
	Value interface{}  `json:"value,omitempty"`
}

type ThingAction struct {
	Name       string          `json:"name"`
	Parameters []ThingProperty `json:"parameters,omitempty"`
}

func (thing Thing) FindAction(action string) ThingAction {
	for _, a := range thing.Actions {
		if a.Name == action {
			return a
		}
	}
	return ThingAction{}
}

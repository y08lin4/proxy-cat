package protocol

type FieldDef struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "string","int","bool","[]string","map","enum"
	Required    bool     `json:"required"`
	Description string   `json:"description"`
	Default     any      `json:"default,omitempty"`
	EnumValues  []string `json:"enumValues,omitempty"`
}

type ProtocolDef struct {
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Transport   []string   `json:"transport"`
	Fields      []FieldDef `json:"fields"`
}

type Registry struct {
	protocols map[string]ProtocolDef
	ordered   []string
}

func New() *Registry {
	return &Registry{protocols: make(map[string]ProtocolDef)}
}

func (r *Registry) Register(def ProtocolDef) {
	r.protocols[def.Type] = def
	r.ordered = append(r.ordered, def.Type)
}

func (r *Registry) All() []ProtocolDef {
	result := make([]ProtocolDef, 0, len(r.ordered))
	for _, typ := range r.ordered {
		result = append(result, r.protocols[typ])
	}
	return result
}

func (r *Registry) Get(typ string) (ProtocolDef, bool) {
	def, ok := r.protocols[typ]
	return def, ok
}

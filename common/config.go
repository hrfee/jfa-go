package common

type SectionMeta struct {
	Name         string `json:"name" yaml:"name" example:"My Section"` // friendly name of the section
	Description  string `json:"description" yaml:"description"`
	Advanced     bool   `json:"advanced,omitempty" yaml:"advanced,omitempty"`
	Disabled     bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`
	DependsTrue  string `json:"depends_true,omitempty" yaml:"depends_true,omitempty"`
	DependsFalse string `json:"depends_false,omitempty" yaml:"depends_false,omitempty"`
	WikiLink     string `json:"wiki_link,omitempty" yaml:"wiki_link,omitempty"`
}

type Option [2]string

type SettingType string

var (
	BoolType     SettingType = "bool"
	SelectType   SettingType = "select"
	TextType     SettingType = "text"
	PasswordType SettingType = "password"
	NumberType   SettingType = "number"
	NoteType     SettingType = "note"
	EmailType    SettingType = "email"
	ListType     SettingType = "list"
)

type Setting struct {
	Setting         string      `json:"setting" yaml:"setting" example:"my_setting"`
	Name            string      `json:"name" yaml:"name" example:"My Setting"`
	Description     string      `json:"description" yaml:"description"`
	Required        bool        `json:"required" yaml:"required"`
	RequiresRestart bool        `json:"requires_restart" yaml:"requires_restart"`
	Advanced        bool        `json:"advanced,omitempty" yaml:"advanced,omitempty"`
	Type            SettingType `json:"type" yaml:"type"` // Type (string, number, bool, etc.)
	Value           any         `json:"value" yaml:"value"`
	Options         []Option    `json:"options,omitempty" yaml:"options,omitempty"`
	DependsTrue     string      `json:"depends_true,omitempty" yaml:"depends_true,omitempty"`   // If specified, this field is enabled when the specified bool setting is enabled.
	DependsFalse    string      `json:"depends_false,omitempty" yaml:"depends_false,omitempty"` // If specified, opposite behaviour of DependsTrue.
	Style           string      `json:"style,omitempty" yaml:"style,omitempty"`
	Deprecated      bool        `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	WikiLink        string      `json:"wiki_link,omitempty" yaml:"wiki_link,omitempty"`
}

type Section struct {
	Section  string      `json:"section" yaml:"section" example:"my_section"`
	Meta     SectionMeta `json:"meta" yaml:"meta"`
	Settings []Setting   `json:"settings" yaml:"settings"`
}

type Config struct {
	Sections []Section `json:"sections" yaml:"sections"`
}

func (c *Config) removeSection(section string) {
	for i, v := range c.Sections {
		if v.Section == section {
			c.Sections = append(c.Sections[:i], c.Sections[i+1:]...)
			break
		}
	}
}

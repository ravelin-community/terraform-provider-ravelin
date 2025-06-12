package gsudo

type Config struct {
	Escalations map[string][]string `yaml:"escalations"` // list of escalation roles per project
	Inherit     bool                `yaml:"inherit"`     // whether the roles are inherited from a group
}

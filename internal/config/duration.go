package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Millis ist eine Dauer, die in der Konfiguration als ganzzahlige Millisekunden steht.
type Millis time.Duration

func (m *Millis) UnmarshalYAML(node *yaml.Node) error {
	var ms int64
	if err := node.Decode(&ms); err != nil {
		return fmt.Errorf("ganzzahlige millisekunden erwartet: %w", err)
	}
	*m = Millis(time.Duration(ms) * time.Millisecond)
	return nil
}

func (m Millis) MarshalYAML() (any, error) {
	return time.Duration(m).Milliseconds(), nil
}

func (m Millis) Duration() time.Duration { return time.Duration(m) }

func (m Millis) String() string { return time.Duration(m).String() }

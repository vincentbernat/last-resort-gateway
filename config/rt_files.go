package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Table is a routing table. It is an uint between 0 and 255 but can
// be parsed and rendered as a string using
// `/etc/iproute2/rt_tables`. No caching is done.
type Table struct {
	ID   uint
	Name string
}

func (t Table) String() string {
	if t.Name != "" {
		return t.Name
	}
	return fmt.Sprintf("%d", t.ID)
}

// Protocol is a routing protocol. It is an uint between 0 and 255 but
// can be parsed and rendered as a string using
// `/etc/iproute2/rt_protos`. No caching is done.
type Protocol struct {
	ID   uint
	Name string
}

func (p Protocol) String() string {
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("%d", p.ID)
}

// UnmarshalText parses a table name
func (t *Table) UnmarshalText(text []byte) error {
	name := string(text)
	id, err := findNameRTFiles([]string{
		"/etc/iproute2/rt_tables",
		"/etc/iproute2/rt_tables.d/*.conf",
	}, name)
	if err != nil {
		return errors.Wrapf(err, "unable to lookup table %q", name)
	}
	*t = Table{
		ID:   id,
		Name: name,
	}
	return nil
}

// UnmarshalText parses a protocol name
func (p *Protocol) UnmarshalText(text []byte) error {
	name := string(text)
	id, err := findNameRTFiles([]string{
		"/etc/iproute2/rt_protos",
		"/etc/iproute2/rt_protos.d/*.conf",
	}, name)
	if err != nil {
		return errors.Wrapf(err, "unable to lookup protocol %q", name)
	}
	*p = Protocol{
		ID:   id,
		Name: name,
	}
	return nil
}

// UnmarshalYAML parses a table from YAML either as an integer or a name.
func (t *Table) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rawUint uint
	if err := unmarshal(&rawUint); err == nil {
		if rawUint > 255 {
			return errors.Errorf("table ID %d is out of range", rawUint)
		}
		*t = Table{ID: rawUint}
		return nil
	}
	var rawString string
	if err := unmarshal(&rawString); err != nil {
		return err
	}
	return t.UnmarshalText([]byte(rawString))
}

// UnmarshalYAML parses a protocol from YAML either as an integer or a name.
func (p *Protocol) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rawUint uint
	if err := unmarshal(&rawUint); err == nil {
		if rawUint > 255 {
			return errors.Errorf("protocol ID %d is out of range", rawUint)
		}
		*p = Protocol{ID: rawUint}
		return nil
	}
	var rawString string
	if err := unmarshal(&rawString); err != nil {
		return err
	}
	return p.UnmarshalText([]byte(rawString))
}

// findNameRTFiles search for the given name in provided RT files and
// return the corresponding ID (between 0 and 255). An error is
// returned if not found or if ID is out of range.
func findNameRTFiles(paths []string, name string) (uint, error) {
	for _, p := range paths {
		files, err := filepath.Glob(p)
		if err != nil {
			return 0, errors.Wrapf(err, "unable to expand %q", p)
		}
		for _, f := range files {
			content, err := ioutil.ReadFile(f)
			if err != nil {
				return 0, errors.Wrapf(err, "unable to read %q", f)
			}
			lines := strings.Split(string(content), "\n")
			for _, l := range lines {
				var id uint
				var table string
				fmt.Sscanf(l, "%d %s", &id, &table)
				if table == name {
					if id > 255 {
						return 0, errors.Errorf("ID %d out of range", id)
					}
					return id, nil
				}
			}
		}
	}
	return 0, errors.Errorf("name %q not found", name)
}

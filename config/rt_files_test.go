package config

import (
	"testing"

	"gopkg.in/yaml.v2"

	"lrg/helpers"
)

func TestUnmarshalTable(t *testing.T) {
	cases := []struct {
		input string
		want  Table
		err   bool
	}{
		{"254", Table{ID: 254}, false},
		{"260", Table{}, true},
		{"-10", Table{}, true},
		{"main", Table{ID: 254, Name: "main"}, false},
		{"public", Table{ID: 90, Name: "public"}, false},
		{"inr.ruhep", Table{}, true},
		{"unknown", Table{}, true},
	}
	for _, tc := range cases {
		var got Table
		err := yaml.Unmarshal([]byte(tc.input), &got)
		switch {
		case err != nil && !tc.err:
			t.Errorf("Unmarshal(%q) error:\n%+v", tc.input, err)
		case err == nil && tc.err:
			t.Errorf("Unmarshal(%q) == %q but expected error", tc.input, got)
		default:
			if diff := helpers.Diff(got, tc.want); diff != "" {
				t.Errorf("Unmarshal(%q) (-got, +want):\n%s", tc.input, diff)
			}
		}
	}
}

func TestUnmarshalProtocol(t *testing.T) {
	cases := []struct {
		input string
		want  Protocol
		err   bool
	}{
		{"254", Protocol{ID: 254}, false},
		{"260", Protocol{}, true},
		{"-10", Protocol{}, true},
		{"lrg", Protocol{ID: 254, Name: "lrg"}, false},
		{"kernel", Protocol{ID: 2, Name: "kernel"}, false},
		{"unknown", Protocol{}, true},
	}
	for _, tc := range cases {
		var got Protocol
		err := yaml.Unmarshal([]byte(tc.input), &got)
		switch {
		case err != nil && !tc.err:
			t.Errorf("Unmarshal(%q) error:\n%+v", tc.input, err)
		case err == nil && tc.err:
			t.Errorf("Unmarshal(%q) == %q but expected error", tc.input, got)
		default:
			if diff := helpers.Diff(got, tc.want); diff != "" {
				t.Errorf("Unmarshal(%q) (-got, +want):\n%s", tc.input, diff)
			}
		}
	}
}

func TestFindNameRTFiles(t *testing.T) {
	cases := []struct {
		descr string
		paths []string
		name  string
		want  uint
		err   bool
	}{
		{
			"simple file lookup",
			[]string{"testdata/rt_protos"},
			"kernel",
			2,
			false,
		}, {
			"simple file failed lookup",
			[]string{"testdata/rt_protos"},
			"nothing",
			0,
			true,
		}, {
			"no files",
			[]string{},
			"kernel",
			0,
			true,
		}, {
			"nonexistent file",
			[]string{"testdata/rt_protos2"},
			"kernel",
			0,
			true,
		}, {
			"empty glob",
			[]string{"testdata/nothing*"},
			"kernel",
			0,
			true,
		}, {
			"empty glob and simple file lookup",
			[]string{"testdata/nothing", "testdata/rt_protos"},
			"kernel",
			2,
			false,
		}, {
			"commented line",
			[]string{"testdata/rt_protos"},
			"commented",
			57,
			false,
		}, {
			"failed lookup with globbing",
			[]string{"testdata/rt_protos", "testdata/rt_protos.d/*.conf"},
			"nothing",
			0,
			true,
		}, {
			"out of range lookup",
			[]string{"testdata/rt_protos", "testdata/rt_protos.d/*.conf"},
			"out-of-range",
			0,
			true,
		}, {
			"duplicate entry",
			[]string{"testdata/rt_protos", "testdata/rt_protos.d/*.conf"},
			"mrt",
			10,
			false,
		}, {
			"non-matching glob",
			[]string{"testdata/rt_protos", "testdata/rt_protos.d/*.conf"},
			"invalid",
			0,
			true,
		}, {
			"matching glob",
			[]string{"testdata/rt_protos", "testdata/rt_protos.d/*.conf"},
			"more",
			100,
			false,
		},
	}
	for _, tc := range cases {
		got, err := findNameRTFiles(tc.paths, tc.name)
		switch {
		case err != nil && !tc.err:
			t.Errorf("findNameRTFiles(%q) error:\n%+v", tc.descr, err)
		case err == nil && tc.err:
			t.Errorf("findNameRTFiles(%q) == %d but expected error", tc.descr, got)
		case err == nil && tc.want != got:
			t.Errorf("findNameRTFiles(%q) == %d but expected %d", tc.descr, got, tc.want)
		}
	}
}

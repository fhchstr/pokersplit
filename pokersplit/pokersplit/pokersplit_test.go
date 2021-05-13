package pokersplit

import (
	"testing"

	"github.com/fhchstr/pokersplit/pokersplit/players"
	"github.com/google/go-cmp/cmp"
)

func TestSorted(t *testing.T) {
	cases := []struct {
		desc  string
		input players.Players
		want  players.Players
	}{
		{
			desc:  "all_lowercase",
			input: players.Players{{Name: "bob"}, {Name: "alice"}, {Name: "dan"}, {Name: "charlie"}},
			want:  players.Players{{Name: "alice"}, {Name: "bob"}, {Name: "charlie"}, {Name: "dan"}},
		},
		{
			desc:  "mixed_case",
			input: players.Players{{Name: "Bob"}, {Name: "alice"}, {Name: "Dan"}, {Name: "charlie"}},
			want:  players.Players{{Name: "alice"}, {Name: "Bob"}, {Name: "charlie"}, {Name: "Dan"}},
		},
		{
			desc:  "special_characters",
			input: players.Players{{Name: "bob"}, {Name: "àlice"}, {Name: "ḓan"}, {Name: "ĉharlie"}},
			want:  players.Players{{Name: "àlice"}, {Name: "bob"}, {Name: "ĉharlie"}, {Name: "ḓan"}},
		},
		{
			desc:  "special_and_mixed_case",
			input: players.Players{{Name: "bob"}, {Name: "Âlice"}, {Name: "ḓan"}, {Name: "Çharlie"}},
			want:  players.Players{{Name: "Âlice"}, {Name: "bob"}, {Name: "Çharlie"}, {Name: "ḓan"}},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := sorted(c.input)
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("sorted() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

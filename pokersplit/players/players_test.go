package players

import (
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var sortPlayer = cmpopts.SortSlices(func(a, b *Player) bool { return a.Name < b.Name })
var sortDebt = cmpopts.SortSlices(func(a, b Debt) bool { return a.Creditor < b.Creditor })

// TestBase64 tests the base64 encoding and decoding functions.
func TestBase64(t *testing.T) {
	cases := []struct {
		name    string
		players Players
	}{
		{
			name: "empty",
		},
		{
			name:    "one_player_name_only",
			players: Players{{Name: "Alice"}},
		},
		{
			name:    "one_player_all_fields",
			players: Players{{Name: "Alice", BuyIn: 100, Stack: 8575}},
		},
		{
			name:    "two_players_names_only",
			players: []*Player{{Name: "Alice"}, {Name: "Bob"}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b64, err := c.players.ToBase64()
			if err != nil {
				t.Fatalf("Players.ToBase64() returned an error: %v", err)
			}

			if _, err := base64.URLEncoding.DecodeString(b64); err != nil {
				t.Fatalf("Players.ToBase64() returned %q, which isn't a valid base64-encoded string: %v", b64, err)
			}

			got, err := FromBase64(b64)
			if err != nil {
				t.Fatalf("FromBase64() return an error: %v", err)
			}

			if diff := cmp.Diff(c.players, got); diff != "" {
				t.Errorf("base64 encoding/decoding mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFromForm(t *testing.T) {
	cases := []struct {
		desc    string
		form    url.Values
		want    Players
		wantErr bool
	}{
		{
			desc: "no_players",
		},
		{
			desc: "one_player",
			form: url.Values{
				"player0": []string{"alice"},
				"buyin0":  []string{"50.25"},
				"stack0":  []string{"123.40"},
			},
			want: Players{{Name: "alice", BuyIn: 5025, Stack: 12340}},
		},
		{
			desc: "one_player_and_other_irrelevant_fields",
			form: url.Values{
				"player0": []string{"alice"},
				"buyin0":  []string{"50.25"},
				"stack0":  []string{"123.40"},
				"buyin1":  []string{"1000"},
				"stack1":  []string{"2000"},
				"buyin2":  []string{"3000"},
				"stack3":  []string{"4000"},
			},
			want: Players{{Name: "alice", BuyIn: 5025, Stack: 12340}},
		},
		{
			desc: "one_player_non_zero_index",
			form: url.Values{
				"player2": []string{"alice"},
				"buyin2":  []string{"50.25"},
				"stack2":  []string{"123.40"},
			},
			want: Players{{Name: "alice", BuyIn: 5025, Stack: 12340}},
		},
		{
			desc: "two_players",
			form: url.Values{
				"player0": []string{"alice"},
				"buyin0":  []string{"50.25"},
				"stack0":  []string{"123.40"},
				"player1": []string{"bob"},
				"buyin1":  []string{"22.75"},
				"stack1":  []string{"15"},
			},
			want: Players{
				{Name: "alice", BuyIn: 5025, Stack: 12340},
				{Name: "bob", BuyIn: 2275, Stack: 1500},
			},
		},
		{
			desc: "emtpy_name",
			form: url.Values{
				"player0": []string{""},
				"buyin0":  []string{"222"},
				"stack0":  []string{"111"},
			},
			want: Players{},
		},
		{
			desc: "name_with_multiple_values",
			form: url.Values{
				"player0": []string{"b", "o", "b"},
				"buyin0":  []string{"222"},
				"stack0":  []string{"111"},
			},
			want: Players{},
		},
		{
			desc: "missing_buyin",
			form: url.Values{
				"player0": []string{"alice"},
				"stack0":  []string{"123.40"},
			},
			want: Players{{Name: "alice", Stack: 12340}},
		},
		{
			desc: "missing_stack",
			form: url.Values{
				"player0": []string{"alice"},
				"buyin0":  []string{"50.25"},
			},
			want: Players{{Name: "alice", BuyIn: 5025}},
		},
		{
			desc: "duplicate_name",
			form: url.Values{
				"player0": []string{"alice"},
				"buyin0":  []string{"222"},
				"stack0":  []string{"111"},
				"player1": []string{"alice"},
				"buyin1":  []string{"100"},
				"stack1":  []string{"200"},
			},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got, err := FromForm(c.form)
			if err != nil && !c.wantErr {
				t.Fatalf("FromForm() returned an error: %v", err)
			}
			if err == nil && c.wantErr {
				t.Fatalf("FromForm() didn't return an error, but one was expected")
			}
			if c.wantErr {
				return
			}
			if diff := cmp.Diff(c.want, got, sortPlayer, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FromForm() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCalculateDebts(t *testing.T) {
	cases := []struct {
		desc    string
		players Players
		want    Debts
		wantErr bool
	}{
		{
			desc:    "single_player",
			players: Players{{Name: "alice", BuyIn: 500, Stack: 500}},
		},
		{
			desc: "all_players_equality",
			players: Players{
				{Name: "alice", BuyIn: 500, Stack: 500},
				{Name: "bob", BuyIn: 1500, Stack: 1500},
				{Name: "charlie", BuyIn: 2000, Stack: 2000},
			},
		},
		{
			desc: "two_players",
			players: Players{
				{Name: "alice", BuyIn: 500, Stack: 1000},
				{Name: "bob", BuyIn: 500, Stack: 0},
			},
			want: Debts{
				"bob": []Debt{{Creditor: "alice", Amount: 500}},
			},
		},
		{
			desc: "two_loosers",
			players: Players{
				{Name: "alice", BuyIn: 500, Stack: 1200},
				{Name: "bob", BuyIn: 500, Stack: 0},
				{Name: "charlie", BuyIn: 1000, Stack: 800},
			},
			want: Debts{
				"bob":     []Debt{{Creditor: "alice", Amount: 500}},
				"charlie": []Debt{{Creditor: "alice", Amount: 200}},
			},
		},
		{
			desc: "two_winners",
			players: Players{
				{Name: "alice", BuyIn: 1000, Stack: 1100},
				{Name: "bob", BuyIn: 500, Stack: 900},
				{Name: "charlie", BuyIn: 1000, Stack: 500},
			},
			want: Debts{
				"charlie": []Debt{
					{Creditor: "alice", Amount: 100},
					{Creditor: "bob", Amount: 400},
				},
			},
		},
		{
			desc: "two_winners_three_loosers",
			players: Players{
				{Name: "alice", BuyIn: 1000, Stack: 1350},
				{Name: "bob", BuyIn: 500, Stack: 1050},
				{Name: "charlie", BuyIn: 1000, Stack: 500},
				{Name: "dan", BuyIn: 300, Stack: 0},
				{Name: "eve", BuyIn: 700, Stack: 600},
			},
			want: Debts{
				"charlie": []Debt{
					{Creditor: "alice", Amount: 350},
					{Creditor: "bob", Amount: 150},
				},
				"dan": []Debt{
					{Creditor: "bob", Amount: 300},
				},
				"eve": []Debt{
					{Creditor: "bob", Amount: 100},
				},
			},
		},
		{
			desc: "buy_in_and_stack_mismatch",
			players: Players{
				{Name: "alice", BuyIn: 500, Stack: 1200},
				{Name: "bob", BuyIn: 1500, Stack: 500},
				{Name: "charlie", BuyIn: 1000, Stack: 2000},
			},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got, err := c.players.CalculateDebts()
			if err != nil && !c.wantErr {
				t.Fatalf("Players.CalculateDebts() returned an error: %v", err)

			}
			if err == nil && c.wantErr {
				t.Fatalf("Players.CalculateDebts() didn't return an error, but one was expected")
			}
			if c.wantErr {
				return
			}
			if diff := cmp.Diff(c.want, got, cmpopts.EquateEmpty(), sortDebt); diff != "" {
				t.Errorf("Players.CalculateDebts() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWinnersAndLoosers(t *testing.T) {
	cases := []struct {
		desc        string
		players     Players
		wantWinners Players
		wantLoosers Players
	}{
		{
			desc: "no_players",
		},
		{
			desc:    "one_player_at_zero",
			players: Players{{Name: "alice", BuyIn: 1500, Stack: 1500}},
			wantWinners: Players{
				{Name: "alice", BuyIn: 1500, Stack: 1500},
			},
		},
		{
			desc: "winners",
			players: Players{
				{Name: "alice", BuyIn: 1500, Stack: 3000},
				{Name: "bob", BuyIn: 250, Stack: 1000},
			},
			wantWinners: Players{
				{Name: "alice", BuyIn: 1500, Stack: 3000},
				{Name: "bob", BuyIn: 250, Stack: 1000},
			},
		},
		{
			desc: "loosers",
			players: Players{
				{Name: "alice", BuyIn: 500, Stack: 100},
				{Name: "bob", BuyIn: 8000, Stack: 4000},
			},
			wantLoosers: Players{
				{Name: "alice", BuyIn: 500, Stack: 100},
				{Name: "bob", BuyIn: 8000, Stack: 4000},
			},
		},
		{
			desc: "winners_and_loosers",
			players: Players{
				{Name: "alice", BuyIn: 500, Stack: 100},
				{Name: "bob", BuyIn: 8000, Stack: 4000},
				{Name: "charlie", BuyIn: 5000, Stack: 6000},
			},
			wantWinners: Players{
				{Name: "charlie", BuyIn: 5000, Stack: 6000},
			},
			wantLoosers: Players{
				{Name: "alice", BuyIn: 500, Stack: 100},
				{Name: "bob", BuyIn: 8000, Stack: 4000},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			gotWinners, gotLoosers := c.players.winnersAndLoosers()
			if diff := cmp.Diff(c.wantWinners, gotWinners, sortPlayer); diff != "" {
				t.Errorf("Players.winnersAndLoosers() winners mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(c.wantLoosers, gotLoosers, sortPlayer); diff != "" {
				t.Errorf("Players.winnersAndLoosers() loosers mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

func TestPlayersBuyInAndStack(t *testing.T) {
	cases := []struct {
		desc      string
		players   Players
		wantBuyIn int
		wantStack int
	}{
		{
			desc:      "no_players",
			wantBuyIn: 0,
			wantStack: 0,
		},
		{
			desc:      "one_player",
			players:   Players{{Name: "alice", BuyIn: 1500, Stack: 3000}},
			wantBuyIn: 1500,
			wantStack: 3000,
		},
		{
			desc: "two_players",
			players: Players{
				{Name: "alice", BuyIn: 1500, Stack: 3000},
				{Name: "bob", BuyIn: 250, Stack: 1000}},
			wantBuyIn: 1750,
			wantStack: 4000,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			gotBuyIn := c.players.BuyIn()
			if gotBuyIn != c.wantBuyIn {
				t.Errorf("Players.BuyIn() = %d, want %d", gotBuyIn, c.wantBuyIn)
			}
			gotStack := c.players.Stack()
			if gotBuyIn != c.wantBuyIn {
				t.Errorf("Players.Stack() = %d, want %d", gotStack, c.wantStack)
			}
		})
	}
}

func TestBest(t *testing.T) {
	cases := []struct {
		desc    string
		players Players
		want    *Player
	}{
		{
			desc: "no_players",
			want: nil,
		},
		// {
		// 	desc:    "one_player_at_balance",
		// 	players: Players{{Name: "alice", BuyIn: 1500, Stack: 1500}},
		// 	want:    nil,
		// },
		{
			desc:    "one_player",
			players: Players{{Name: "alice", BuyIn: 1500, Stack: 500}},
			want:    &Player{Name: "alice", BuyIn: 1500, Stack: 500},
		},
		{
			desc: "winners",
			players: Players{
				{Name: "alice", BuyIn: 1500, Stack: 3000},
				{Name: "bob", BuyIn: 500, Stack: 5000},
				{Name: "charlie", BuyIn: 5000, Stack: 6000},
			},
			want: &Player{Name: "bob", BuyIn: 500, Stack: 5000},
		},
		{
			desc: "loosers",
			players: Players{
				{Name: "alice", BuyIn: 500, Stack: 100},
				{Name: "bob", BuyIn: 8000, Stack: 4000},
				{Name: "charlie", BuyIn: 3000, Stack: 2500},
			},
			want: &Player{Name: "alice", BuyIn: 500, Stack: 100},
		},
		{
			desc: "winners_and_loosers",
			players: Players{
				{Name: "alice", BuyIn: 500, Stack: 100},
				{Name: "bob", BuyIn: 8000, Stack: 4000},
				{Name: "charlie", BuyIn: 5000, Stack: 6000},
			},
			want: &Player{Name: "charlie", BuyIn: 5000, Stack: 6000},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := c.players.best()
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("Players.best() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

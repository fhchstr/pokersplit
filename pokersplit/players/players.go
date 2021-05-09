// Package players implements functions to keep track of players' money and encode/decode them.
package players

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
)

// Player holds the player's data.
type Player struct {
	// Name of the player.
	Name string `json:"p"`
	// BuyIn is how much cash money the player invested, in cents.
	BuyIn int `json:"b,omitempty"`
	// Stack is how much money the player has, in cents.
	Stack int `json:"s,omitempty"`
}

// Players is a collection of Player.
type Players []*Player

// ToBase64 encodes Players in base64 URL-encoding. It can be decoded using FromBase64().
func (p Players) ToBase64() (string, error) {
	if len(p) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	encode := func() error {
		base64Encoder := base64.NewEncoder(base64.URLEncoding, &buf)
		defer base64Encoder.Close()
		gzipEncoder := gzip.NewWriter(base64Encoder)
		defer gzipEncoder.Close()
		jsonEncoder := json.NewEncoder(gzipEncoder)
		if err := jsonEncoder.Encode(p); err != nil {
			return err
		}
		return nil
	}
	if err := encode(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// FromBase64 decodes Players which were base64 URL-encoded using ToBase64().
func FromBase64(data string) (Players, error) {
	var ret Players
	if data == "" {
		return ret, nil
	}

	base64Decoder := base64.NewDecoder(base64.URLEncoding, strings.NewReader(data))
	gzipDecoder, err := gzip.NewReader(base64Decoder)
	if err != nil {
		return nil, fmt.Errorf("gzip decompression failed: %v", err)
	}
	jsonDecoder := json.NewDecoder(gzipDecoder)
	if err := jsonDecoder.Decode(&ret); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}
	return ret, nil
}

// FromForm creates Players from an HTML form's data. It expects the form to
// contain tuples in the form of fieldNameX, where fieldName is the name of
// the field: "player", "buyin" and "stack", and X is an ID, the same for all
// fields part of the same tuple.
func FromForm(form url.Values) (Players, error) {
	var ret Players
	// Keep track of players' names to detect duplicates.
	playerNames := make(map[string]bool)
	for k, v := range form {
		// Only consider the player fields. The other ones are infered using its ID.
		if !strings.HasPrefix(k, "player") {
			continue
		}
		if len(v) != 1 || strings.TrimSpace(v[0]) == "" {
			continue
		}
		name := v[0]
		if playerNames[name] {
			return nil, fmt.Errorf("duplicate player with name %q", name)
		}
		playerNames[name] = true

		i := strings.TrimPrefix(k, "player")
		cash, err := strconv.ParseFloat(form.Get("buyin"+i), 64)
		if err != nil {
			cash = 0
		}
		stack, err := strconv.ParseFloat(form.Get("stack"+i), 64)
		if err != nil {
			stack = 0
		}

		// The Player struct expects the amounts to be in cents. The ones in the
		// form aren't. Multiply them by 100 to convert them to cents.
		ret = append(ret, &Player{
			Name:  name,
			BuyIn: int(math.Round(cash * 100)),
			Stack: int(math.Round(stack * 100)),
		})
	}
	return ret, nil
}

// BuyIn returns the total amount of money invested by all Players, in cents.
func (p Players) BuyIn() int {
	ret := 0
	for _, player := range p {
		ret += player.BuyIn
	}
	return ret
}

// Stack returns the sum of all Players' stacks, in cents.
func (p Players) Stack() int {
	ret := 0
	for _, player := range p {
		ret += player.Stack
	}
	return ret
}

// Debt holds the debt's details.
type Debt struct {
	// Creditor is the name of the person to whom money is owed.
	Creditor string
	// Amount owed, in cents.
	Amount int
}

// Debts is a collection of debts, grouped by debtor.
type Debts map[string][]Debt

// CalculateDebts figures out who owes how much to whom. To limit the number of
// transactions, the best looser (the player who lost the least) owes monney to
// the best winner (the player who won the most). After each iteration, the
// balances are updated and the best winner/looser are re-identified.
func (p Players) CalculateDebts() (Debts, error) {
	if p.BuyIn() != p.Stack() {
		return nil, fmt.Errorf("the total of the buy-ins doesn't match the total of the stacks")
	}
	ret := make(Debts)
	winners, loosers := p.winnersAndLoosers()
	// The algorithm modifies the stacks to keep track of the debts already
	// taken into account. Once all winners have their stack equal to their
	// buy-in, it means that all debts are settled.
	for winners.BuyIn() != winners.Stack() {
		bLooser := loosers.best()
		bWinner := winners.best()
		amount := bLooser.BuyIn - bLooser.Stack
		if bWinner.Stack-bWinner.BuyIn < amount {
			amount = bWinner.Stack - bWinner.BuyIn
		}
		bLooser.Stack += amount
		bWinner.Stack -= amount
		debt := Debt{Creditor: bWinner.Name, Amount: amount}
		ret[bLooser.Name] = append(ret[bLooser.Name], debt)
	}
	return ret, nil
}

// winnersAndLoosers returns the players which won/lost money.
func (p Players) winnersAndLoosers() (winners, loosers Players) {
	for _, aPlayer := range p {
		// Make a copy of the player, because the algorithm modifies its stack
		// to settle the debts.
		player := &Player{
			Name:  aPlayer.Name,
			BuyIn: aPlayer.BuyIn,
			Stack: aPlayer.Stack,
		}
		gain := int(player.Stack - player.BuyIn)
		if gain >= 0 {
			winners = append(winners, player)
		} else {
			loosers = append(loosers, player)
		}
	}
	return
}

// best returns the Player who won the most, or lost the least, if they all lost.
// Players having a balance of zero are ignored.
func (p Players) best() *Player {
	best := -1
	for i := range p {
		// Ignore the players having a balance of zero, their debt is considered
		// settled. They must be ignored because otherwise they are returned
		// instead of the player who lost the least in case they all lost.
		if p[i].BuyIn == p[i].Stack {
			continue
		}
		// This is the first iteration of a player having a non-zero balance.
		// This is thus the best player seen so far.
		if best < 0 {
			best = i
			continue
		}
		if p[i].Stack-p[i].BuyIn > p[best].Stack-p[best].BuyIn {
			best = i
		}
	}
	if best < 0 {
		return nil
	}
	return p[best]
}

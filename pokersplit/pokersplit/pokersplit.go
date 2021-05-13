// Package pokersplit implements the core of the pokersplit program.
package pokersplit

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/fhchstr/pokersplit/pokersplit/players"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

//go:embed index.tmpl
var index string

var (
	tmpl = template.Must(template.New("index").Funcs(template.FuncMap{
		// Cents converts the cents to a full currency unit.
		"Cents": func(cents int) float64 {
			return float64(cents) / 100
		},
		// Iterate returns a slice of the given length. The items' value is their index.
		"Iterate": func(i int) []int {
			var ret []int
			for j := 0; j < i; j++ {
				ret = append(ret, j)
			}
			return ret
		},
		"Sorted": sorted,
	}).Parse(index))
)

// sorted returns the Players sorted by name, ignoring case and accents.
func sorted(p players.Players) players.Players {
	playersByName := make(map[string]*players.Player)
	for _, player := range p {
		playersByName[player.Name] = player
	}

	var playerNames []string
	for name := range playersByName {
		playerNames = append(playerNames, name)
	}
	cl := collate.New(language.English, collate.IgnoreCase, collate.IgnoreDiacritics)
	cl.SortStrings(playerNames)

	ret := make(players.Players, len(p))
	for i, name := range playerNames {
		ret[i] = playersByName[name]
	}
	return ret
}

type tmplData struct {
	Players players.Players
	Debts   players.Debts
	Error   error
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.Method {
	case http.MethodGet:
		err = show(w, r)
	case http.MethodPost:
		err = update(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		err = fmt.Errorf("unsupported HTTP method: %s", r.Method)
	}
	if err != nil {
		tmpl.Execute(w, tmplData{Error: err})
	}
}

func show(w http.ResponseWriter, r *http.Request) error {
	var tData tmplData
	p, err := players.FromBase64(strings.TrimPrefix(r.URL.Path, "/"))
	if err != nil {
		tData.Error = fmt.Errorf("failed to decode players: %v", err)
	}
	tData.Players = p
	if p.BuyIn() == p.Stack() {
		debts, err := p.CalculateDebts()
		if err != nil {
			tData.Error = fmt.Errorf("failed to calculate debts: %v", err)
		}
		tData.Debts = debts
	}
	return tmpl.Execute(w, tData)
}

func update(w http.ResponseWriter, r *http.Request) error {
	var tData tmplData
	if err := r.ParseForm(); err != nil {
		tData.Error = fmt.Errorf("failed to parse form: %v", err)
		return tmpl.Execute(w, tData)
	}
	p, err := players.FromForm(r.PostForm)
	if err != nil {
		tData.Error = fmt.Errorf("failed to parse players from form: %v", err)
		return tmpl.Execute(w, tData)
	}
	data, err := p.ToBase64()
	if err != nil {
		tData.Error = fmt.Errorf("failed to encode players: %v", err)
		return tmpl.Execute(w, tData)
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	u := url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   "/" + data,
	}
	w.Header().Set("Location", u.String())
	w.WriteHeader(http.StatusSeeOther)
	return nil
}

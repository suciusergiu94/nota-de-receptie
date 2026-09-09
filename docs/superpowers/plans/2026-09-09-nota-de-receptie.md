# Notă de recepție — plan de implementare

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** O aplicație desktop Go + Wails care completează, arhivează și tipărește nota de recepție a S.C. Largiana Carn S.R.L.

**Architecture:** Backend Go împărțit în pachete cu o singură responsabilitate fiecare (`appdir`, `model`, `calc`, `store`, `pdfdoc`), legat de un frontend TypeScript vanilla prin metodele exportate de `App` în `app.go`. Persistența e SQLite locală; PDF-ul se randează cu fpdf. Fiecare rând de document păstrează o fotografie a produsului, așa că editarea catalogului nu atinge documentele salvate.

**Tech Stack:** Go 1.25, Wails v2.15, `modernc.org/sqlite` v1.58, `github.com/go-pdf/fpdf` v0.9, `github.com/pkg/browser`, TypeScript 5.4, Vite 8, Vitest 5.

**Spec:** `docs/superpowers/specs/2026-09-09-nota-de-receptie-design.md` — se citește integral înainte de prima sarcină; planul argumentează din el.

## Global Constraints

- Modulul Go se numește `nota-de-receptie`. Go 1.25.0 minim, Node 20+ minim.
- Dependențele native trebuie să rămână pur Go (`modernc.org/sqlite`, `go-pdf/fpdf`), ca să se poată cross-compila pentru Windows de pe macOS. Nu se adaugă dependențe cgo.
- Tot ce vede utilizatorul este în limba română, cu diacritice. Comentariile din cod și mesajele de commit sunt scrise ca în aplicația soră `proces-verbal-transare`: comentarii în engleză, care explică *de ce*, nu *ce*.
- Erorile întoarse din Go către frontend sunt în română (`fmt.Errorf("citire document: %w", err)`).
- U/M are exact două valori permise: `Buc.` și `Kg.`.
- Cota T.V.A. implicită la prima instalare: `11`. Unitatea implicită: `S.C. Largiana Carn S.R.L.`. `next_nr` implicit: `1`. Catalogul de produse și lista de furnizori pornesc goale.
- Prețul de vânzare este cu T.V.A. Adaosul este `valoareVanzare − valoareCuTVA`; procentul de adaos este `adaos / valoareCuTVA × 100`.
- Cantitatea nu se totalizează nicăieri — nici pe ecran, nici în PDF.
- Rotunjirea peste tot: două zecimale, jumătate depărtându-se de zero.
- PDF-ul e A4 **landscape**, margini 10 mm, fără coloanele Adaos și Cota T.V.A.
- Fiecare commit se încheie cu trailerele:
  ```
  Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
  ```
- `go test ./...` trebuie să treacă la finalul fiecărei sarcini din partea de Go; `cd frontend && npm test` la finalul fiecărei sarcini din partea de frontend.

## Structura de fișiere

| Fișier | Responsabilitate |
|---|---|
| `go.mod`, `go.sum` | dependențele modulului |
| `main.go` | pornirea Wails, opțiunile ferestrei, `//go:embed all:frontend/dist` |
| `app.go` | obiectul legat de frontend; fiecare metodă exportată e un apel din UI |
| `app_test.go` | testele legăturilor, pe un store temporar |
| `internal/appdir/appdir.go` | unde stă `data.db` |
| `internal/model/model.go` | structurile partajate de toate celelalte pachete |
| `internal/calc/calc.go` | aritmetica rândului și a totalurilor |
| `internal/store/store.go` | `Open`, `Close`, setările |
| `internal/store/schema.go` | schema SQL, versiunea, migrarea, semințele |
| `internal/store/products.go` | catalogul de produse |
| `internal/store/furnizori.go` | sugestiile de furnizori |
| `internal/store/documents.go` | documentele și rândurile lor |
| `internal/pdfdoc/fold.go` | plierea diacriticelor la ASCII |
| `internal/pdfdoc/pdf.go` | randarea PDF |
| `frontend/src/api.ts` | reexportă legăturile generate + `showError` |
| `frontend/src/calc.ts` | oglinda lui `internal/calc` pentru recalcul instant |
| `frontend/src/format.ts` | numere cu virgulă, date `ZZ/LL/AAAA` |
| `frontend/src/fuzzy.ts` | căutarea în catalog |
| `frontend/src/nota.ts` | logica pură a formularului: rândul gol, rândul din produs, validarea |
| `frontend/src/router.ts` | rutare pe hash |
| `frontend/src/dialog.ts` | `showConfirm`, `showAlert` |
| `frontend/src/toast.ts` | confirmarea scurtă de la salvare |
| `frontend/src/sidebar.ts` | bara laterală și `escapeHtml` |
| `frontend/src/views/document.ts` | formularul de document |
| `frontend/src/views/setari.ts` | ecranul de setări |
| `frontend/src/style.css` | tot stilul aplicației |
| `Makefile`, `build/` | instalatoarele macOS și Windows |
| `README.md` | instalare, dezvoltare, teste, unde stau datele |

Sarcinile merg de jos în sus: întâi datele și aritmetica (testabile fără interfață), apoi persistența, apoi PDF-ul, apoi legăturile, apoi ecranele.

---

### Task 1: Schelet de proiect

**Files:**
- Create: `go.mod`, `main.go`, `app.go`, `.gitignore`, `wails.json`
- Create: `internal/appdir/appdir.go`, `internal/appdir/appdir_test.go`
- Create: `internal/model/model.go`
- Create: `frontend/dist/.gitkeep`

**Interfaces:**
- Consumes: nimic.
- Produces: `appdir.DBPath() (string, error)`; tot pachetul `model` — `model.Settings`, `model.Product`, `model.Document`, `model.Rand`, `model.DocumentSummary`, cu câmpurile din spec.

- [ ] **Step 1: Verifică uneltele**

Run:
```bash
go version && node --version && command -v wails
```
Expected: Go ≥ 1.25, Node ≥ 20, un drum către `wails`. Dacă `wails` lipsește:
`go install github.com/wailsapp/wails/v2/cmd/wails@latest`.

- [ ] **Step 2: Inițializează modulul și adaugă dependențele**

```bash
go mod init nota-de-receptie
go get github.com/wailsapp/wails/v2@v2.15.0
go get github.com/go-pdf/fpdf@v0.9.0
go get modernc.org/sqlite@v1.58.0
go get github.com/pkg/browser
```

- [ ] **Step 3: Scrie `.gitignore`**

```gitignore
# Build output
/build/bin

# Frontend
/frontend/dist/*
!/frontend/dist/.gitkeep
/frontend/node_modules

# IDE
.vscode
.idea

# OS
.DS_Store

# Build artifacts that should not be tracked
/nota-de-receptie
/frontend/package.json.md5

# Instalatoare generate
/dist

# Worktrees si spatiul de lucru al agentilor
/.claude/worktrees/
/.superpowers/
```

Apoi `mkdir -p frontend/dist && touch frontend/dist/.gitkeep`. Fișierul e
necesar ca `//go:embed all:frontend/dist` din `main.go` să compileze înainte ca
frontendul să existe.

- [ ] **Step 4: Scrie testul căzut pentru `appdir`**

`internal/appdir/appdir_test.go`:
```go
package appdir

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDBPathIsInsideUserConfigDir(t *testing.T) {
	path, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath: %v", err)
	}
	if filepath.Base(path) != "data.db" {
		t.Errorf("fisierul = %q, vrem data.db", filepath.Base(path))
	}
	if !strings.Contains(path, "nota-de-receptie") {
		t.Errorf("drumul %q nu contine numele aplicatiei", path)
	}
}

func TestDBPathCreatesTheDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath: %v", err)
	}
	// Directorul trebuie sa existe deja; fisierul, nu.
	if _, err := filepath.Abs(filepath.Dir(path)); err != nil {
		t.Fatalf("director: %v", err)
	}
}
```

- [ ] **Step 5: Rulează testul ca să cadă**

Run: `go test ./internal/appdir/`
Expected: FAIL — `undefined: DBPath`.

- [ ] **Step 6: Scrie `internal/appdir/appdir.go`**

```go
// Package appdir resolves where the application keeps its data.
package appdir

import (
	"os"
	"path/filepath"
)

const dirName = "nota-de-receptie"

// DBPath returns the SQLite file path inside the per-user config directory,
// creating the directory if it does not exist.
func DBPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "data.db"), nil
}
```

- [ ] **Step 7: Rulează testul ca să treacă**

Run: `go test ./internal/appdir/`
Expected: PASS.

- [ ] **Step 8: Scrie `internal/model/model.go`**

```go
// Package model holds the plain data structures shared by the store, the
// calculation helpers, the PDF renderer and the Wails bindings.
package model

// Settings is the single-row application configuration.
type Settings struct {
	UnitateNume string `json:"unitateNume"`
	// NextNr is the number the next document is created with.
	NextNr int `json:"nextNr"`
	// CotaTVA is the TVA percentage a new product starts from, e.g. 11 for
	// the Romanian food rate. It is a default, not a rule: every product and
	// every row carries its own rate and may depart from this one.
	CotaTVA float64 `json:"cotaTva"`
}

// Product is one entry of the catalogue kept in Setări. A document row starts
// from a product but never reads from it afterwards — see model.Rand.
type Product struct {
	ID       int64  `json:"id"`
	Denumire string `json:"denumire"`
	// UM is "Buc." or "Kg.".
	UM string `json:"um"`
	// PretVanzare is the selling price, with TVA included, so it is
	// comparable with the row's valoare cu TVA when the adaos is worked out.
	PretVanzare float64 `json:"pretVanzare"`
	CotaTVA     float64 `json:"cotaTva"`
	Ordine      int     `json:"ordine"`
}

// Rand is one line of the reception's product table.
//
// Denumire, UM, PretVanzare and CotaTVA are snapshots taken from the product
// when the document is saved, and are the only thing ever read back: editing
// the catalogue must not change a document that is already filed. ProductID
// records where the row started from and nothing more.
type Rand struct {
	ID        int64  `json:"id"`
	ProductID *int64 `json:"productId"`
	Pozitie   int    `json:"pozitie"`
	Denumire  string `json:"denumire"`
	UM        string `json:"um"`
	Cantitate float64 `json:"cantitate"`
	// PretFaraTVA is the purchase price, typed in at reception: it varies
	// from delivery to delivery, so it is not carried on the product.
	PretFaraTVA float64 `json:"pretFaraTva"`
	CotaTVA     float64 `json:"cotaTva"`
	PretVanzare float64 `json:"pretVanzare"`
}

// Document is a full notă de recepție.
type Document struct {
	ID   int64  `json:"id"`
	Nr   int    `json:"nr"`
	Data string `json:"data"` // ISO YYYY-MM-DD
	// Unitate is copied from the settings when the document is created, so a
	// PDF reprinted a year later shows what it showed when it was signed.
	Unitate             string `json:"unitate"`
	DocumentLivrare     string `json:"documentLivrare"`
	DocumentLivrareNr   string `json:"documentLivrareNr"`
	DocumentLivrareData string `json:"documentLivrareData"` // ISO, may be empty
	Furnizor            string `json:"furnizor"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
	Randuri             []Rand `json:"randuri"`
}

// DocumentSummary is the sidebar list entry.
type DocumentSummary struct {
	ID       int64  `json:"id"`
	Nr       int    `json:"nr"`
	Data     string `json:"data"`
	Furnizor string `json:"furnizor"`
}
```

- [ ] **Step 9: Scrie `main.go` și un `app.go` minimal**

`main.go`:
```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Nota de receptie",
		Width:  1440,
		Height: 900,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
```

`app.go` — deocamdată doar scheletul; store-ul se leagă în Task 8:
```go
package main

import "context"

// App is the Wails-bound application object. Every exported method here is
// callable from the frontend.
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {}
```

- [ ] **Step 10: Scrie `wails.json`**

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "nota-de-receptie",
  "outputfilename": "nota-de-receptie",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto",
  "author": {
    "name": "S.C. Largiana Carn S.R.L.",
    "email": ""
  },
  "info": {
    "companyName": "S.C. Largiana Carn S.R.L.",
    "productName": "Nota de receptie",
    "productVersion": "1.0.0",
    "copyright": "Copyright © 2026 S.C. Largiana Carn S.R.L.",
    "comments": "Aplicatie pentru completarea, arhivarea si tiparirea notei de receptie."
  }
}
```

- [ ] **Step 11: Verifică tot**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: build fără erori, `ok nota-de-receptie/internal/appdir`.

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
Schelet de proiect Go + Wails

Modulul, dependentele, structurile de date partajate si locatia bazei
de date. Frontendul are doar un dist gol, cat sa compileze go:embed.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 2: Aritmetica documentului (`internal/calc`)

**Files:**
- Create: `internal/calc/calc.go`
- Test: `internal/calc/calc_test.go`

**Interfaces:**
- Consumes: `model.Rand` din Task 1.
- Produces:
  - `calc.Round2(v float64) float64`
  - `calc.Valori` — struct cu `ValoareFaraTVA`, `ValoareCuTVA`, `ValoareVanzare`, `Adaos`, `AdaosProcent`, `AreAdaosProcent` (toate `float64`, ultimul `bool`)
  - `calc.ValoriRand(r model.Rand) Valori`
  - `calc.Totaluri(randuri []model.Rand) Valori`

- [ ] **Step 1: Scrie testele căzute**

`internal/calc/calc_test.go`:
```go
package calc

import (
	"math"
	"testing"

	"nota-de-receptie/internal/model"
)

func aproape(t *testing.T, nume string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, vrem %v", nume, got, want)
	}
}

func TestRound2(t *testing.T) {
	cazuri := []struct{ in, want float64 }{
		{1.005, 1.01},
		{2.344, 2.34},
		{2.345, 2.35},
		{-2.345, -2.35},
		{0, 0},
	}
	for _, c := range cazuri {
		aproape(t, "Round2", Round2(c.in), c.want)
	}
}

func TestValoriRand(t *testing.T) {
	// Exemplul din specificatie: 10 x 20 lei fara TVA, 11%, vandut cu 30 lei.
	v := ValoriRand(model.Rand{
		Cantitate:   10,
		PretFaraTVA: 20,
		CotaTVA:     11,
		PretVanzare: 30,
	})
	aproape(t, "ValoareFaraTVA", v.ValoareFaraTVA, 200)
	aproape(t, "ValoareCuTVA", v.ValoareCuTVA, 222)
	aproape(t, "ValoareVanzare", v.ValoareVanzare, 300)
	aproape(t, "Adaos", v.Adaos, 78)
	aproape(t, "AdaosProcent", v.AdaosProcent, 35.14)
	if !v.AreAdaosProcent {
		t.Error("AreAdaosProcent = false, vrem true")
	}
}

func TestAdaosNegativCandSeVindeSubPretulDeAchizitie(t *testing.T) {
	v := ValoriRand(model.Rand{Cantitate: 1, PretFaraTVA: 100, CotaTVA: 11, PretVanzare: 100})
	aproape(t, "Adaos", v.Adaos, -11)
	aproape(t, "AdaosProcent", v.AdaosProcent, -9.91)
}

func TestFaraAdaosProcentCandValoareaCuTVAEZero(t *testing.T) {
	// Marfa primita gratuit, sau un rand inca necompletat: procentul nu se
	// poate calcula, si nu trebuie sa devina nici infinit, nici zero.
	v := ValoriRand(model.Rand{Cantitate: 5, PretFaraTVA: 0, CotaTVA: 11, PretVanzare: 4})
	aproape(t, "ValoareCuTVA", v.ValoareCuTVA, 0)
	aproape(t, "Adaos", v.Adaos, 20)
	if v.AreAdaosProcent {
		t.Error("AreAdaosProcent = true, vrem false")
	}
	aproape(t, "AdaosProcent", v.AdaosProcent, 0)
}

func TestCoteDiferitePeRanduri(t *testing.T) {
	v := ValoriRand(model.Rand{Cantitate: 2, PretFaraTVA: 50, CotaTVA: 21, PretVanzare: 70})
	aproape(t, "ValoareCuTVA", v.ValoareCuTVA, 121)
	aproape(t, "Adaos", v.Adaos, 19)
}

func TestTotaluri(t *testing.T) {
	randuri := []model.Rand{
		{Cantitate: 10, PretFaraTVA: 20, CotaTVA: 11, PretVanzare: 30},
		{Cantitate: 2, PretFaraTVA: 50, CotaTVA: 21, PretVanzare: 70},
	}
	tot := Totaluri(randuri)
	aproape(t, "ValoareFaraTVA", tot.ValoareFaraTVA, 300)
	aproape(t, "ValoareCuTVA", tot.ValoareCuTVA, 343)
	aproape(t, "ValoareVanzare", tot.ValoareVanzare, 440)
	aproape(t, "Adaos", tot.Adaos, 97)
	// Din totaluri, nu media procentelor de pe randuri (35.14 si 15.70).
	aproape(t, "AdaosProcent", tot.AdaosProcent, 28.28)
}

func TestTotaluriPeListaGoala(t *testing.T) {
	tot := Totaluri(nil)
	aproape(t, "ValoareCuTVA", tot.ValoareCuTVA, 0)
	if tot.AreAdaosProcent {
		t.Error("AreAdaosProcent = true pe lista goala, vrem false")
	}
}
```

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `go test ./internal/calc/`
Expected: FAIL — `undefined: Round2`.

- [ ] **Step 3: Scrie `internal/calc/calc.go`**

```go
// Package calc holds the arithmetic of the notă de recepție: the values of one
// row and the totals over a document's rows.
package calc

import (
	"math"

	"nota-de-receptie/internal/model"
)

// Valori is what one row, or a whole document, comes to. The same shape serves
// both: a total is the sum of its rows in every field but the percentage,
// which is worked out from the summed figures.
type Valori struct {
	ValoareFaraTVA float64 `json:"valoareFaraTva"`
	ValoareCuTVA   float64 `json:"valoareCuTva"`
	ValoareVanzare float64 `json:"valoareVanzare"`
	Adaos          float64 `json:"adaos"`
	AdaosProcent   float64 `json:"adaosProcent"`
	// AreAdaosProcent is false when the purchase value is zero — goods
	// received for nothing, or a row not filled in yet. The percentage has no
	// value then, and the form shows a dash rather than a zero that would
	// read as "no markup".
	AreAdaosProcent bool `json:"areAdaosProcent"`
}

// Round2 rounds to two decimals, half away from zero.
func Round2(v float64) float64 {
	r := math.Round(math.Abs(v)*100) / 100
	if v < 0 {
		return -r
	}
	return r
}

// ValoriRand works out one row's values.
//
// PretVanzare carries TVA, like ValoareCuTVA, so the adaos subtracts two
// figures stated the same way. Subtracting a price without TVA from one with
// it would produce a number that is neither a margin nor anything else.
func ValoriRand(r model.Rand) Valori {
	faraTVA := Round2(r.Cantitate * r.PretFaraTVA)
	cuTVA := Round2(faraTVA * (1 + r.CotaTVA/100))
	vanzare := Round2(r.Cantitate * r.PretVanzare)
	return valoriDin(faraTVA, cuTVA, vanzare)
}

// Totaluri sums a document's rows.
//
// Quantity is deliberately absent: U/M is "Buc." on one row and "Kg." on the
// next, so a sum over them would mean nothing. The spec drops the quantity
// total from both the screen and the PDF.
func Totaluri(randuri []model.Rand) Valori {
	var faraTVA, cuTVA, vanzare float64
	for _, r := range randuri {
		v := ValoriRand(r)
		faraTVA += v.ValoareFaraTVA
		cuTVA += v.ValoareCuTVA
		vanzare += v.ValoareVanzare
	}
	return valoriDin(Round2(faraTVA), Round2(cuTVA), Round2(vanzare))
}

// valoriDin derives the adaos and its percentage from three already-rounded
// values. Both ValoriRand and Totaluri end here, so a row and a total can
// never disagree about what an adaos is.
func valoriDin(faraTVA, cuTVA, vanzare float64) Valori {
	v := Valori{
		ValoareFaraTVA: faraTVA,
		ValoareCuTVA:   cuTVA,
		ValoareVanzare: vanzare,
		Adaos:          Round2(vanzare - cuTVA),
	}
	if cuTVA != 0 {
		v.AdaosProcent = Round2(v.Adaos / cuTVA * 100)
		v.AreAdaosProcent = true
	}
	return v
}
```

- [ ] **Step 4: Rulează testele ca să treacă**

Run: `go test ./internal/calc/ -v`
Expected: PASS, toate cele șapte teste.

- [ ] **Step 5: Commit**

```bash
git add internal/calc
git commit -m "$(cat <<'EOF'
Aritmetica documentului

Valorile unui rand si totalurile peste randuri, cu adaosul ca diferenta
intre valoarea de vanzare si cea de achizitie, ambele cu TVA. Cantitatea
nu se totalizeaza: U/M variaza de la rand la rand.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 3: Store — schemă, migrare, setări

**Files:**
- Create: `internal/store/store.go`, `internal/store/schema.go`
- Test: `internal/store/store_test.go`

**Interfaces:**
- Consumes: `model.Settings` din Task 1.
- Produces:
  - `store.Open(path string) (*Store, error)`, `(*Store).Close() error`
  - `(*Store).GetSettings() (model.Settings, error)`, `(*Store).SaveSettings(model.Settings) error`
  - `store.ErrNotFound` — folosit de Task 6
  - constantele `defaultUnitate`, `defaultCotaTVA` (nedeclarate public)

- [ ] **Step 1: Scrie testele căzute**

`internal/store/store_test.go`:
```go
package store

import (
	"path/filepath"
	"testing"
)

// deschide creates a store on a fresh file under the test's temp dir.
func deschide(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenSeamanaSetarileImplicite(t *testing.T) {
	s := deschide(t)
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.UnitateNume != "S.C. Largiana Carn S.R.L." {
		t.Errorf("UnitateNume = %q", got.UnitateNume)
	}
	if got.NextNr != 1 {
		t.Errorf("NextNr = %d, vrem 1", got.NextNr)
	}
	if got.CotaTVA != 11 {
		t.Errorf("CotaTVA = %v, vrem 11", got.CotaTVA)
	}
}

func TestCatalogulSiFurnizoriiPornescGoale(t *testing.T) {
	s := deschide(t)
	var produse, furnizori int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&produse); err != nil {
		t.Fatalf("numarare produse: %v", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM furnizori`).Scan(&furnizori); err != nil {
		t.Fatalf("numarare furnizori: %v", err)
	}
	if produse != 0 || furnizori != 0 {
		t.Errorf("produse = %d, furnizori = %d; vrem 0 si 0", produse, furnizori)
	}
}

func TestSaveSettingsSuprascrie(t *testing.T) {
	s := deschide(t)
	vrem := model.Settings{UnitateNume: "Alta firma", NextNr: 42, CotaTVA: 21}
	if err := s.SaveSettings(vrem); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got != vrem {
		t.Errorf("setari = %+v, vrem %+v", got, vrem)
	}
}

func TestADouaDeschidereNuReseamana(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SaveSettings(model.Settings{UnitateNume: "X", NextNr: 7, CotaTVA: 5}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("a doua Open: %v", err)
	}
	defer s2.Close()
	got, err := s2.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.NextNr != 7 {
		t.Errorf("NextNr = %d dupa redeschidere, vrem 7", got.NextNr)
	}
}
```

Adaugă `"nota-de-receptie/internal/model"` la importuri.

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `go test ./internal/store/`
Expected: FAIL — `undefined: Open`.

- [ ] **Step 3: Scrie `internal/store/schema.go`**

```go
package store

import (
	"database/sql"
	"fmt"
)

const schemaVersion = 1

const schemaSQL = `
CREATE TABLE IF NOT EXISTS settings (
  id           INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT    NOT NULL DEFAULT '',
  next_nr      INTEGER NOT NULL DEFAULT 1,
  cota_tva     REAL    NOT NULL DEFAULT 11
);

CREATE TABLE IF NOT EXISTS products (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  denumire     TEXT    NOT NULL,
  um           TEXT    NOT NULL DEFAULT 'Kg.',
  pret_vanzare REAL    NOT NULL DEFAULT 0,
  cota_tva     REAL    NOT NULL DEFAULT 11,
  ordine       INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_denumire
  ON products(denumire COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS furnizori (
  id   INTEGER PRIMARY KEY AUTOINCREMENT,
  nume TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_furnizori_nume
  ON furnizori(nume COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS documents (
  id                    INTEGER PRIMARY KEY AUTOINCREMENT,
  nr                    INTEGER NOT NULL,
  data                  TEXT    NOT NULL,
  unitate               TEXT    NOT NULL DEFAULT '',
  document_livrare      TEXT    NOT NULL DEFAULT '',
  document_livrare_nr   TEXT    NOT NULL DEFAULT '',
  document_livrare_data TEXT    NOT NULL DEFAULT '',
  furnizor              TEXT    NOT NULL DEFAULT '',
  created_at            TEXT    NOT NULL,
  updated_at            TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS document_rows (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id   INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  product_id    INTEGER          REFERENCES products(id)  ON DELETE SET NULL,
  pozitie       INTEGER NOT NULL,
  denumire      TEXT    NOT NULL,
  um            TEXT    NOT NULL DEFAULT 'Kg.',
  cantitate     REAL    NOT NULL DEFAULT 0,
  pret_fara_tva REAL    NOT NULL DEFAULT 0,
  cota_tva      REAL    NOT NULL DEFAULT 11,
  pret_vanzare  REAL    NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_rows_document ON document_rows(document_id, pozitie);
`

// defaultUnitate is the company the paper form is printed for.
const defaultUnitate = "S.C. Largiana Carn S.R.L."

// defaultCotaTVA is the standard Romanian TVA rate for food, in percent.
const defaultCotaTVA = 11.0

// migrate brings the database to schemaVersion and writes the first-run
// settings row. It is safe to call on every startup: the row is written only
// when it is absent.
//
// There is only one schema version so far. When the shape has to change on a
// database that has shipped, this is where a real migration goes — never a
// drop-and-recreate, since by then the file holds documents nobody can retype.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return err
	}
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return err
	}

	var seeded int
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings WHERE id = 1`).Scan(&seeded); err != nil {
		return err
	}
	if seeded > 0 {
		return nil
	}
	_, err := db.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva) VALUES (1, ?, 1, ?)`,
		defaultUnitate, defaultCotaTVA,
	)
	return err
}
```

Catalogul de produse și lista de furnizori rămân goale intenționat — nu se
scrie nicio sămânță pentru ele.

- [ ] **Step 4: Scrie `internal/store/store.go`**

```go
// Package store is the SQLite persistence layer.
package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"nota-de-receptie/internal/model"
)

// ErrNotFound is returned when an id does not exist.
var ErrNotFound = errors.New("documentul nu a fost gasit")

// Store owns the database handle.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite file at path, applies the schema
// and writes the first-run settings row.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("deschidere bazei de date: %w", err)
	}
	// SQLite tolerates a single writer; one connection keeps writes serialized.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectare la baza de date: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrare baza de date: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// GetSettings reads the single settings row.
func (s *Store) GetSettings() (model.Settings, error) {
	var out model.Settings
	err := s.db.QueryRow(
		`SELECT unitate_nume, next_nr, cota_tva FROM settings WHERE id = 1`,
	).Scan(&out.UnitateNume, &out.NextNr, &out.CotaTVA)
	if err != nil {
		return model.Settings{}, fmt.Errorf("citire setari: %w", err)
	}
	return out, nil
}

// SaveSettings overwrites the settings row.
func (s *Store) SaveSettings(in model.Settings) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (id, unitate_nume, next_nr, cota_tva)
		 VALUES (1, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET unitate_nume = excluded.unitate_nume,
		        next_nr = excluded.next_nr, cota_tva = excluded.cota_tva`,
		in.UnitateNume, in.NextNr, in.CotaTVA,
	)
	if err != nil {
		return fmt.Errorf("salvare setari: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Rulează testele ca să treacă**

Run: `go test ./internal/store/ -v`
Expected: PASS, patru teste.

- [ ] **Step 6: Commit**

```bash
git add internal/store
git commit -m "$(cat <<'EOF'
Store: schema, migrare si setari

Baza SQLite cu tabelele de setari, produse, furnizori, documente si
randuri. La prima pornire se scrie doar randul de setari; catalogul si
lista de furnizori raman goale, cum cere specificatia.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 4: Store — catalogul de produse

**Files:**
- Create: `internal/store/products.go`
- Test: `internal/store/products_test.go`

**Interfaces:**
- Consumes: `Store` și helperul de test `deschide(t)` din Task 3; `model.Product`.
- Produces:
  - `(*Store).ListProducts() ([]model.Product, error)` — în ordinea `ordine`, apoi `id`
  - `(*Store).SaveProducts(produse []model.Product) error` — sincronizează catalogul păstrând id-urile
  - `(*Store).AddProduct(p model.Product) (model.Product, error)` — inserează unul singur, la coada listei
  - `store.ErrDenumireDuplicata`

- [ ] **Step 1: Scrie testele căzute**

`internal/store/products_test.go`:
```go
package store

import (
	"errors"
	"testing"

	"nota-de-receptie/internal/model"
)

func TestCatalogulPornesteGol(t *testing.T) {
	s := deschide(t)
	produse, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(produse) != 0 {
		t.Errorf("len = %d, vrem 0", len(produse))
	}
}

func TestAddProductPuneLaCoada(t *testing.T) {
	s := deschide(t)
	unu, err := s.AddProduct(model.Product{Denumire: "Pulpa fara os", UM: "Kg.", PretVanzare: 21.9, CotaTVA: 11})
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if unu.ID == 0 {
		t.Error("produsul salvat nu a primit id")
	}
	doi, err := s.AddProduct(model.Product{Denumire: "Oua", UM: "Buc.", PretVanzare: 1.2, CotaTVA: 11})
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if doi.Ordine <= unu.Ordine {
		t.Errorf("ordine = %d si %d; al doilea produs trebuie sa vina dupa primul", unu.Ordine, doi.Ordine)
	}

	produse, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(produse) != 2 || produse[0].Denumire != "Pulpa fara os" || produse[1].Denumire != "Oua" {
		t.Errorf("catalogul = %+v", produse)
	}
}

func TestAddProductRefuzaDenumireaDuplicata(t *testing.T) {
	s := deschide(t)
	if _, err := s.AddProduct(model.Product{Denumire: "Costita", UM: "Kg.", CotaTVA: 11}); err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	// Aceeasi denumire cu alte majuscule este tot un duplicat: utilizatorul
	// n-are cum sa deosebeasca doua intrari care arata la fel in lista.
	_, err := s.AddProduct(model.Product{Denumire: "costita", UM: "Kg.", CotaTVA: 11})
	if !errors.Is(err, ErrDenumireDuplicata) {
		t.Errorf("err = %v, vrem ErrDenumireDuplicata", err)
	}
}

func TestSaveProductsPastreazaIdurileCeloraRamase(t *testing.T) {
	s := deschide(t)
	a, _ := s.AddProduct(model.Product{Denumire: "A", UM: "Kg.", PretVanzare: 10, CotaTVA: 11})
	b, _ := s.AddProduct(model.Product{Denumire: "B", UM: "Kg.", PretVanzare: 20, CotaTVA: 11})

	// B se redenumeste si isi schimba pretul, A ramane, se adauga un C nou.
	err := s.SaveProducts([]model.Product{
		{ID: a.ID, Denumire: "A", UM: "Kg.", PretVanzare: 10, CotaTVA: 11, Ordine: 0},
		{ID: b.ID, Denumire: "B redenumit", UM: "Buc.", PretVanzare: 25, CotaTVA: 21, Ordine: 1},
		{ID: 0, Denumire: "C", UM: "Kg.", PretVanzare: 5, CotaTVA: 11, Ordine: 2},
	})
	if err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	produse, _ := s.ListProducts()
	if len(produse) != 3 {
		t.Fatalf("len = %d, vrem 3", len(produse))
	}
	if produse[1].ID != b.ID {
		t.Errorf("id-ul lui B = %d, vrem %d pastrat", produse[1].ID, b.ID)
	}
	if produse[1].Denumire != "B redenumit" || produse[1].UM != "Buc." || produse[1].PretVanzare != 25 || produse[1].CotaTVA != 21 {
		t.Errorf("B nu s-a actualizat: %+v", produse[1])
	}
	if produse[2].Denumire != "C" || produse[2].ID == 0 {
		t.Errorf("C nu s-a inserat: %+v", produse[2])
	}
}

func TestSaveProductsStergeCeLipseste(t *testing.T) {
	s := deschide(t)
	a, _ := s.AddProduct(model.Product{Denumire: "A", UM: "Kg.", CotaTVA: 11})
	s.AddProduct(model.Product{Denumire: "B", UM: "Kg.", CotaTVA: 11})

	if err := s.SaveProducts([]model.Product{{ID: a.ID, Denumire: "A", UM: "Kg.", CotaTVA: 11, Ordine: 0}}); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}
	produse, _ := s.ListProducts()
	if len(produse) != 1 || produse[0].Denumire != "A" {
		t.Errorf("catalogul = %+v, vrem doar A", produse)
	}
}

func TestSaveProductsRefuzaDuplicatele(t *testing.T) {
	s := deschide(t)
	err := s.SaveProducts([]model.Product{
		{Denumire: "Costita", UM: "Kg.", CotaTVA: 11, Ordine: 0},
		{Denumire: "COSTITA", UM: "Kg.", CotaTVA: 11, Ordine: 1},
	})
	if !errors.Is(err, ErrDenumireDuplicata) {
		t.Errorf("err = %v, vrem ErrDenumireDuplicata", err)
	}
	// Refuzul e total: nimic nu trebuie sa fi ramas scris.
	produse, _ := s.ListProducts()
	if len(produse) != 0 {
		t.Errorf("catalogul = %+v dupa un refuz, vrem gol", produse)
	}
}
```

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `go test ./internal/store/`
Expected: FAIL — `undefined: ErrDenumireDuplicata`.

- [ ] **Step 3: Scrie `internal/store/products.go`**

```go
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"nota-de-receptie/internal/model"
)

// ErrDenumireDuplicata is returned when two products would share a name.
// Names are compared without regard to case: two catalogue entries that read
// the same are the same as far as the person picking one is concerned.
var ErrDenumireDuplicata = errors.New("exista deja un produs cu aceasta denumire")

// ListProducts returns the whole catalogue in display order.
func (s *Store) ListProducts() ([]model.Product, error) {
	rows, err := s.db.Query(
		`SELECT id, denumire, um, pret_vanzare, cota_tva, ordine
		 FROM products ORDER BY ordine, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire produse: %w", err)
	}
	defer rows.Close()

	out := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Denumire, &p.UM, &p.PretVanzare, &p.CotaTVA, &p.Ordine); err != nil {
			return nil, fmt.Errorf("citire produs: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AddProduct inserts one product at the end of the catalogue and returns it
// with the id and ordine it was given. It exists so a name typed into the
// document form can be filed without leaving the form.
func (s *Store) AddProduct(p model.Product) (model.Product, error) {
	var maxOrdine sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(ordine) FROM products`).Scan(&maxOrdine); err != nil {
		return model.Product{}, fmt.Errorf("citire ordine produse: %w", err)
	}
	p.Ordine = int(maxOrdine.Int64) + 1
	if !maxOrdine.Valid {
		p.Ordine = 0
	}

	res, err := s.db.Exec(
		`INSERT INTO products (denumire, um, pret_vanzare, cota_tva, ordine)
		 VALUES (?, ?, ?, ?, ?)`,
		strings.TrimSpace(p.Denumire), p.UM, p.PretVanzare, p.CotaTVA, p.Ordine,
	)
	if err != nil {
		if esteConflictDeDenumire(err) {
			return model.Product{}, ErrDenumireDuplicata
		}
		return model.Product{}, fmt.Errorf("salvare produs: %w", err)
	}
	if p.ID, err = res.LastInsertId(); err != nil {
		return model.Product{}, fmt.Errorf("salvare produs: %w", err)
	}
	p.Denumire = strings.TrimSpace(p.Denumire)
	return p, nil
}

// SaveProducts brings the catalogue in line with the given list: rows with an
// id are updated, rows without one are inserted, and anything not in the list
// is deleted.
//
// It deliberately does not delete everything and reinsert. document_rows keeps
// a product_id with ON DELETE SET NULL, and wiping the table would sever every
// saved document's link to the catalogue — the documents would survive (they
// carry their own snapshot) but they would all look as though their products
// had been deleted. Updating in place keeps the ids, so only products the user
// actually removed lose their link.
func (s *Store) SaveProducts(produse []model.Product) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	defer tx.Rollback()

	pastrate := make([]any, 0, len(produse))
	for i, p := range produse {
		denumire := strings.TrimSpace(p.Denumire)
		if p.ID == 0 {
			res, err := tx.Exec(
				`INSERT INTO products (denumire, um, pret_vanzare, cota_tva, ordine)
				 VALUES (?, ?, ?, ?, ?)`,
				denumire, p.UM, p.PretVanzare, p.CotaTVA, i,
			)
			if err != nil {
				if esteConflictDeDenumire(err) {
					return ErrDenumireDuplicata
				}
				return fmt.Errorf("salvare produs %q: %w", denumire, err)
			}
			id, err := res.LastInsertId()
			if err != nil {
				return fmt.Errorf("salvare produs %q: %w", denumire, err)
			}
			pastrate = append(pastrate, id)
			continue
		}
		if _, err := tx.Exec(
			`UPDATE products SET denumire = ?, um = ?, pret_vanzare = ?, cota_tva = ?, ordine = ?
			 WHERE id = ?`,
			denumire, p.UM, p.PretVanzare, p.CotaTVA, i, p.ID,
		); err != nil {
			if esteConflictDeDenumire(err) {
				return ErrDenumireDuplicata
			}
			return fmt.Errorf("salvare produs %q: %w", denumire, err)
		}
		pastrate = append(pastrate, p.ID)
	}

	if err := stergeProduseleLipsa(tx, pastrate); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("salvare produse: %w", err)
	}
	return nil
}

// stergeProduseleLipsa removes every product whose id is not in pastrate.
func stergeProduseleLipsa(tx *sql.Tx, pastrate []any) error {
	if len(pastrate) == 0 {
		if _, err := tx.Exec(`DELETE FROM products`); err != nil {
			return fmt.Errorf("stergere produse: %w", err)
		}
		return nil
	}
	semne := strings.TrimSuffix(strings.Repeat("?,", len(pastrate)), ",")
	if _, err := tx.Exec(`DELETE FROM products WHERE id NOT IN (`+semne+`)`, pastrate...); err != nil {
		return fmt.Errorf("stergere produse: %w", err)
	}
	return nil
}

// esteConflictDeDenumire reports whether err is the unique index on
// products.denumire refusing a duplicate. modernc.org/sqlite reports it as a
// message rather than a typed error, so the message is what there is to match
// on; the index name makes the match specific to this one constraint.
func esteConflictDeDenumire(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: products.denumire")
}
```

- [ ] **Step 4: Rulează testele ca să treacă**

Run: `go test ./internal/store/ -run Product -v`
Expected: PASS, șase teste.

Dacă `esteConflictDeDenumire` nu prinde eroarea, tipărește mesajul real
(`t.Logf("%v", err)`) și potrivește pe textul întors de driver — indexul e
`idx_products_denumire`, deci mesajul poate suna și
`UNIQUE constraint failed: index 'idx_products_denumire'`. Ajustează șirul
căutat, nu testul.

- [ ] **Step 5: Commit**

```bash
git add internal/store
git commit -m "$(cat <<'EOF'
Store: catalogul de produse

Listare, adaugare individuala (pentru salvarea din formular) si
sincronizarea intregii liste. Sincronizarea actualizeaza in loc, nu
sterge si reinsereaza: altfel fiecare document salvat si-ar pierde
legatura cu catalogul.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 5: Store — sugestiile de furnizori

**Files:**
- Create: `internal/store/furnizori.go`
- Test: `internal/store/furnizori_test.go`

**Interfaces:**
- Consumes: `Store`, `deschide(t)`.
- Produces:
  - `(*Store).ListFurnizori() ([]string, error)` — alfabetic
  - `(*Store).DeleteFurnizor(nume string) error`
  - `retineFurnizor(tx *sql.Tx, nume string) error` — nepublică, chemată de `SaveDocument` în Task 6

- [ ] **Step 1: Scrie testele căzute**

`internal/store/furnizori_test.go`:
```go
package store

import "testing"

func TestFurnizoriiPornescGol(t *testing.T) {
	s := deschide(t)
	f, err := s.ListFurnizori()
	if err != nil {
		t.Fatalf("ListFurnizori: %v", err)
	}
	if len(f) != 0 {
		t.Errorf("len = %d, vrem 0", len(f))
	}
}

func TestRetineFurnizorIgnoraDuplicateleSiGolul(t *testing.T) {
	s := deschide(t)
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	for _, nume := range []string{"Alfa SRL", "alfa srl", "  ", "", "Beta SA"} {
		if err := retineFurnizor(tx, nume); err != nil {
			t.Fatalf("retineFurnizor(%q): %v", nume, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := s.ListFurnizori()
	if err != nil {
		t.Fatalf("ListFurnizori: %v", err)
	}
	if len(got) != 2 || got[0] != "Alfa SRL" || got[1] != "Beta SA" {
		t.Errorf("furnizori = %v, vrem [Alfa SRL Beta SA]", got)
	}
}

func TestDeleteFurnizor(t *testing.T) {
	s := deschide(t)
	tx, _ := s.db.Begin()
	retineFurnizor(tx, "Alfa SRL")
	retineFurnizor(tx, "Beta SA")
	tx.Commit()

	if err := s.DeleteFurnizor("alfa srl"); err != nil {
		t.Fatalf("DeleteFurnizor: %v", err)
	}
	got, _ := s.ListFurnizori()
	if len(got) != 1 || got[0] != "Beta SA" {
		t.Errorf("furnizori = %v, vrem [Beta SA]", got)
	}
}

func TestDeleteFurnizorInexistentNuEEroare(t *testing.T) {
	s := deschide(t)
	if err := s.DeleteFurnizor("Cineva"); err != nil {
		t.Errorf("DeleteFurnizor pe un nume inexistent: %v", err)
	}
}
```

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `go test ./internal/store/ -run Furnizor`
Expected: FAIL — `undefined: retineFurnizor`.

- [ ] **Step 3: Scrie `internal/store/furnizori.go`**

```go
package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// ListFurnizori returns the remembered supplier names, alphabetically.
func (s *Store) ListFurnizori() ([]string, error) {
	rows, err := s.db.Query(`SELECT nume FROM furnizori ORDER BY nume COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("citire furnizori: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var nume string
		if err := rows.Scan(&nume); err != nil {
			return nil, fmt.Errorf("citire furnizor: %w", err)
		}
		out = append(out, nume)
	}
	return out, rows.Err()
}

// DeleteFurnizor forgets one suggestion. Documents keep the supplier as plain
// text, so nothing that has been filed changes: this removes an entry from a
// list of hints, not a supplier from history.
//
// A name that is not there is not an error — the user asked for it to be gone,
// and it is.
func (s *Store) DeleteFurnizor(nume string) error {
	if _, err := s.db.Exec(
		`DELETE FROM furnizori WHERE nume = ? COLLATE NOCASE`, strings.TrimSpace(nume),
	); err != nil {
		return fmt.Errorf("stergere furnizor: %w", err)
	}
	return nil
}

// retineFurnizor records a supplier name as a suggestion for later documents.
// It runs inside SaveDocument's transaction, so a document and the suggestion
// it produced are committed together.
//
// An empty name is skipped rather than stored: a document may legitimately go
// without a supplier, and a blank suggestion would be an unpickable entry in
// the list. A name already there, in any casing, is left as it was — the first
// spelling wins, so the list does not grow a second entry that reads the same.
func retineFurnizor(tx *sql.Tx, nume string) error {
	nume = strings.TrimSpace(nume)
	if nume == "" {
		return nil
	}
	if _, err := tx.Exec(
		`INSERT INTO furnizori (nume) VALUES (?)
		 ON CONFLICT DO NOTHING`, nume,
	); err != nil {
		return fmt.Errorf("memorare furnizor: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Rulează testele ca să treacă**

Run: `go test ./internal/store/ -run Furnizor -v`
Expected: PASS, patru teste.

- [ ] **Step 5: Commit**

```bash
git add internal/store
git commit -m "$(cat <<'EOF'
Store: sugestiile de furnizori

Numele se memoreaza in tranzactia documentului si se pot sterge din
Setari. Stergerea nu atinge documentele: ele pastreaza furnizorul ca
text.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 6: Store — documentele

**Files:**
- Create: `internal/store/documents.go`
- Test: `internal/store/documents_test.go`

**Interfaces:**
- Consumes: `Store`, `deschide(t)`, `retineFurnizor`, `ErrNotFound`, `model.Document`, `model.Rand`, `model.DocumentSummary`.
- Produces:
  - `(*Store).ListDocuments() ([]model.DocumentSummary, error)`
  - `(*Store).GetDocument(id int64) (model.Document, error)`
  - `(*Store).SaveDocument(doc model.Document) (model.Document, error)`
  - `(*Store).DeleteDocument(id int64) error`

- [ ] **Step 1: Scrie testele căzute**

`internal/store/documents_test.go`:
```go
package store

import (
	"errors"
	"testing"

	"nota-de-receptie/internal/model"
)

// docExemplu is a two-row reception, ready to save.
func docExemplu(produsID *int64) model.Document {
	return model.Document{
		Nr:                  1,
		Data:                "2026-09-09",
		Unitate:             "S.C. Largiana Carn S.R.L.",
		DocumentLivrare:     "Factura",
		DocumentLivrareNr:   "1234",
		DocumentLivrareData: "2026-09-08",
		Furnizor:            "Alfa SRL",
		Randuri: []model.Rand{
			{ProductID: produsID, Pozitie: 0, Denumire: "Pulpa fara os", UM: "Kg.",
				Cantitate: 10, PretFaraTVA: 20, CotaTVA: 11, PretVanzare: 30},
			{Pozitie: 1, Denumire: "Oua", UM: "Buc.",
				Cantitate: 30, PretFaraTVA: 0.9, CotaTVA: 11, PretVanzare: 1.5},
		},
	}
}

func TestSaveSiGetDocument(t *testing.T) {
	s := deschide(t)
	salvat, err := s.SaveDocument(docExemplu(nil))
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if salvat.ID == 0 {
		t.Fatal("documentul salvat nu a primit id")
	}
	if salvat.CreatedAt == "" || salvat.UpdatedAt == "" {
		t.Error("createdAt/updatedAt nu au fost completate")
	}

	got, err := s.GetDocument(salvat.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Nr != 1 || got.Furnizor != "Alfa SRL" || got.DocumentLivrare != "Factura" {
		t.Errorf("antetul = %+v", got)
	}
	if len(got.Randuri) != 2 {
		t.Fatalf("len(Randuri) = %d, vrem 2", len(got.Randuri))
	}
	if got.Randuri[0].Denumire != "Pulpa fara os" || got.Randuri[0].UM != "Kg." {
		t.Errorf("randul 0 = %+v", got.Randuri[0])
	}
	if got.Randuri[1].UM != "Buc." || got.Randuri[1].Cantitate != 30 {
		t.Errorf("randul 1 = %+v", got.Randuri[1])
	}
}

func TestGetDocumentInexistent(t *testing.T) {
	s := deschide(t)
	_, err := s.GetDocument(999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, vrem ErrNotFound", err)
	}
}

func TestSaveDocumentUrcaContorul(t *testing.T) {
	s := deschide(t)
	if _, err := s.SaveDocument(docExemplu(nil)); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	setari, _ := s.GetSettings()
	if setari.NextNr != 2 {
		t.Errorf("NextNr = %d dupa documentul 1, vrem 2", setari.NextNr)
	}

	// Un numar sarit urca contorul peste el, nu cu unu.
	d := docExemplu(nil)
	d.Nr = 10
	if _, err := s.SaveDocument(d); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	setari, _ = s.GetSettings()
	if setari.NextNr != 11 {
		t.Errorf("NextNr = %d dupa documentul 10, vrem 11", setari.NextNr)
	}

	// Un numar mai mic nu coboara contorul.
	d2 := docExemplu(nil)
	d2.Nr = 3
	if _, err := s.SaveDocument(d2); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	setari, _ = s.GetSettings()
	if setari.NextNr != 11 {
		t.Errorf("NextNr = %d dupa documentul 3, vrem tot 11", setari.NextNr)
	}
}

func TestSaveDocumentExistentNuUrcaContorul(t *testing.T) {
	s := deschide(t)
	salvat, _ := s.SaveDocument(docExemplu(nil))
	setari, _ := s.GetSettings()
	inainte := setari.NextNr

	salvat.Furnizor = "Beta SA"
	if _, err := s.SaveDocument(salvat); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	setari, _ = s.GetSettings()
	if setari.NextNr != inainte {
		t.Errorf("NextNr = %d dupa o reeditare, vrem tot %d", setari.NextNr, inainte)
	}
}

func TestSaveDocumentInlocuiesteRandurile(t *testing.T) {
	s := deschide(t)
	salvat, _ := s.SaveDocument(docExemplu(nil))

	salvat.Randuri = []model.Rand{
		{Pozitie: 0, Denumire: "Singurul rand", UM: "Kg.", Cantitate: 1, PretFaraTVA: 1, CotaTVA: 11, PretVanzare: 2},
	}
	if _, err := s.SaveDocument(salvat); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	got, _ := s.GetDocument(salvat.ID)
	if len(got.Randuri) != 1 || got.Randuri[0].Denumire != "Singurul rand" {
		t.Errorf("randuri = %+v", got.Randuri)
	}
}

func TestSaveDocumentMemoreazaFurnizorul(t *testing.T) {
	s := deschide(t)
	if _, err := s.SaveDocument(docExemplu(nil)); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	f, _ := s.ListFurnizori()
	if len(f) != 1 || f[0] != "Alfa SRL" {
		t.Errorf("furnizori = %v, vrem [Alfa SRL]", f)
	}
}

func TestEditareaProdusuluiNuSchimbaDocumentulSalvat(t *testing.T) {
	// Cerinta centrala a aplicatiei: randul e o fotografie, nu o legatura.
	s := deschide(t)
	p, err := s.AddProduct(model.Product{Denumire: "Pulpa fara os", UM: "Kg.", PretVanzare: 30, CotaTVA: 11})
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	salvat, err := s.SaveDocument(docExemplu(&p.ID))
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	if err := s.SaveProducts([]model.Product{
		{ID: p.ID, Denumire: "Pulpa fara os (alt nume)", UM: "Buc.", PretVanzare: 99, CotaTVA: 21, Ordine: 0},
	}); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	got, _ := s.GetDocument(salvat.ID)
	r := got.Randuri[0]
	if r.Denumire != "Pulpa fara os" || r.UM != "Kg." || r.PretVanzare != 30 || r.CotaTVA != 11 {
		t.Errorf("randul s-a schimbat odata cu produsul: %+v", r)
	}
}

func TestStergereaProdusuluiLasaRandulIntreg(t *testing.T) {
	s := deschide(t)
	p, _ := s.AddProduct(model.Product{Denumire: "Pulpa fara os", UM: "Kg.", PretVanzare: 30, CotaTVA: 11})
	salvat, _ := s.SaveDocument(docExemplu(&p.ID))

	if err := s.SaveProducts(nil); err != nil {
		t.Fatalf("SaveProducts: %v", err)
	}

	got, _ := s.GetDocument(salvat.ID)
	r := got.Randuri[0]
	if r.ProductID != nil {
		t.Errorf("ProductID = %v, vrem nil dupa stergerea produsului", *r.ProductID)
	}
	if r.Denumire != "Pulpa fara os" || r.PretVanzare != 30 {
		t.Errorf("randul si-a pierdut datele: %+v", r)
	}
}

func TestListDocumentsCelMaiNouPrimul(t *testing.T) {
	s := deschide(t)
	vechi := docExemplu(nil)
	vechi.Data = "2026-01-01"
	vechi.Nr = 1
	s.SaveDocument(vechi)

	nou := docExemplu(nil)
	nou.Data = "2026-09-09"
	nou.Nr = 2
	s.SaveDocument(nou)

	lista, err := s.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(lista) != 2 {
		t.Fatalf("len = %d, vrem 2", len(lista))
	}
	if lista[0].Nr != 2 {
		t.Errorf("primul din lista = NR %d, vrem NR 2", lista[0].Nr)
	}
	if lista[0].Furnizor != "Alfa SRL" {
		t.Errorf("rezumatul nu poarta furnizorul: %+v", lista[0])
	}
}

func TestDeleteDocumentSterageSiRandurile(t *testing.T) {
	s := deschide(t)
	salvat, _ := s.SaveDocument(docExemplu(nil))

	if err := s.DeleteDocument(salvat.ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	if _, err := s.GetDocument(salvat.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, vrem ErrNotFound", err)
	}
	var randuri int
	s.db.QueryRow(`SELECT COUNT(*) FROM document_rows`).Scan(&randuri)
	if randuri != 0 {
		t.Errorf("au ramas %d randuri orfane", randuri)
	}
}
```

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `go test ./internal/store/ -run Document`
Expected: FAIL — `undefined: (*Store).SaveDocument`.

- [ ] **Step 3: Scrie `internal/store/documents.go`**

```go
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nota-de-receptie/internal/model"
)

// ListDocuments returns every document, newest first.
func (s *Store) ListDocuments() ([]model.DocumentSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, nr, data, furnizor FROM documents ORDER BY data DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("citire documente: %w", err)
	}
	defer rows.Close()

	out := []model.DocumentSummary{}
	for rows.Next() {
		var d model.DocumentSummary
		if err := rows.Scan(&d.ID, &d.Nr, &d.Data, &d.Furnizor); err != nil {
			return nil, fmt.Errorf("citire document: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDocument reads one document with its rows.
func (s *Store) GetDocument(id int64) (model.Document, error) {
	var d model.Document
	err := s.db.QueryRow(
		`SELECT id, nr, data, unitate, document_livrare, document_livrare_nr,
		        document_livrare_data, furnizor, created_at, updated_at
		 FROM documents WHERE id = ?`, id,
	).Scan(
		&d.ID, &d.Nr, &d.Data, &d.Unitate, &d.DocumentLivrare, &d.DocumentLivrareNr,
		&d.DocumentLivrareData, &d.Furnizor, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Document{}, ErrNotFound
	}
	if err != nil {
		return model.Document{}, fmt.Errorf("citire document: %w", err)
	}

	if d.Randuri, err = s.randuri(id); err != nil {
		return model.Document{}, err
	}
	return d, nil
}

func (s *Store) randuri(documentID int64) ([]model.Rand, error) {
	rows, err := s.db.Query(
		`SELECT id, product_id, pozitie, denumire, um, cantitate, pret_fara_tva,
		        cota_tva, pret_vanzare
		 FROM document_rows WHERE document_id = ? ORDER BY pozitie, id`, documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("citire randuri: %w", err)
	}
	defer rows.Close()

	out := []model.Rand{}
	for rows.Next() {
		var r model.Rand
		var productID sql.NullInt64
		if err := rows.Scan(&r.ID, &productID, &r.Pozitie, &r.Denumire, &r.UM,
			&r.Cantitate, &r.PretFaraTVA, &r.CotaTVA, &r.PretVanzare); err != nil {
			return nil, fmt.Errorf("citire rand: %w", err)
		}
		if productID.Valid {
			id := productID.Int64
			r.ProductID = &id
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveDocument creates or updates a document and returns it as stored.
//
// Everything happens in one transaction: the document, its rows, the supplier
// suggestion and the number counter. There is no interleaving in which a save
// reports failure for a document that was in fact written, and none in which a
// document exists while the counter still points at its number.
func (s *Store) SaveDocument(doc model.Document) (model.Document, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return model.Document{}, fmt.Errorf("salvare document: %w", err)
	}
	defer tx.Rollback()

	acum := time.Now().UTC().Format(time.RFC3339)
	esteNou := doc.ID == 0

	if esteNou {
		doc.CreatedAt = acum
		doc.UpdatedAt = acum
		res, err := tx.Exec(
			`INSERT INTO documents (nr, data, unitate, document_livrare, document_livrare_nr,
			        document_livrare_data, furnizor, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			doc.Nr, doc.Data, doc.Unitate, doc.DocumentLivrare, doc.DocumentLivrareNr,
			doc.DocumentLivrareData, doc.Furnizor, doc.CreatedAt, doc.UpdatedAt,
		)
		if err != nil {
			return model.Document{}, fmt.Errorf("salvare document: %w", err)
		}
		if doc.ID, err = res.LastInsertId(); err != nil {
			return model.Document{}, fmt.Errorf("salvare document: %w", err)
		}
	} else {
		doc.UpdatedAt = acum
		if _, err := tx.Exec(
			`UPDATE documents SET nr = ?, data = ?, unitate = ?, document_livrare = ?,
			        document_livrare_nr = ?, document_livrare_data = ?, furnizor = ?,
			        updated_at = ?
			 WHERE id = ?`,
			doc.Nr, doc.Data, doc.Unitate, doc.DocumentLivrare, doc.DocumentLivrareNr,
			doc.DocumentLivrareData, doc.Furnizor, doc.UpdatedAt, doc.ID,
		); err != nil {
			return model.Document{}, fmt.Errorf("salvare document: %w", err)
		}
	}

	// The rows are replaced wholesale rather than diffed: they have no identity
	// of their own beyond their position, the form lets any of them be deleted
	// or reordered, and a document never holds enough of them for the rewrite
	// to cost anything.
	if _, err := tx.Exec(`DELETE FROM document_rows WHERE document_id = ?`, doc.ID); err != nil {
		return model.Document{}, fmt.Errorf("salvare randuri: %w", err)
	}
	for i := range doc.Randuri {
		r := &doc.Randuri[i]
		r.Pozitie = i
		var productID any
		if r.ProductID != nil {
			productID = *r.ProductID
		}
		res, err := tx.Exec(
			`INSERT INTO document_rows (document_id, product_id, pozitie, denumire, um,
			        cantitate, pret_fara_tva, cota_tva, pret_vanzare)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			doc.ID, productID, r.Pozitie, r.Denumire, r.UM, r.Cantitate,
			r.PretFaraTVA, r.CotaTVA, r.PretVanzare,
		)
		if err != nil {
			return model.Document{}, fmt.Errorf("salvare rand %d: %w", i+1, err)
		}
		if r.ID, err = res.LastInsertId(); err != nil {
			return model.Document{}, fmt.Errorf("salvare rand %d: %w", i+1, err)
		}
	}

	if err := retineFurnizor(tx, doc.Furnizor); err != nil {
		return model.Document{}, err
	}

	// The counter only ever moves forward, and only for a document that is
	// being created. Re-editing an old document must not drag the counter back
	// onto numbers already used, and saving one deliberately numbered below the
	// counter must not either.
	if esteNou {
		if _, err := tx.Exec(
			`UPDATE settings SET next_nr = ? WHERE id = 1 AND next_nr <= ?`,
			doc.Nr+1, doc.Nr,
		); err != nil {
			return model.Document{}, fmt.Errorf("actualizare numarator: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return model.Document{}, fmt.Errorf("salvare document: %w", err)
	}
	return doc, nil
}

// DeleteDocument removes a document and, through the foreign key, its rows.
func (s *Store) DeleteDocument(id int64) error {
	res, err := s.db.Exec(`DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("stergere document: %w", err)
	}
	afectate, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("stergere document: %w", err)
	}
	if afectate == 0 {
		return ErrNotFound
	}
	return nil
}
```

- [ ] **Step 4: Rulează testele ca să treacă**

Run: `go test ./internal/store/ -v`
Expected: PASS, toate testele din Task 3, 4, 5 și 6.

Dacă `TestDeleteDocumentSterageSiRandurile` cade cu rânduri rămase, cheia
străină nu e activă: verifică pragma `foreign_keys(1)` din `Open`.

- [ ] **Step 5: Commit**

```bash
git add internal/store
git commit -m "$(cat <<'EOF'
Store: documentele si randurile lor

Salvarea scrie documentul, randurile, sugestia de furnizor si urcarea
numaratorului intr-o singura tranzactie. Randurile sunt fotografii ale
produselor: editarea catalogului nu le atinge, iar stergerea unui produs
le lasa intregi, doar fara legatura.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 7: Randarea PDF (`internal/pdfdoc`)

**Files:**
- Create: `internal/pdfdoc/fold.go`, `internal/pdfdoc/pdf.go`
- Test: `internal/pdfdoc/pdf_test.go`

**Interfaces:**
- Consumes: `model.Document`, `calc.ValoriRand`, `calc.Totaluri`, `calc.Round2`.
- Produces:
  - `pdfdoc.Fold(s string) string`
  - `pdfdoc.Render(doc model.Document) ([]byte, error)`

Documentul poartă deja unitatea (`doc.Unitate`), deci `Render` nu are nevoie de
setări.

- [ ] **Step 1: Scrie testele căzute**

`internal/pdfdoc/pdf_test.go`:
```go
package pdfdoc

import (
	"bytes"
	"testing"

	"nota-de-receptie/internal/model"
)

func doc() model.Document {
	return model.Document{
		Nr:                  7,
		Data:                "2026-09-09",
		Unitate:             "S.C. Largiana Carn S.R.L.",
		DocumentLivrare:     "Factură",
		DocumentLivrareNr:   "1234",
		DocumentLivrareData: "2026-09-08",
		Furnizor:            "Alfa SRL",
		Randuri: []model.Rand{
			{Pozitie: 0, Denumire: "Pulpă fără os", UM: "Kg.", Cantitate: 10,
				PretFaraTVA: 20, CotaTVA: 11, PretVanzare: 30},
			{Pozitie: 1, Denumire: "Ouă", UM: "Buc.", Cantitate: 30,
				PretFaraTVA: 0.9, CotaTVA: 11, PretVanzare: 1.5},
		},
	}
}

func TestFold(t *testing.T) {
	got := Fold("Pulpă fără os, șuncă țărănească, ÎNTOCMIT")
	want := "Pulpa fara os, sunca taraneasca, INTOCMIT"
	if got != want {
		t.Errorf("Fold = %q, vrem %q", got, want)
	}
}

func TestRenderIntoarceUnPDF(t *testing.T) {
	data, err := Render(doc())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Errorf("iesirea nu incepe cu %%PDF: %q", data[:min(8, len(data))])
	}
	if len(data) < 1000 {
		t.Errorf("PDF-ul are doar %d octeti; pare gol", len(data))
	}
}

func TestRenderPeDocumentFaraRanduri(t *testing.T) {
	d := doc()
	d.Randuri = nil
	data, err := Render(d)
	if err != nil {
		t.Fatalf("Render pe document gol: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Error("iesirea nu e un PDF")
	}
}

func TestRenderPeMulteRanduriTreceInPaginaUrmatoare(t *testing.T) {
	d := doc()
	d.Randuri = nil
	for i := 0; i < 120; i++ {
		d.Randuri = append(d.Randuri, model.Rand{
			Pozitie: i, Denumire: "Produs", UM: "Kg.", Cantitate: 1,
			PretFaraTVA: 1, CotaTVA: 11, PretVanzare: 2,
		})
	}
	// Numarul de pagini se citeste de la fpdf, nu se ghiceste din octetii
	// fisierului: reprezentarea lor interna e treaba bibliotecii si se poate
	// schimba fara ca nimic sa fie in neregula.
	if n := construieste(d).PageNo(); n < 2 {
		t.Errorf("am construit %d pagini, vrem cel putin 2", n)
	}
	if _, err := Render(d); err != nil {
		t.Fatalf("Render: %v", err)
	}
}

func TestLatimileColoanelorUmpluPagina(t *testing.T) {
	var suma float64
	for _, w := range colWidths {
		suma += w
	}
	if suma != 277 {
		t.Errorf("suma latimilor = %v, vrem 277 (A4 landscape minus marginile)", suma)
	}
	if len(colWidths) != 9 {
		t.Errorf("len(colWidths) = %d, vrem 9 coloane", len(colWidths))
	}
}

func TestRandulTotalNuAreCantitate(t *testing.T) {
	// U/M variaza de la rand la rand, deci o suma peste cantitati ar fi falsa.
	got := randTotal(doc())
	if got[3] != "" {
		t.Errorf("celula de cantitate din TOTAL = %q, vrem goala", got[3])
	}
	if got[5] == "" || got[6] == "" || got[8] == "" {
		t.Errorf("TOTAL nu are valorile: %q", got)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `go test ./internal/pdfdoc/`
Expected: FAIL — `undefined: Fold`.

- [ ] **Step 3: Scrie `internal/pdfdoc/fold.go`**

```go
// Package pdfdoc renders a notă de recepție to a PDF mirroring the paper form.
package pdfdoc

import "strings"

// diacritics maps the Romanian letters to their ASCII equivalents. Both the
// comma-below (correct) and cedilla (legacy) forms of s and t are covered.
var diacritics = strings.NewReplacer(
	"ă", "a", "Ă", "A",
	"â", "a", "Â", "A",
	"î", "i", "Î", "I",
	"ș", "s", "Ș", "S",
	"ş", "s", "Ş", "S",
	"ț", "t", "Ț", "T",
	"ţ", "t", "Ţ", "T",
)

// Fold replaces Romanian diacritics with plain ASCII. The core PDF fonts are
// Latin-1, which cannot represent them, and the paper form is printed without
// them anyway.
func Fold(s string) string {
	return diacritics.Replace(s)
}
```

- [ ] **Step 4: Scrie `internal/pdfdoc/pdf.go`**

```go
package pdfdoc

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/go-pdf/fpdf"

	"nota-de-receptie/internal/calc"
	"nota-de-receptie/internal/model"
)

// colWidths are the product table's column widths in mm, summing to the 277mm
// printable width of a landscape A4 with 10mm margins. Landscape is what makes
// nine columns readable; portrait would squeeze the denumire to nothing.
var colWidths = []float64{13, 70, 16, 26, 28, 30, 30, 28, 36}

// livrareWidths are the delivery table's four columns, over the same 277mm.
var livrareWidths = []float64{80, 45, 52, 100}

const (
	marginLeft = 10.0
	rowHeight  = 5.5
)

// Render draws doc onto landscape A4 pages and returns the PDF bytes.
func Render(doc model.Document) ([]byte, error) {
	var buf bytes.Buffer
	if err := construieste(doc).Output(&buf); err != nil {
		return nil, fmt.Errorf("generare PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// construieste draws the whole document and hands back the fpdf document
// itself. Render only serialises it; keeping the two apart lets a test ask how
// many pages a document came to, which reading the serialised bytes cannot
// answer without depending on how fpdf lays them out.
func construieste(doc model.Document) *fpdf.Fpdf {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(marginLeft, 10, marginLeft)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	drawHeader(pdf, doc)
	drawLivrare(pdf, doc)
	pdf.Ln(6)
	drawProduse(pdf, doc)
	drawFooter(pdf)
	return pdf
}

func drawHeader(pdf *fpdf.Fpdf, doc model.Document) {
	pdf.SetFont("Arial", "B", 15)
	pdf.CellFormat(277, 8, "NOTA DE RECEPTIE", "", 1, "C", false, 0, "")

	pdf.Ln(2)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(150, 6, "UNITATEA: "+Fold(doc.Unitate), "", 0, "L", false, 0, "")
	pdf.CellFormat(60, 6, fmt.Sprintf("nr. %d", doc.Nr), "", 0, "L", false, 0, "")
	pdf.CellFormat(67, 6, "din "+formatDate(doc.Data), "", 1, "L", false, 0, "")
	pdf.Ln(3)
}

// drawLivrare draws the delivery table: header row, then the single row of
// values. Cod fiscal and Achitat cu are gone from the form, so four columns
// are all there is.
func drawLivrare(pdf *fpdf.Fpdf, doc model.Document) {
	header := []string{"DOCUMENT LIVRARE", "NR.", "DATA", "FURNIZORUL"}
	valori := []string{
		Fold(doc.DocumentLivrare),
		Fold(doc.DocumentLivrareNr),
		formatDate(doc.DocumentLivrareData),
		Fold(doc.Furnizor),
	}

	pdf.SetFont("Arial", "B", 9)
	pdf.SetX(marginLeft)
	for i, cell := range header {
		pdf.CellFormat(livrareWidths[i], rowHeight+1, cell, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 9)
	pdf.SetX(marginLeft)
	for i, cell := range valori {
		pdf.CellFormat(livrareWidths[i], rowHeight+1, cell, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
}

func headerRow() []string {
	return []string{
		"Nr. crt.", "DENUMIREA", "U/M", "Cantitatea", "Pret fara T.V.A.",
		"Valoare fara T.V.A.", "Valoare cu T.V.A.", "Pret de vanzare",
		"Valoare la pret de vanzare",
	}
}

// randCells renders one product row. The adaos is absent by design: it is a
// working figure for the screen, and the form it is printed on has no column
// for it.
func randCells(index int, r model.Rand) []string {
	v := calc.ValoriRand(r)
	return []string{
		strconv.Itoa(index + 1),
		Fold(r.Denumire),
		Fold(r.UM),
		num(r.Cantitate),
		num(r.PretFaraTVA),
		num(v.ValoareFaraTVA),
		num(v.ValoareCuTVA),
		num(r.PretVanzare),
		num(v.ValoareVanzare),
	}
}

// randTotal renders the TOTAL row. The quantity cell stays empty: U/M is
// "Buc." on one row and "Kg." on the next, so a sum over them would be a
// figure nobody could defend on a signed document.
func randTotal(doc model.Document) []string {
	t := calc.Totaluri(doc.Randuri)
	return []string{
		"", "TOTAL", "", "", "",
		num(t.ValoareFaraTVA), num(t.ValoareCuTVA), "", num(t.ValoareVanzare),
	}
}

func drawProduse(pdf *fpdf.Fpdf, doc model.Document) {
	ensureHeaderFits(pdf)
	drawTableHeader(pdf)

	pdf.SetFont("Arial", "", 8)
	for i, r := range doc.Randuri {
		ensureRowFits(pdf)
		pdf.SetFont("Arial", "", 8)
		pdf.SetX(marginLeft)
		for j, cell := range randCells(i, r) {
			pdf.CellFormat(colWidths[j], rowHeight, cell, "1", 0, align(j), false, 0, "")
		}
		pdf.Ln(-1)
	}

	ensureRowFits(pdf)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetX(marginLeft)
	for j, cell := range randTotal(doc) {
		pdf.CellFormat(colWidths[j], rowHeight, cell, "1", 0, align(j), false, 0, "")
	}
	pdf.Ln(-1)
}

// align keeps the denumire left and every figure right, which is how a column
// of numbers is read.
func align(col int) string {
	switch col {
	case 1:
		return "L"
	case 0, 2:
		return "C"
	default:
		return "R"
	}
}

// drawTableHeader draws one instance of the column header row. It is taller
// than a body row because several of the labels wrap onto two lines.
func drawTableHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 7)
	pdf.SetX(marginLeft)
	for i, cell := range headerRow() {
		pdf.CellFormat(colWidths[i], rowHeight*2, cell, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
}

// ensureRowFits forces a page break, and redraws the column header on the new
// page, if a single row would not fit above the bottom margin. It is a no-op
// while the current page still has room, so a one-page document is unaffected.
func ensureRowFits(pdf *fpdf.Fpdf) {
	if !incape(pdf, rowHeight) {
		pdf.AddPage()
		drawTableHeader(pdf)
	}
}

// ensureHeaderFits breaks the page if the header row itself would not fit. It
// does not draw the header — the caller does that immediately after — so it
// never duplicates it the way ensureRowFits' redraw would.
func ensureHeaderFits(pdf *fpdf.Fpdf) {
	if !incape(pdf, rowHeight*2) {
		pdf.AddPage()
	}
}

func incape(pdf *fpdf.Fpdf, inaltime float64) bool {
	_, pageHeight := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	return pdf.GetY()+inaltime <= pageHeight-bottom
}

// drawFooter draws the three signature labels. They carry no names: the form
// is signed by hand, so the app has no field for them.
func drawFooter(pdf *fpdf.Fpdf) {
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetX(marginLeft)
	pdf.CellFormat(92, 6, "COMISIA DE RECEPTIE,", "", 0, "C", false, 0, "")
	pdf.CellFormat(92, 6, "INTOCMIT,", "", 0, "C", false, 0, "")
	pdf.CellFormat(93, 6, "GESTIONAR,", "", 1, "C", false, 0, "")
}

// num renders a money or quantity value, leaving zero blank the way the paper
// form leaves unused cells empty.
func num(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(calc.Round2(v), 'f', 2, 64)
}

// formatDate turns an ISO date into the dd/mm/yyyy the form is written in. An
// empty or malformed value comes back as it went in — the delivery date is
// optional, and printing a placeholder date would be worse than printing none.
func formatDate(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "/" + iso[5:7] + "/" + iso[0:4]
}
```

- [ ] **Step 5: Rulează testele ca să treacă**

Run: `go test ./internal/pdfdoc/ -v`
Expected: PASS, șase teste.

- [ ] **Step 6: Uită-te la un PDF adevărat**

Adaugă temporar în `internal/pdfdoc/pdf_test.go` un test care scrie fișierul pe disc:

```go
func TestScriePDFDeVerificat(t *testing.T) {
	if testing.Short() {
		t.Skip("doar pentru verificare vizuala")
	}
	data, err := Render(doc())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if err := os.WriteFile("/tmp/nota.pdf", data, 0o644); err != nil {
		t.Fatalf("scriere: %v", err)
	}
	t.Log("scris /tmp/nota.pdf")
}
```

Run: `go test ./internal/pdfdoc/ -run TestScriePDFDeVerificat && open /tmp/nota.pdf`

Verifică ochiometric:
titlul centrat, tabelul de livrare cu patru coloane, cele nouă coloane de
produse fără Cod / T.V.A. deductibil / T.V.A. colectat / Adaos, celula de
cantitate goală pe rândul TOTAL, cele trei etichete de semnătură.

Șterge testul și importul `os` după verificare — nu se comite.

- [ ] **Step 7: Commit**

```bash
git add internal/pdfdoc
git commit -m "$(cat <<'EOF'
Randarea PDF

A4 landscape, noua coloane de produse pe 277mm, tabelul de livrare
redus la patru coloane si etichetele de semnatura la baza. Fara Cod,
fara T.V.A. deductibil/colectat, fara Adaos si fara total de cantitate.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 8: Legăturile Wails (`app.go`)

**Files:**
- Modify: `app.go` (înlocuiește scheletul din Task 1)
- Test: `app_test.go`

**Interfaces:**
- Consumes: tot ce produc sarcinile 1–7.
- Produces: metodele exportate de `App`, care devin funcțiile din
  `frontend/wailsjs/go/main/App`:
  - `GetSettings() (model.Settings, error)`, `SaveSettings(model.Settings) error`
  - `ListProducts() ([]model.Product, error)`, `SaveProducts([]model.Product) error`, `AddProduct(model.Product) (model.Product, error)`
  - `ListFurnizori() ([]string, error)`, `DeleteFurnizor(string) error`
  - `ListDocuments() ([]model.DocumentSummary, error)`, `GetDocument(int64) (model.Document, error)`
  - `NewDocumentDraft() (model.Document, error)`, `SaveDocument(model.Document) (model.Document, error)`, `DeleteDocument(int64) error`
  - `ExportPDF(int64) (string, error)`

- [ ] **Step 1: Scrie testele căzute**

`app_test.go`:
```go
package main

import (
	"path/filepath"
	"testing"

	"nota-de-receptie/internal/model"
	"nota-de-receptie/internal/store"
)

// appDeTest builds an App on a throwaway database. ExportPDF is left out of
// these tests: it opens a native save dialog, which needs a running Wails
// context.
func appDeTest(t *testing.T) *App {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return &App{store: s}
}

func TestNewDocumentDraft(t *testing.T) {
	a := appDeTest(t)
	draft, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if draft.ID != 0 {
		t.Errorf("ID = %d, vrem 0 pe o schita nesalvata", draft.ID)
	}
	if draft.Nr != 1 {
		t.Errorf("Nr = %d, vrem 1 pe prima instalare", draft.Nr)
	}
	if draft.Unitate != "S.C. Largiana Carn S.R.L." {
		t.Errorf("Unitate = %q; schita nu a preluat-o din setari", draft.Unitate)
	}
	if len(draft.Data) != 10 {
		t.Errorf("Data = %q, vrem o data ISO", draft.Data)
	}
	if draft.Randuri == nil {
		t.Error("Randuri = nil; vrem o lista goala, ca frontendul sa nu vada null")
	}
	if len(draft.Randuri) != 0 {
		t.Errorf("len(Randuri) = %d, vrem 0: nota de receptie porneste goala", len(draft.Randuri))
	}
}

func TestSchitaUrmeazaContorul(t *testing.T) {
	a := appDeTest(t)
	primul, _ := a.NewDocumentDraft()
	primul.Randuri = []model.Rand{
		{Denumire: "X", UM: "Kg.", Cantitate: 1, PretFaraTVA: 10, CotaTVA: 11, PretVanzare: 15},
	}
	if _, err := a.SaveDocument(primul); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	alDoilea, err := a.NewDocumentDraft()
	if err != nil {
		t.Fatalf("NewDocumentDraft: %v", err)
	}
	if alDoilea.Nr != 2 {
		t.Errorf("Nr = %d pe a doua schita, vrem 2", alDoilea.Nr)
	}
}

func TestAddProductDinFormular(t *testing.T) {
	a := appDeTest(t)
	p, err := a.AddProduct(model.Product{Denumire: "Costita", UM: "Kg.", PretVanzare: 29, CotaTVA: 11})
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if p.ID == 0 {
		t.Error("produsul nu a primit id")
	}
	produse, _ := a.ListProducts()
	if len(produse) != 1 {
		t.Errorf("len = %d, vrem 1", len(produse))
	}
}

func TestListeleIntorcSliceGolNuNil(t *testing.T) {
	// Un nil devine null in JSON, iar frontendul ar cadea pe .map().
	a := appDeTest(t)
	documente, err := a.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if documente == nil {
		t.Error("ListDocuments = nil, vrem []")
	}
	produse, _ := a.ListProducts()
	if produse == nil {
		t.Error("ListProducts = nil, vrem []")
	}
	furnizori, _ := a.ListFurnizori()
	if furnizori == nil {
		t.Error("ListFurnizori = nil, vrem []")
	}
}

func TestSaveSiSterge(t *testing.T) {
	a := appDeTest(t)
	draft, _ := a.NewDocumentDraft()
	draft.Furnizor = "Alfa SRL"
	draft.Randuri = []model.Rand{
		{Denumire: "X", UM: "Kg.", Cantitate: 2, PretFaraTVA: 10, CotaTVA: 11, PretVanzare: 15},
	}
	salvat, err := a.SaveDocument(draft)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	got, err := a.GetDocument(salvat.ID)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Furnizor != "Alfa SRL" {
		t.Errorf("Furnizor = %q", got.Furnizor)
	}

	if err := a.DeleteDocument(salvat.ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	lista, _ := a.ListDocuments()
	if len(lista) != 0 {
		t.Errorf("len = %d dupa stergere, vrem 0", len(lista))
	}
}
```

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `go test .`
Expected: FAIL — `unknown field store in struct literal`.

- [ ] **Step 3: Scrie `app.go`**

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/browser"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"nota-de-receptie/internal/appdir"
	"nota-de-receptie/internal/model"
	"nota-de-receptie/internal/pdfdoc"
	"nota-de-receptie/internal/store"
)

// App is the Wails-bound application object. Every exported method here is
// callable from the frontend.
type App struct {
	ctx   context.Context
	store *store.Store
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup opens the database and keeps the Wails context for runtime calls.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	path, err := appdir.DBPath()
	if err != nil {
		panic("nu s-a putut determina locatia bazei de date: " + err.Error())
	}
	s, err := store.Open(path)
	if err != nil {
		panic("nu s-a putut deschide baza de date: " + err.Error())
	}
	a.store = s
}

// shutdown closes the database.
func (a *App) shutdown(ctx context.Context) {
	if a.store != nil {
		a.store.Close()
	}
}

// GetSettings returns the application settings.
func (a *App) GetSettings() (model.Settings, error) { return a.store.GetSettings() }

// SaveSettings overwrites the application settings.
func (a *App) SaveSettings(s model.Settings) error { return a.store.SaveSettings(s) }

// ListProducts returns the product catalogue in display order.
func (a *App) ListProducts() ([]model.Product, error) { return a.store.ListProducts() }

// SaveProducts brings the catalogue in line with the list from Setări.
func (a *App) SaveProducts(produse []model.Product) error { return a.store.SaveProducts(produse) }

// AddProduct files one product without leaving the document form.
func (a *App) AddProduct(p model.Product) (model.Product, error) { return a.store.AddProduct(p) }

// ListFurnizori returns the remembered supplier names.
func (a *App) ListFurnizori() ([]string, error) { return a.store.ListFurnizori() }

// DeleteFurnizor forgets one supplier suggestion.
func (a *App) DeleteFurnizor(nume string) error { return a.store.DeleteFurnizor(nume) }

// ListDocuments returns the sidebar history, newest first.
func (a *App) ListDocuments() ([]model.DocumentSummary, error) { return a.store.ListDocuments() }

// GetDocument loads one saved document.
func (a *App) GetDocument(id int64) (model.Document, error) { return a.store.GetDocument(id) }

// DeleteDocument removes a document and its rows.
func (a *App) DeleteDocument(id int64) error { return a.store.DeleteDocument(id) }

// SaveDocument creates or updates a document.
func (a *App) SaveDocument(doc model.Document) (model.Document, error) {
	return a.store.SaveDocument(doc)
}

// NewDocumentDraft builds an unsaved reception: the next number, today's date
// and the unit from the settings.
//
// The product table starts empty. A notă de recepție records what actually
// arrived, which is a handful of items out of a catalogue that may hold
// hundreds — prefilling it would mean deleting more rows than filling in.
func (a *App) NewDocumentDraft() (model.Document, error) {
	settings, err := a.store.GetSettings()
	if err != nil {
		return model.Document{}, err
	}
	return model.Document{
		Nr:      settings.NextNr,
		Data:    time.Now().Format("2006-01-02"),
		Unitate: settings.UnitateNume,
		// An empty slice, not nil: nil marshals to null, and the form would
		// then have nothing to append a row to.
		Randuri: []model.Rand{},
	}, nil
}

// ExportPDF renders a saved document, asks the user where to put the PDF and
// opens it with the system default handler. It returns the saved path, or an
// empty string when the user cancels the dialog.
func (a *App) ExportPDF(id int64) (string, error) {
	doc, err := a.store.GetDocument(id)
	if err != nil {
		return "", err
	}

	data, err := pdfdoc.Render(doc)
	if err != nil {
		return "", err
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvează PDF",
		DefaultFilename: fmt.Sprintf("nota-de-receptie-%d-%s.pdf", doc.Nr, doc.Data),
		Filters: []runtime.FileFilter{
			{DisplayName: "Fișiere PDF (*.pdf)", Pattern: "*.pdf"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("alegere fișier: %w", err)
	}
	if path == "" {
		return "", nil // user cancelled
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("scriere fișier PDF: %w", err)
	}
	if err := browser.OpenFile(path); err != nil {
		// The file is on disk; failing to open the viewer is not fatal.
		runtime.LogWarningf(a.ctx, "nu s-a putut deschide PDF-ul: %v", err)
	}
	return path, nil
}
```

- [ ] **Step 4: Rulează testele ca să treacă**

Run: `go test ./... `
Expected: PASS pe toate pachetele.

- [ ] **Step 5: Generează legăturile TypeScript**

Run:
```bash
wails generate module
```
Expected: apare `frontend/wailsjs/go/main/App.d.ts`, `App.js` și
`frontend/wailsjs/go/models.ts`, cu toate cele treisprezece metode și cu
tipurile din `model`. Verifică:

```bash
grep -c "export function" frontend/wailsjs/go/main/App.js
grep "class Document\|class Product\|class Rand\|class Settings" frontend/wailsjs/go/models.ts
```
Expected: 13 funcții; cele patru clase prezente.

Dacă `wails generate module` cere un frontend care încă nu există, sari peste
pasul acesta și repetă-l la începutul Task 9, după ce `frontend/package.json`
e pe disc.

- [ ] **Step 6: Commit**

```bash
git add app.go app_test.go frontend/wailsjs
git commit -m "$(cat <<'EOF'
Legaturile Wails

Cele treisprezece metode chemate din interfata, plus exportul PDF care
intreaba unde sa salveze si deschide fisierul. Schita de document nou
porneste cu tabelul de produse gol.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 9: Schelet de frontend

**Files:**
- Create: `frontend/package.json`, `frontend/tsconfig.json`, `frontend/vite.config.ts`, `frontend/index.html`
- Create: `frontend/src/main.ts`, `frontend/src/router.ts`, `frontend/src/dialog.ts`, `frontend/src/toast.ts`, `frontend/src/format.ts`, `frontend/src/style.css`
- Test: `frontend/src/format.test.ts`

**Interfaces:**
- Consumes: legăturile generate în Task 8.
- Produces:
  - `router.ts`: `navigate(hash: string): void`, `startRouter(routes: Route[], outlet: HTMLElement): void`, tipul `Route = { pattern: RegExp; render: (outlet: HTMLElement, ...params: string[]) => void | Promise<void> }`
  - `dialog.ts`: `showConfirm(message: string): Promise<boolean>`, `showAlert(message: string): Promise<void>`
  - `toast.ts`: `showToast(message: string): void`
  - `format.ts`: `parseNumber(input: string): number`, `formatNumber(value: number, decimals?: number): string`, `formatLei(value: number): string`, `formatProcent(value: number): string`, `formatDateRO(iso: string): string`, `parseDateRO(input: string): string | undefined`
  - `main.ts`: `refreshSidebar(): Promise<void>`

Aplicația soră `/Users/roxanasuciu/git/proces-verbal-transare` are aceste
fișiere într-o formă aproape identică. `router.ts`, `dialog.ts` și `toast.ts` se
copiază de acolo neschimbate — sunt generice, testate în producție, și nu au
nimic legat de procesul verbal în ele. `format.ts` se copiază și se completează
cu `formatLei` și `formatProcent`. `style.css` se copiază și se curăță de
regulile pentru șabloane și marjă (`.template-*`, `.marja*`), apoi se adaugă
regulile noi din Step 6.

- [ ] **Step 1: Scrie `frontend/package.json`, `tsconfig.json`, `vite.config.ts`, `index.html`**

`frontend/package.json`:
```json
{
  "name": "frontend",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "test": "vitest run"
  },
  "devDependencies": {
    "typescript": "^5.4.0",
    "vite": "^8.2.2",
    "vitest": "^5.0.0"
  }
}
```

`frontend/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "strict": true,
    "types": ["vite/client"]
  },
  "include": ["src"]
}
```

`frontend/vite.config.ts`:
```ts
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: { css: true },
});
```

`frontend/index.html`:
```html
<!doctype html>
<html lang="ro">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Notă de recepție</title>
</head>
<body>
<div id="app"></div>
<script type="module" src="/src/main.ts"></script>
</body>
</html>
```

Apoi: `cd frontend && npm install`.

- [ ] **Step 2: Copiază fișierele generice din aplicația soră**

```bash
REF=/Users/roxanasuciu/git/proces-verbal-transare/frontend/src
cp "$REF/router.ts" "$REF/dialog.ts" "$REF/toast.ts" "$REF/format.ts" "$REF/style.css" frontend/src/
```

Verifică apoi că niciunul dintre fișierele copiate nu pomenește șabloane sau
gestiuni — sunt noțiuni ale aplicației surori, nu ale acesteia:

```bash
grep -rn "sablon\|șablon\|gestiune\|marja\|marjă" frontend/src/
```
Expected: nimic în `router.ts`, `dialog.ts`, `toast.ts`, `format.ts`. Ce apare
în `style.css` se șterge la Step 6.

- [ ] **Step 3: Scrie testele căzute pentru `format.ts`**

`frontend/src/format.test.ts` — copiază testele existente din aplicația soră
(`$REF/format.test.ts`) și adaugă:

```ts
import { describe, expect, it } from 'vitest';
import { formatLei, formatProcent } from './format';

describe('formatLei', () => {
  it('scrie doua zecimale cu virgula', () => {
    expect(formatLei(78)).toBe('78,00');
    expect(formatLei(1234.5)).toBe('1.234,50');
    expect(formatLei(-11)).toBe('-11,00');
  });

  it('scrie zero ca zero, nu ca gol', () => {
    expect(formatLei(0)).toBe('0,00');
  });
});

describe('formatProcent', () => {
  it('adauga semnul procentului', () => {
    expect(formatProcent(35.14)).toBe('35,14 %');
  });

  it('scrie liniuta cand procentul nu se poate calcula', () => {
    expect(formatProcent(undefined)).toBe('—');
  });
});
```

- [ ] **Step 4: Rulează testele ca să cadă**

Run: `cd frontend && npm test`
Expected: FAIL — `formatLei is not a function`.

- [ ] **Step 5: Completează `frontend/src/format.ts`**

Adaugă la fișierul copiat:

```ts
/**
 * Renders a money amount the Romanian way: two decimals, a comma for the
 * decimal separator and a dot grouping the thousands.
 *
 * Zero is written out rather than left blank. On screen an empty cell reads as
 * "not filled in yet", and a genuine zero — goods received for nothing — has to
 * be distinguishable from that. The PDF makes the opposite choice, for the
 * opposite reason: an empty cell on paper is how the form is meant to look.
 */
export function formatLei(value: number): string {
  const [intreg, zecimale] = Math.abs(value).toFixed(2).split('.');
  const grupat = intreg.replace(/\B(?=(\d{3})+$)/g, '.');
  return `${value < 0 ? '-' : ''}${grupat},${zecimale}`;
}

/**
 * Renders a markup percentage, or a dash when there is none to render — the
 * purchase value was zero, so the percentage has no value. A zero there would
 * read as "no markup", which is a different and wrong statement.
 */
export function formatProcent(value: number | undefined): string {
  if (value === undefined) return '—';
  return `${value.toFixed(2).replace('.', ',')} %`;
}
```

Gruparea miilor se face pe partea întreagă separat de zecimale, tocmai ca
regula `(\d{3})+$` să nu poată prinde și zecimalele. `Intl.NumberFormat('ro-RO')`
ar fi fost alternativa, dar separatorul de mii pe care îl produce diferă între
versiuni de Node, iar testele ar fi devenit dependente de mediu.

- [ ] **Step 6: Adaugă regulile CSS noi**

La finalul lui `frontend/src/style.css`, după ce ai șters regulile
`.template-*` și `.marja*`:

```css
/* Panoul de totaluri: etichetele deasupra valorilor, ca sa se citeasca
   pe verticala, nu ca un rand pierdut printre coloanele tabelului. */
.totaluri {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 1px;
  margin: 18px 0 24px;
  background: var(--border);
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
}

.totaluri .celula {
  background: var(--panel);
  padding: 10px 12px;
}

.totaluri .eticheta {
  display: block;
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 4px;
}

.totaluri .valoare {
  display: block;
  font-size: 18px;
  font-variant-numeric: tabular-nums;
}

.totaluri .valoare.negativ {
  color: var(--danger);
}

/* Caseta de cautare din coloana Denumirea. */
.combo {
  position: relative;
}

.combo-lista {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  z-index: 20;
  margin: 2px 0 0;
  padding: 0;
  list-style: none;
  max-height: 220px;
  overflow-y: auto;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgb(0 0 0 / 35%);
}

.combo-lista li {
  padding: 6px 10px;
  cursor: pointer;
}

.combo-lista li:hover,
.combo-lista li.evidentiat {
  background: var(--accent);
  color: #fff;
}

.combo-lista .produs-um {
  float: right;
  opacity: 0.7;
  font-size: 12px;
}

.combo-nou {
  padding: 6px 10px;
  border-top: 1px solid var(--border);
  font-size: 12px;
  color: var(--muted);
}

/* Coloanele read-only ale tabelului de produse. */
td.derivat {
  text-align: right;
  font-variant-numeric: tabular-nums;
  color: var(--muted);
  padding-right: 10px;
}

td.derivat.negativ {
  color: var(--danger);
}
```

Dacă variabilele `--panel`, `--muted`, `--danger`, `--accent`, `--border` nu
există în `:root`-ul copiat, adaugă-le cu valorile din aplicația soră.

- [ ] **Step 7: Scrie `frontend/src/main.ts`**

```ts
import './style.css';
import { renderSidebar } from './sidebar';
import { startRouter } from './router';
import { renderDocumentView } from './views/document';
import { renderSetariView } from './views/setari';

document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
  <aside class="sidebar" id="sidebar"></aside>
  <main class="main" id="outlet"></main>
`;

const sidebar = document.getElementById('sidebar') as HTMLElement;
const outlet = document.getElementById('outlet') as HTMLElement;

/** Re-reads the document history; called after any save or delete. */
export function refreshSidebar(): Promise<void> {
  return renderSidebar(sidebar);
}

void refreshSidebar();

startRouter(
  [
    {
      pattern: /^#\/document\/new$/,
      render: (el) => renderDocumentView(el, undefined, refreshSidebar),
    },
    {
      pattern: /^#\/document\/(\d+)$/,
      render: (el, id) => renderDocumentView(el, id, refreshSidebar),
    },
    {
      pattern: /^#\/setari$/,
      render: (el) => renderSetariView(el, refreshSidebar),
    },
  ],
  outlet,
);
```

`main.ts` importă module scrise în sarcinile 11–13, deci `npm run build` nu va
trece încă. Asta e în regulă: `npm test` rulează doar testele, care nu ating
`main.ts`.

- [ ] **Step 8: Rulează testele ca să treacă**

Run: `cd frontend && npm test`
Expected: PASS — testele pentru `format.ts`, inclusiv cele noi.

- [ ] **Step 9: Commit**

```bash
git add frontend
git commit -m "$(cat <<'EOF'
Schelet de frontend

Configuratia Vite/TypeScript si modulele generice preluate din
aplicatia sora: router pe hash, dialoguri desenate de pagina, toast si
formatarea numerelor si datelor. In plus, formatLei si formatProcent,
care scriu liniuta cand procentul nu se poate calcula.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 10: Aritmetica și căutarea pe partea de frontend

**Files:**
- Create: `frontend/src/calc.ts`, `frontend/src/fuzzy.ts`
- Test: `frontend/src/calc.test.ts`, `frontend/src/fuzzy.test.ts`

**Interfaces:**
- Consumes: nimic din Wails — ambele module sunt pure, ca să poată fi testate în node, fără DOM.
- Produces:
  - `calc.ts`: `round2(v: number): number`; tipul `RandCalculabil = { cantitate: number; pretFaraTva: number; cotaTva: number; pretVanzare: number }`; tipul `Valori = { valoareFaraTva: number; valoareCuTva: number; valoareVanzare: number; adaos: number; adaosProcent: number | undefined }`; `valoriRand(r: RandCalculabil): Valori`; `totaluri(randuri: RandCalculabil[]): Valori`
  - `fuzzy.ts`: `normalizeaza(s: string): string`; `scorFuzzy(text: string, cautare: string): number | undefined`; `cautaProduse<T extends { denumire: string }>(produse: T[], cautare: string, limita?: number): T[]`

`calc.ts` este oglinda lui `internal/calc`. Testele lui folosesc **exact
aceleași exemple** ca `internal/calc/calc_test.go`, ca cele două să nu poată
devia una de alta fără ca un test să cadă.

- [ ] **Step 1: Scrie testele căzute pentru `calc.ts`**

`frontend/src/calc.test.ts`:
```ts
import { describe, expect, it } from 'vitest';
import { round2, totaluri, valoriRand } from './calc';

describe('round2', () => {
  it('rotunjeste la doua zecimale, jumatate departandu-se de zero', () => {
    expect(round2(1.005)).toBe(1.01);
    expect(round2(2.344)).toBe(2.34);
    expect(round2(2.345)).toBe(2.35);
    expect(round2(-2.345)).toBe(-2.35);
    expect(round2(0)).toBe(0);
  });
});

describe('valoriRand', () => {
  it('calculeaza exemplul din specificatie', () => {
    const v = valoriRand({ cantitate: 10, pretFaraTva: 20, cotaTva: 11, pretVanzare: 30 });
    expect(v.valoareFaraTva).toBe(200);
    expect(v.valoareCuTva).toBe(222);
    expect(v.valoareVanzare).toBe(300);
    expect(v.adaos).toBe(78);
    expect(v.adaosProcent).toBe(35.14);
  });

  it('da adaos negativ cand se vinde sub pretul de achizitie', () => {
    const v = valoriRand({ cantitate: 1, pretFaraTva: 100, cotaTva: 11, pretVanzare: 100 });
    expect(v.adaos).toBe(-11);
    expect(v.adaosProcent).toBe(-9.91);
  });

  it('nu da procent cand valoarea cu TVA e zero', () => {
    const v = valoriRand({ cantitate: 5, pretFaraTva: 0, cotaTva: 11, pretVanzare: 4 });
    expect(v.valoareCuTva).toBe(0);
    expect(v.adaos).toBe(20);
    expect(v.adaosProcent).toBeUndefined();
  });

  it('aplica cota fiecarui rand in parte', () => {
    const v = valoriRand({ cantitate: 2, pretFaraTva: 50, cotaTva: 21, pretVanzare: 70 });
    expect(v.valoareCuTva).toBe(121);
    expect(v.adaos).toBe(19);
  });
});

describe('totaluri', () => {
  it('insumeaza randurile si calculeaza procentul din totaluri', () => {
    const t = totaluri([
      { cantitate: 10, pretFaraTva: 20, cotaTva: 11, pretVanzare: 30 },
      { cantitate: 2, pretFaraTva: 50, cotaTva: 21, pretVanzare: 70 },
    ]);
    expect(t.valoareFaraTva).toBe(300);
    expect(t.valoareCuTva).toBe(343);
    expect(t.valoareVanzare).toBe(440);
    expect(t.adaos).toBe(97);
    // Din totaluri, nu media procentelor de pe randuri (35,14 si 15,70).
    expect(t.adaosProcent).toBe(28.28);
  });

  it('da zero si niciun procent pe lista goala', () => {
    const t = totaluri([]);
    expect(t.valoareCuTva).toBe(0);
    expect(t.adaosProcent).toBeUndefined();
  });
});
```

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `cd frontend && npm test`
Expected: FAIL — nu există `./calc`.

- [ ] **Step 3: Scrie `frontend/src/calc.ts`**

```ts
/**
 * The arithmetic of the notă de recepție, mirroring internal/calc on the Go
 * side. Both exist because the form recalculates on every keystroke, which
 * cannot go through Wails, while the PDF and the store must not depend on the
 * browser. calc.test.ts uses the same worked examples as calc_test.go, so the
 * two cannot drift apart without a test failing.
 */

/** One row's inputs. */
export interface RandCalculabil {
  cantitate: number;
  pretFaraTva: number;
  cotaTva: number;
  pretVanzare: number;
}

/** What a row, or a whole document, comes to. */
export interface Valori {
  valoareFaraTva: number;
  valoareCuTva: number;
  valoareVanzare: number;
  adaos: number;
  /**
   * Undefined when the purchase value is zero: the percentage has no value
   * then, and a zero would read as "no markup", which is a different claim.
   */
  adaosProcent: number | undefined;
}

/** Rounds to two decimals, half away from zero. */
export function round2(v: number): number {
  const r = Math.round(Math.abs(v) * 100) / 100;
  return v < 0 ? -r : r;
}

/** Works out one row's values. */
export function valoriRand(r: RandCalculabil): Valori {
  const faraTva = round2(r.cantitate * r.pretFaraTva);
  const cuTva = round2(faraTva * (1 + r.cotaTva / 100));
  const vanzare = round2(r.cantitate * r.pretVanzare);
  return valoriDin(faraTva, cuTva, vanzare);
}

/**
 * Sums a document's rows. Quantity is deliberately absent: U/M is "Buc." on
 * one row and "Kg." on the next, so a sum over them would mean nothing.
 */
export function totaluri(randuri: RandCalculabil[]): Valori {
  let faraTva = 0;
  let cuTva = 0;
  let vanzare = 0;
  for (const r of randuri) {
    const v = valoriRand(r);
    faraTva += v.valoareFaraTva;
    cuTva += v.valoareCuTva;
    vanzare += v.valoareVanzare;
  }
  return valoriDin(round2(faraTva), round2(cuTva), round2(vanzare));
}

/**
 * Derives the adaos and its percentage from three already-rounded values.
 * Both valoriRand and totaluri end here, so a row and a total can never
 * disagree about what an adaos is.
 */
function valoriDin(valoareFaraTva: number, valoareCuTva: number, valoareVanzare: number): Valori {
  const adaos = round2(valoareVanzare - valoareCuTva);
  return {
    valoareFaraTva,
    valoareCuTva,
    valoareVanzare,
    adaos,
    adaosProcent: valoareCuTva === 0 ? undefined : round2((adaos / valoareCuTva) * 100),
  };
}
```

- [ ] **Step 4: Scrie testele căzute pentru `fuzzy.ts`**

`frontend/src/fuzzy.test.ts`:
```ts
import { describe, expect, it } from 'vitest';
import { cautaProduse, normalizeaza, scorFuzzy } from './fuzzy';

const produse = [
  { denumire: 'Pulpă fără os' },
  { denumire: 'Pulpă cu os' },
  { denumire: 'Ceafă fără os' },
  { denumire: 'Cotlet cu os' },
  { denumire: 'Ouă' },
];

describe('normalizeaza', () => {
  it('scoate diacriticele si majusculele', () => {
    expect(normalizeaza('Pulpă Fără Os')).toBe('pulpa fara os');
    expect(normalizeaza('ȘUNCĂ ȚĂRĂNEASCĂ')).toBe('sunca taraneasca');
  });
});

describe('scorFuzzy', () => {
  it('nu potriveste ce nu e subsecventa', () => {
    expect(scorFuzzy('pulpa fara os', 'zzz')).toBeUndefined();
  });

  it('potriveste literele in ordine, chiar daca sar peste altele', () => {
    expect(scorFuzzy('pulpa fara os', 'pfo')).toBeDefined();
  });

  it('da scor mai bun potrivirii de la inceput decat celei din mijloc', () => {
    const laInceput = scorFuzzy('pulpa fara os', 'pulpa')!;
    const inMijloc = scorFuzzy('ceafa pulpa', 'pulpa')!;
    expect(laInceput).toBeLessThan(inMijloc);
  });

  it('da scor mai bun potrivirii compacte decat celei imprastiate', () => {
    const compact = scorFuzzy('pulpa fara os', 'pul')!;
    const imprastiat = scorFuzzy('pulpa fara os', 'pas')!;
    expect(compact).toBeLessThan(imprastiat);
  });
});

describe('cautaProduse', () => {
  it('intoarce catalogul in ordinea lui cand cautarea e goala', () => {
    expect(cautaProduse(produse, '').map((p) => p.denumire)).toEqual(
      produse.map((p) => p.denumire),
    );
  });

  it('gaseste fara diacritice ce e scris cu diacritice', () => {
    const gasite = cautaProduse(produse, 'pulpa fara');
    expect(gasite[0].denumire).toBe('Pulpă fără os');
  });

  it('gaseste dupa initiale', () => {
    const gasite = cautaProduse(produse, 'cfo');
    expect(gasite[0].denumire).toBe('Ceafă fără os');
  });

  it('nu intoarce nimic pentru o cautare fara potriviri', () => {
    expect(cautaProduse(produse, 'zzzz')).toEqual([]);
  });

  it('respecta limita', () => {
    expect(cautaProduse(produse, 'o', 2)).toHaveLength(2);
  });
});
```

- [ ] **Step 5: Rulează testele ca să cadă**

Run: `cd frontend && npm test`
Expected: FAIL — nu există `./fuzzy`.

- [ ] **Step 6: Scrie `frontend/src/fuzzy.ts`**

```ts
/**
 * Searching the product catalogue from the document form.
 *
 * The match is a subsequence match, not a substring one: the catalogue is
 * typed in Romanian with diacritics but searched on a keyboard where they are
 * awkward, and people type initials ("cfo" for "Ceafă fără os") as often as
 * they type prefixes. Both have to find the row.
 */

const DIACRITICE: Record<string, string> = {
  ă: 'a', â: 'a', î: 'i', ș: 's', ş: 's', ț: 't', ţ: 't',
};

/** Lowercases and strips Romanian diacritics, so search ignores both. */
export function normalizeaza(s: string): string {
  return s
    .toLowerCase()
    .replace(/[ăâîșşțţ]/g, (ch) => DIACRITICE[ch] ?? ch)
    .replace(/\s+/g, ' ')
    .trim();
}

/**
 * How well `cautare` matches `text`, as a penalty: smaller is better, and
 * undefined means it does not match at all.
 *
 * The penalty is where the match starts plus how far it had to jump between
 * letters. That ranks a prefix above a match buried in the middle, and a
 * compact match above one scattered across the name — which is the order
 * someone scanning the list expects.
 *
 * Both arguments are normalised here rather than by the caller, so a caller
 * cannot forget to.
 */
export function scorFuzzy(text: string, cautare: string): number | undefined {
  const t = normalizeaza(text);
  const c = normalizeaza(cautare);
  if (c === '') return 0;

  let inceput = -1;
  let ultima = -1;
  let salturi = 0;
  let i = 0;

  for (const litera of c) {
    const gasit = t.indexOf(litera, ultima + 1);
    if (gasit === -1) return undefined;
    if (i === 0) {
      inceput = gasit;
    } else {
      salturi += gasit - ultima - 1;
    }
    ultima = gasit;
    i += 1;
  }
  return inceput * 2 + salturi;
}

/**
 * The products matching `cautare`, best first.
 *
 * An empty search returns the catalogue in its own order — the list the user
 * arranged in Setări — rather than an alphabetised one, so the dropdown opens
 * on something familiar.
 */
export function cautaProduse<T extends { denumire: string }>(
  produse: T[],
  cautare: string,
  limita = 8,
): T[] {
  if (normalizeaza(cautare) === '') return produse.slice(0, limita);

  return produse
    .map((produs) => ({ produs, scor: scorFuzzy(produs.denumire, cautare) }))
    .filter((p): p is { produs: T; scor: number } => p.scor !== undefined)
    .sort((a, b) => a.scor - b.scor || a.produs.denumire.localeCompare(b.produs.denumire, 'ro'))
    .slice(0, limita)
    .map((p) => p.produs);
}
```

- [ ] **Step 7: Rulează testele ca să treacă**

Run: `cd frontend && npm test`
Expected: PASS — `calc.test.ts`, `fuzzy.test.ts`, `format.test.ts`.

Dacă testul „scor mai bun potrivirii compacte" cade, verifică formula: `pul`
pornește de la 0 fără salturi (scor 0), iar `pas` pornește de la 0 cu salturi
(scor > 0).

- [ ] **Step 8: Verifică oglinda cu partea de Go**

Run: `go test ./internal/calc/ && cd frontend && npm test -- calc`
Expected: ambele PASS, pe aceleași cifre (200 / 222 / 300 / 78 / 35,14 și
totalurile 300 / 343 / 440 / 97 / 28,28).

- [ ] **Step 9: Commit**

```bash
git add frontend/src
git commit -m "$(cat <<'EOF'
Aritmetica si cautarea pe partea de frontend

calc.ts oglindeste internal/calc pe aceleasi exemple, ca recalculul din
formular sa nu poata devia de la ce salveaza si tipareste Go.
fuzzy.ts cauta in catalog dupa subsecventa, fara diacritice, ca sa
gaseasca si dupa initiale.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 11: Puntea către Go și bara laterală

**Files:**
- Create: `frontend/src/api.ts`, `frontend/src/sidebar.ts`
- Test: `frontend/src/sidebar.test.ts`

**Interfaces:**
- Consumes: `frontend/wailsjs/go/main/App` și `frontend/wailsjs/go/models` (generate în Task 8), `dialog.ts`, `format.ts`, `router.ts`.
- Produces:
  - `api.ts`: reexportă cele treisprezece funcții; tipurile `Document`, `DocumentSummary`, `Rand`, `Product`, `Settings`; `showError(prefix: string, err: unknown): void`
  - `sidebar.ts`: `renderSidebar(el: HTMLElement): Promise<void>`, `escapeHtml(value: string): string`, `currentHash(hash?: string): string`, constanta `DRAFT_HASH = '#/document/new'`

- [ ] **Step 1: Scrie `frontend/src/api.ts`**

```ts
import {
  AddProduct,
  DeleteDocument,
  DeleteFurnizor,
  ExportPDF,
  GetDocument,
  GetSettings,
  ListDocuments,
  ListFurnizori,
  ListProducts,
  NewDocumentDraft,
  SaveDocument,
  SaveProducts,
  SaveSettings,
} from '../wailsjs/go/main/App';
import { model } from '../wailsjs/go/models';
import { showAlert } from './dialog';

export type Document = model.Document;
export type DocumentSummary = model.DocumentSummary;
export type Rand = model.Rand;
export type Product = model.Product;
export type Settings = model.Settings;

export {
  AddProduct,
  DeleteDocument,
  DeleteFurnizor,
  ExportPDF,
  GetDocument,
  GetSettings,
  ListDocuments,
  ListFurnizori,
  ListProducts,
  NewDocumentDraft,
  SaveDocument,
  SaveProducts,
  SaveSettings,
};

/** Shows a Go-side error to the user in Romanian. */
export function showError(prefix: string, err: unknown): void {
  const message = err instanceof Error ? err.message : String(err);
  void showAlert(`${prefix}: ${message}`);
}
```

- [ ] **Step 2: Scrie testele căzute pentru `sidebar.ts`**

`frontend/src/sidebar.test.ts` — doar funcțiile pure, fiindcă `renderSidebar`
cheamă Wails:

```ts
import { describe, expect, it } from 'vitest';
import { currentHash, escapeHtml } from './sidebar';

describe('escapeHtml', () => {
  it('escapeaza si ghilimelele, nu doar unghiularele', () => {
    // Aproape toate apelurile pun rezultatul intr-un atribut HTML, deci o
    // ghilimea neescapata ar rupe atributul.
    expect(escapeHtml('a "b" <c> & \'d\'')).toBe('a &quot;b&quot; &lt;c&gt; &amp; &#39;d&#39;');
  });

  it('escapeaza ampersandul intai, ca sa nu dubleze secventele', () => {
    expect(escapeHtml('&lt;')).toBe('&amp;lt;');
  });
});

describe('currentHash', () => {
  it('cade pe ruta documentului nou cand hash-ul e gol', () => {
    expect(currentHash('')).toBe('#/document/new');
  });

  it('lasa neatins un hash existent', () => {
    expect(currentHash('#/setari')).toBe('#/setari');
  });
});
```

- [ ] **Step 3: Rulează testele ca să cadă**

Run: `cd frontend && npm test`
Expected: FAIL — nu există `./sidebar`.

- [ ] **Step 4: Scrie `frontend/src/sidebar.ts`**

```ts
import { DocumentSummary, ListDocuments, showError } from './api';
import { formatDateRO } from './format';
import { navigate } from './router';

/** The route an unsaved document lives at. */
export const DRAFT_HASH = '#/document/new';

let hashchangeListenerRegistered = false;

/**
 * The hash to match sidebar entries against. A freshly launched app has an
 * empty hash and the router falls back to the new-document route, so the same
 * fallback applies here — otherwise no entry would be highlighted on the one
 * screen the app always opens on.
 */
export function currentHash(hash: string = window.location.hash): string {
  return hash || DRAFT_HASH;
}

/** Renders the sidebar: new-document button, history, settings link. */
export async function renderSidebar(el: HTMLElement): Promise<void> {
  let documente: DocumentSummary[];
  try {
    documente = await ListDocuments();
  } catch (err) {
    showError('Nu s-a putut încărca lista de documente', err);
    documente = [];
  }

  const items = documente
    .map(
      (doc) => `
        <li>
          <a class="doc-link" href="#/document/${doc.id}">
            <span class="doc-nr">NR ${doc.nr}</span>
            <span class="doc-meta">${escapeHtml(formatDateRO(doc.data))}${
              doc.furnizor ? ` — ${escapeHtml(doc.furnizor)}` : ''
            }</span>
          </a>
        </li>`,
    )
    .join('');

  // The draft entry is rendered every pass and shown or hidden by markActive,
  // so moving in and out of the draft route only toggles an attribute instead
  // of rebuilding a list that costs a round trip to SQLite. It sits above the
  // saved documents because the list is newest-first and an unsaved document
  // is newer than all of them.
  el.innerHTML = `
    <div class="new-doc-wrap">
      <button class="btn btn-primary" id="new-doc">+ Notă nouă</button>
    </div>
    <ul class="doc-list">
      <li id="draft-item" hidden>
        <a class="doc-link" id="draft-link" href="${DRAFT_HASH}">
          <span class="doc-nr">Notă nouă</span>
          <span class="doc-meta draft-meta">Nesalvată</span>
        </a>
      </li>
      ${items}
    </ul>
    <a class="settings-link" href="#/setari">Setări</a>
  `;

  el.querySelector<HTMLButtonElement>('#new-doc')!.addEventListener('click', () => {
    navigate(DRAFT_HASH);
  });

  markActive(el);

  // Registered once, against window, which outlives any single render.
  if (!hashchangeListenerRegistered) {
    hashchangeListenerRegistered = true;
    window.addEventListener('hashchange', () => markActive(el));
  }
}

function markActive(el: HTMLElement): void {
  const hash = currentHash();
  el.querySelector('#draft-item')?.toggleAttribute('hidden', hash !== DRAFT_HASH);
  el.querySelectorAll('a').forEach((link) => {
    link.classList.toggle('active', link.getAttribute('href') === hash);
  });
}

/**
 * Escapes text that goes into an innerHTML template. Almost every call site
 * places the result inside an HTML attribute (`value="${escapeHtml(...)}"`),
 * so quotes must be escaped too — a quote in user data would otherwise break
 * out of the attribute, corrupting the value or injecting an attribute of its
 * own. Ampersand goes first so the other replacements' `&...;` sequences are
 * not re-escaped.
 */
export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
```

- [ ] **Step 5: Rulează testele ca să treacă**

Run: `cd frontend && npm test`
Expected: PASS — patru teste noi.

- [ ] **Step 6: Commit**

```bash
git add frontend/src
git commit -m "$(cat <<'EOF'
Puntea catre Go si bara laterala

api.ts reexporta legaturile generate si traduce erorile pentru
utilizator; sidebar.ts deseneaza istoricul, intrarea pentru nota
nesalvata si legatura catre Setari.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 12: Logica pură a formularului

**Files:**
- Create: `frontend/src/nota.ts`
- Test: `frontend/src/nota.test.ts`

**Interfaces:**
- Consumes: tipurile `Document`, `Product`, `Rand` din `api.ts`; `parseDateRO` din `format.ts`.
- Produces:
  - `randGol(cotaImplicita: number): Rand`
  - `randDinProdus(p: Product): Rand`
  - `validareDocument(doc: Document, dataTastata: string): string | undefined` — mesajul primei probleme, sau `undefined`
  - `UM_PERMISE: readonly ['Buc.', 'Kg.']`

Logica stă separat de `views/document.ts` ca să poată fi testată: vitest rulează
în node, fără DOM, iar view-ul e doar cablaj.

- [ ] **Step 1: Scrie testele căzute**

`frontend/src/nota.test.ts`:
```ts
import { describe, expect, it } from 'vitest';
import type { Document, Product } from './api';
import { UM_PERMISE, randDinProdus, randGol, validareDocument } from './nota';

function docValid(): Document {
  return {
    id: 0,
    nr: 1,
    data: '2026-09-09',
    unitate: 'S.C. Largiana Carn S.R.L.',
    documentLivrare: 'Factură',
    documentLivrareNr: '1234',
    documentLivrareData: '2026-09-08',
    furnizor: 'Alfa SRL',
    createdAt: '',
    updatedAt: '',
    randuri: [
      {
        id: 0, productId: undefined, pozitie: 0, denumire: 'Pulpă fără os',
        um: 'Kg.', cantitate: 10, pretFaraTva: 20, cotaTva: 11, pretVanzare: 30,
      },
    ],
  } as unknown as Document;
}

describe('randGol', () => {
  it('porneste de la cota implicita si de la Kg.', () => {
    const r = randGol(11);
    expect(r.cotaTva).toBe(11);
    expect(r.um).toBe('Kg.');
    expect(r.denumire).toBe('');
    expect(r.cantitate).toBe(0);
    expect(r.productId).toBeUndefined();
  });
});

describe('randDinProdus', () => {
  it('copiaza denumirea, unitatea, pretul de vanzare si cota', () => {
    const p = {
      id: 7, denumire: 'Ouă', um: 'Buc.', pretVanzare: 1.5, cotaTva: 11, ordine: 0,
    } as unknown as Product;
    const r = randDinProdus(p);
    expect(r.productId).toBe(7);
    expect(r.denumire).toBe('Ouă');
    expect(r.um).toBe('Buc.');
    expect(r.pretVanzare).toBe(1.5);
    expect(r.cotaTva).toBe(11);
    // Pretul de achizitie nu vine de la produs: variaza de la livrare la livrare.
    expect(r.pretFaraTva).toBe(0);
    expect(r.cantitate).toBe(0);
  });
});

describe('validareDocument', () => {
  it('accepta un document complet', () => {
    expect(validareDocument(docValid(), '09/09/2026')).toBeUndefined();
  });

  it('refuza un numar mai mic de 1', () => {
    const d = docValid();
    d.nr = 0;
    expect(validareDocument(d, '09/09/2026')).toMatch(/număr/i);
  });

  it('refuza o data care nu exista', () => {
    expect(validareDocument(docValid(), '31/02/2026')).toMatch(/dat/i);
  });

  it('refuza un document fara randuri', () => {
    const d = docValid();
    d.randuri = [];
    expect(validareDocument(d, '09/09/2026')).toMatch(/produs/i);
  });

  it('refuza un rand fara denumire si spune care', () => {
    const d = docValid();
    d.randuri[0].denumire = '   ';
    const problema = validareDocument(d, '09/09/2026');
    expect(problema).toMatch(/rândul 1/i);
  });

  it('refuza o unitate de masura pe care formularul n-o cunoaste', () => {
    const d = docValid();
    d.randuri[0].um = 'litri';
    expect(validareDocument(d, '09/09/2026')).toMatch(/U\/M/i);
  });
});

describe('UM_PERMISE', () => {
  it('are exact cele doua unitati de pe formular', () => {
    expect([...UM_PERMISE]).toEqual(['Buc.', 'Kg.']);
  });
});
```

- [ ] **Step 2: Rulează testele ca să cadă**

Run: `cd frontend && npm test`
Expected: FAIL — nu există `./nota`.

- [ ] **Step 3: Scrie `frontend/src/nota.ts`**

```ts
import type { Document, Product, Rand } from './api';
import { parseDateRO } from './format';

/**
 * The units the form accepts. The paper form's U/M column is free-hand, but
 * this app writes it, and two values are all the business uses — restricting
 * it keeps a document from carrying "kg", "Kg", "kilograme" and "Kg." as four
 * different things.
 */
export const UM_PERMISE = ['Buc.', 'Kg.'] as const;

/** A blank row, ready to be typed into. */
export function randGol(cotaImplicita: number): Rand {
  return {
    id: 0,
    productId: undefined,
    pozitie: 0,
    denumire: '',
    um: 'Kg.',
    cantitate: 0,
    pretFaraTva: 0,
    cotaTva: cotaImplicita,
    pretVanzare: 0,
  } as unknown as Rand;
}

/**
 * A row starting from a catalogue product: name, unit, selling price and rate
 * are copied in, and from this point the row owns them. Editing the product
 * later leaves this row alone.
 *
 * The purchase price is not copied, because the product does not carry one:
 * it is what this particular delivery cost, and it is typed in at reception.
 */
export function randDinProdus(p: Product): Rand {
  return {
    id: 0,
    productId: p.id,
    pozitie: 0,
    denumire: p.denumire,
    um: p.um,
    cantitate: 0,
    pretFaraTva: 0,
    cotaTva: p.cotaTva,
    pretVanzare: p.pretVanzare,
  } as unknown as Rand;
}

/**
 * The first thing wrong with the document, or undefined if nothing is.
 *
 * The date is checked as it was typed rather than as it is stored: a
 * half-typed "09/09/20" never becomes an ISO date, so checking doc.data would
 * silently pass the last valid value while the field on screen says otherwise.
 */
export function validareDocument(doc: Document, dataTastata: string): string | undefined {
  if (!Number.isInteger(doc.nr) || doc.nr < 1) {
    return 'Numărul notei trebuie să fie un număr întreg, cel puțin 1.';
  }
  if (parseDateRO(dataTastata) === undefined) {
    return 'Data notei nu este o dată validă. Se scrie ZZ/LL/AAAA.';
  }
  if (doc.documentLivrareData !== '' && parseDateRO(formatCaLaTastare(doc.documentLivrareData)) === undefined) {
    return 'Data documentului de livrare nu este o dată validă. Se scrie ZZ/LL/AAAA.';
  }
  if (doc.randuri.length === 0) {
    return 'Nota trebuie să aibă cel puțin un produs.';
  }
  for (const [i, r] of doc.randuri.entries()) {
    if (r.denumire.trim() === '') {
      return `Rândul ${i + 1} nu are denumire.`;
    }
    if (!(UM_PERMISE as readonly string[]).includes(r.um)) {
      return `Rândul ${i + 1}: U/M trebuie să fie „Buc." sau „Kg.".`;
    }
  }
  return undefined;
}

/** Turns a stored ISO date back into the ZZ/LL/AAAA that parseDateRO reads. */
function formatCaLaTastare(iso: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  return match === null ? iso : `${match[3]}/${match[2]}/${match[1]}`;
}
```

- [ ] **Step 4: Rulează testele ca să treacă**

Run: `cd frontend && npm test`
Expected: PASS — zece teste noi.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/nota.ts frontend/src/nota.test.ts
git commit -m "$(cat <<'EOF'
Logica pura a formularului

Randul gol, randul pornit dintr-un produs si validarea la salvare, tinute
separat de view ca sa poata fi testate fara DOM. Randul copiaza produsul,
nu il refera: pretul de achizitie ramane al livrarii.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 13: Ecranul formularului (`views/document.ts`)

**Files:**
- Create: `frontend/src/views/document.ts`

**Interfaces:**
- Consumes: `api.ts`, `calc.ts`, `format.ts`, `fuzzy.ts`, `nota.ts`, `dialog.ts`, `toast.ts`, `router.ts`, `sidebar.ts` (pentru `escapeHtml`).
- Produces: `renderDocumentView(outlet: HTMLElement, id: string | undefined, refreshSidebar: () => Promise<void>): Promise<void>` — chemată de `main.ts` din Task 9.

Ecranul e cablaj: starea e un singur obiect `doc` ținut în closure, orice
tastare îl actualizează și recalculează coloanele derivate, iar `Salvează` îl
trimite la Go. Structura urmează `views/document.ts` din aplicația soră.

- [ ] **Step 1: Scrie scheletul view-ului**

`frontend/src/views/document.ts`:

```ts
import {
  AddProduct,
  DeleteDocument,
  ExportPDF,
  GetDocument,
  GetSettings,
  ListFurnizori,
  ListProducts,
  NewDocumentDraft,
  SaveDocument,
  showError,
} from '../api';
import type { Document, Product, Rand } from '../api';
import { totaluri, valoriRand } from '../calc';
import { showAlert, showConfirm } from '../dialog';
import { UM_PERMISE, randGol, validareDocument } from '../nota';
import { formatDateRO, formatLei, formatNumber, formatProcent, parseDateRO, parseNumber } from '../format';
import { navigate } from '../router';
import { escapeHtml } from '../sidebar';
import { showToast } from '../toast';

export async function renderDocumentView(
  outlet: HTMLElement,
  id: string | undefined,
  refreshSidebar: () => Promise<void>,
): Promise<void> {
  let doc: Document;
  let produse: Product[];
  let furnizori: string[];
  let cotaImplicita: number;

  try {
    const setari = await GetSettings();
    cotaImplicita = setari.cotaTva;
    [produse, furnizori] = await Promise.all([ListProducts(), ListFurnizori()]);
    doc = id === undefined ? await NewDocumentDraft() : await GetDocument(Number(id));
  } catch (err) {
    showError('Nu s-a putut încărca nota', err);
    outlet.innerHTML = '<p class="empty">Nota nu a putut fi încărcată.</p>';
    return;
  }

  render();

  function render(): void {
    outlet.innerHTML = `
      <h1>${doc.id === 0 ? 'Notă de recepție nouă' : `Notă de recepție NR ${doc.nr}`}</h1>

      <div class="header-grid">
        <div class="field">
          <label for="f-unitate">Unitatea</label>
          <input id="f-unitate" value="${escapeHtml(doc.unitate)}" readonly />
        </div>
        <div class="field">
          <label for="f-nr">nr.</label>
          <input id="f-nr" class="num" type="number" min="1" step="1" value="${doc.nr}" />
        </div>
        <div class="field">
          <label for="f-data">din data</label>
          <input id="f-data" type="text" inputmode="numeric" maxlength="10"
                 placeholder="ZZ/LL/AAAA" value="${escapeHtml(formatDateRO(doc.data))}" />
        </div>
      </div>

      <h2>Document livrare</h2>
      <div class="header-grid">
        <div class="field">
          <label for="f-livrare">Document livrare</label>
          <input id="f-livrare" placeholder="Factură, Aviz…" value="${escapeHtml(doc.documentLivrare)}" />
        </div>
        <div class="field">
          <label for="f-livrare-nr">Nr.</label>
          <input id="f-livrare-nr" value="${escapeHtml(doc.documentLivrareNr)}" />
        </div>
        <div class="field">
          <label for="f-livrare-data">Data</label>
          <input id="f-livrare-data" type="text" inputmode="numeric" maxlength="10"
                 placeholder="ZZ/LL/AAAA" value="${escapeHtml(formatDateRO(doc.documentLivrareData))}" />
        </div>
        <div class="field">
          <label for="f-furnizor">Furnizorul</label>
          <input id="f-furnizor" list="lista-furnizori" value="${escapeHtml(doc.furnizor)}" />
          <datalist id="lista-furnizori">
            ${furnizori.map((f) => `<option value="${escapeHtml(f)}"></option>`).join('')}
          </datalist>
        </div>
      </div>

      <h2>Produse</h2>
      ${tabelProduse()}
      <div class="table-actions">
        <button class="btn" id="add-rand">+ Adaugă rând</button>
      </div>

      ${panouTotaluri()}

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
        ${doc.id === 0 ? '' : '<button class="btn" id="print">Printează (PDF)</button>'}
        ${doc.id === 0 ? '' : '<button class="btn btn-danger" id="delete">Șterge</button>'}
      </div>
    `;

    wireEvents();
    recompute();
  }
```

- [ ] **Step 2: Adaugă tabelul de produse**

Continuă în același fișier, tot înăuntrul lui `renderDocumentView`:

```ts
  /**
   * The product table. Cota T.V.A., Adaos and Adaos % are working columns:
   * they exist here and not on the printed form. The derived cells carry no
   * inputs — they are recomputed from the row on every keystroke.
   */
  function tabelProduse(): string {
    const randuri = doc.randuri
      .map(
        (r, i) => `
        <tr data-rand="${i}">
          <td class="nr-crt">${i + 1}</td>
          <td class="combo">
            <input class="denumire" data-camp="denumire" value="${escapeHtml(r.denumire)}"
                   autocomplete="off" />
          </td>
          <td>
            <select data-camp="um">
              ${UM_PERMISE.map(
                (um) => `<option value="${um}" ${r.um === um ? 'selected' : ''}>${um}</option>`,
              ).join('')}
            </select>
          </td>
          <td><input class="num" data-camp="cantitate" value="${formatNumber(r.cantitate)}" /></td>
          <td><input class="num" data-camp="pretFaraTva" value="${formatNumber(r.pretFaraTva)}" /></td>
          <td><input class="num cota" data-camp="cotaTva" value="${formatNumber(r.cotaTva)}" /></td>
          <td class="derivat" data-derivat="valoareFaraTva"></td>
          <td class="derivat" data-derivat="valoareCuTva"></td>
          <td><input class="num" data-camp="pretVanzare" value="${formatNumber(r.pretVanzare)}" /></td>
          <td class="derivat" data-derivat="valoareVanzare"></td>
          <td class="derivat" data-derivat="adaos"></td>
          <td class="derivat" data-derivat="adaosProcent"></td>
          <td><button class="btn-icon sterge-rand" title="Șterge rândul">×</button></td>
        </tr>`,
      )
      .join('');

    return `
      <table class="tabel-produse">
        <thead>
          <tr>
            <th>Nr. crt.</th>
            <th>Denumirea</th>
            <th>U/M</th>
            <th class="num">Cantitatea</th>
            <th class="num">Preț fără T.V.A.</th>
            <th class="num cota">Cota T.V.A. %</th>
            <th class="num">Valoare fără T.V.A.</th>
            <th class="num">Valoare cu T.V.A.</th>
            <th class="num">Preț de vânzare</th>
            <th class="num">Valoare la preț de vânzare</th>
            <th class="num">Adaos</th>
            <th class="num">Adaos %</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${randuri || '<tr><td colspan="13" class="empty">Nicio linie. Apasă „+ Adaugă rând".</td></tr>'}
        </tbody>
      </table>`;
  }

  /**
   * The totals block. It sits under the table as labelled cells rather than as
   * a row inside it: the columns it sums are far apart, and a TOTAL row leaves
   * the reader counting columns to find out which figure is which.
   *
   * There is no quantity total — U/M varies from row to row.
   */
  function panouTotaluri(): string {
    return `
      <div class="totaluri">
        <div class="celula">
          <span class="eticheta">Total valoare fără T.V.A.</span>
          <span class="valoare" id="t-fara-tva">—</span>
        </div>
        <div class="celula">
          <span class="eticheta">Total valoare cu T.V.A.</span>
          <span class="valoare" id="t-cu-tva">—</span>
        </div>
        <div class="celula">
          <span class="eticheta">Total valoare la preț de vânzare</span>
          <span class="valoare" id="t-vanzare">—</span>
        </div>
        <div class="celula">
          <span class="eticheta">Total adaos</span>
          <span class="valoare" id="t-adaos">—</span>
        </div>
        <div class="celula">
          <span class="eticheta">Adaos %</span>
          <span class="valoare" id="t-adaos-procent">—</span>
        </div>
      </div>`;
  }
```

- [ ] **Step 3: Adaugă recalculul**

```ts
  /**
   * Recomputes every derived cell and the totals block from `doc`. It runs on
   * every keystroke, so it only writes text into existing cells — it never
   * rebuilds the table, which would take the focus out of the field being
   * typed into.
   */
  function recompute(): void {
    doc.randuri.forEach((r, i) => {
      const rand = outlet.querySelector(`tr[data-rand="${i}"]`);
      if (rand === null) return;
      const v = valoriRand(r);
      scrieDerivat(rand, 'valoareFaraTva', formatLei(v.valoareFaraTva));
      scrieDerivat(rand, 'valoareCuTva', formatLei(v.valoareCuTva));
      scrieDerivat(rand, 'valoareVanzare', formatLei(v.valoareVanzare));
      scrieDerivat(rand, 'adaos', formatLei(v.adaos), v.adaos < 0);
      scrieDerivat(rand, 'adaosProcent', formatProcent(v.adaosProcent),
        v.adaosProcent !== undefined && v.adaosProcent < 0);
    });

    const t = totaluri(doc.randuri);
    setText('t-fara-tva', formatLei(t.valoareFaraTva));
    setText('t-cu-tva', formatLei(t.valoareCuTva));
    setText('t-vanzare', formatLei(t.valoareVanzare));
    setText('t-adaos', formatLei(t.adaos), t.adaos < 0);
    setText('t-adaos-procent', formatProcent(t.adaosProcent),
      t.adaosProcent !== undefined && t.adaosProcent < 0);
  }

  function scrieDerivat(rand: Element, camp: string, text: string, negativ = false): void {
    const celula = rand.querySelector(`[data-derivat="${camp}"]`);
    if (celula === null) return;
    celula.textContent = text;
    celula.classList.toggle('negativ', negativ);
  }

  function setText(id: string, text: string, negativ = false): void {
    const el = outlet.querySelector(`#${id}`);
    if (el === null) return;
    el.textContent = text;
    el.classList.toggle('negativ', negativ);
  }
```

- [ ] **Step 4: Adaugă cablajul evenimentelor**

```ts
  function wireEvents(): void {
    outlet.querySelector<HTMLInputElement>('#f-nr')!.addEventListener('input', (e) => {
      doc.nr = Number((e.target as HTMLInputElement).value);
    });
    outlet.querySelector<HTMLInputElement>('#f-data')!.addEventListener('input', (e) => {
      const iso = parseDateRO((e.target as HTMLInputElement).value);
      // A half-typed date is left alone rather than written as garbage; the
      // save-time validation is what refuses it, with a message.
      if (iso !== undefined) doc.data = iso;
    });
    legaText('#f-livrare', (v) => (doc.documentLivrare = v));
    legaText('#f-livrare-nr', (v) => (doc.documentLivrareNr = v));
    legaText('#f-furnizor', (v) => (doc.furnizor = v));
    outlet.querySelector<HTMLInputElement>('#f-livrare-data')!.addEventListener('input', (e) => {
      const text = (e.target as HTMLInputElement).value;
      const iso = parseDateRO(text);
      if (text.trim() === '') doc.documentLivrareData = '';
      else if (iso !== undefined) doc.documentLivrareData = iso;
    });

    outlet.querySelectorAll<HTMLElement>('tr[data-rand]').forEach((rand) => {
      const index = Number(rand.dataset.rand);
      rand.querySelectorAll<HTMLInputElement | HTMLSelectElement>('[data-camp]').forEach((camp) => {
        camp.addEventListener('input', () => {
          aplicaCamp(index, camp.dataset.camp!, camp.value);
          recompute();
        });
      });
      rand.querySelector('.sterge-rand')!.addEventListener('click', () => {
        doc.randuri.splice(index, 1);
        render();
      });
    });

    outlet.querySelector<HTMLButtonElement>('#add-rand')!.addEventListener('click', () => {
      doc.randuri.push(randGol(cotaImplicita));
      render();
      // The new row's name field is where typing continues.
      const inputuri = outlet.querySelectorAll<HTMLInputElement>('input.denumire');
      inputuri[inputuri.length - 1]?.focus();
    });

    outlet.querySelector<HTMLButtonElement>('#save')!.addEventListener('click', salveaza);
    outlet.querySelector<HTMLButtonElement>('#print')?.addEventListener('click', printeaza);
    outlet.querySelector<HTMLButtonElement>('#delete')?.addEventListener('click', sterge);
  }

  function legaText(selector: string, seteaza: (v: string) => void): void {
    outlet.querySelector<HTMLInputElement>(selector)!.addEventListener('input', (e) => {
      seteaza((e.target as HTMLInputElement).value);
    });
  }

  /**
   * Writes one typed field back onto its row.
   *
   * Editing the name by hand detaches the row from its product: what is on the
   * row no longer says what the catalogue says, and keeping the link would
   * make the row look like a product it is not. Picking from the dropdown (see
   * the combobox task) re-attaches it.
   */
  function aplicaCamp(index: number, camp: string, valoare: string): void {
    const r = doc.randuri[index] as unknown as Record<string, unknown>;
    switch (camp) {
      case 'denumire':
        r.denumire = valoare;
        r.productId = undefined;
        break;
      case 'um':
        r.um = valoare;
        break;
      default:
        r[camp] = parseNumber(valoare);
    }
  }
```

- [ ] **Step 5: Adaugă salvarea, printarea și ștergerea**

```ts
  async function salveaza(): Promise<void> {
    const dataTastata = outlet.querySelector<HTMLInputElement>('#f-data')!.value;
    const problema = validareDocument(doc, dataTastata);
    if (problema !== undefined) {
      await showAlert(problema);
      return;
    }
    try {
      doc = await SaveDocument(doc);
    } catch (err) {
      showError('Nu s-a putut salva nota', err);
      return;
    }
    showToast('Nota a fost salvată.');
    await refreshSidebar();
    navigate(`#/document/${doc.id}`);
  }

  async function printeaza(): Promise<void> {
    try {
      const path = await ExportPDF(doc.id);
      // An empty path is the user cancelling the save dialog, which needs no
      // confirmation of its own.
      if (path !== '') showToast('PDF salvat.');
    } catch (err) {
      showError('Nu s-a putut genera PDF-ul', err);
    }
  }

  async function sterge(): Promise<void> {
    if (!(await showConfirm(`Ștergi nota de recepție NR ${doc.nr}?`))) return;
    try {
      await DeleteDocument(doc.id);
    } catch (err) {
      showError('Nu s-a putut șterge nota', err);
      return;
    }
    await refreshSidebar();
    navigate('#/document/new');
  }
}
```

Verifică la final că acoladele se închid: `renderDocumentView` conține toate
funcțiile de mai sus și se termină cu `}`.

- [ ] **Step 6: Compilează**

Run: `cd frontend && npx tsc --noEmit`
Expected: fără erori, cu excepția lui `views/setari.ts`, care încă nu există.
Comentează temporar importul lui din `main.ts` ca să obții o compilare curată,
și decomentează-l în Task 15.

`AddProduct` și `ListProducts` sunt importate în Step 1 dar folosite abia în
Task 14; `cautaProduse` și `randDinProdus` se adaugă la importuri tot acolo.
TypeScript nu se plânge de un import nefolosit cu setările din `tsconfig.json`.

- [ ] **Step 7: Pornește aplicația și uită-te la ea**

Run: `wails dev`
Expected: fereastra se deschide pe „Notă de recepție nouă", cu antetul, tabelul
de livrare, tabelul gol de produse, panoul de totaluri cu cinci celule pe „—"
și butonul Salvează. Adaugă un rând, scrie o denumire, o cantitate, un preț
fără TVA și un preț de vânzare: cele patru coloane derivate și panoul de
totaluri se completează la fiecare tastă. Salvează, apoi verifică în bara
laterală că nota apare cu NR-ul și furnizorul ei.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/views/document.ts
git commit -m "$(cat <<'EOF'
Ecranul formularului

Antetul, tabelul de livrare, tabelul de produse cu recalcul la fiecare
tasta si panoul de totaluri cu etichete deasupra valorilor. Cota T.V.A.,
adaosul in lei si adaosul procentual sunt coloane de lucru: exista aici,
nu si pe formularul tiparit.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 14: Caseta de căutare din coloana Denumirea

**Files:**
- Modify: `frontend/src/views/document.ts` — funcția `wireEvents`, plus funcțiile noi de mai jos

**Interfaces:**
- Consumes: `cautaProduse` din `fuzzy.ts`, `randDinProdus` din `nota.ts`, `AddProduct` și `ListProducts` din `api.ts`.
- Produces: nimic nou în afara fișierului; `renderDocumentView` își păstrează semnătura.

Ce trebuie să facă: la tastare în câmpul de denumire apare o listă cu produsele
potrivite; alegerea unuia completează U/M, prețul de vânzare și cota și
reatașează rândul la produs; dacă denumirea nu există în catalog, sub listă
apare un buton care o adaugă în Setări fără a părăsi formularul.

- [ ] **Step 1: Completează importurile**

La blocul de importuri din `frontend/src/views/document.ts`, adaugă:

```ts
import { cautaProduse } from '../fuzzy';
import { randDinProdus } from '../nota';   // langa UM_PERMISE, randGol, validareDocument
```

`AddProduct` și `ListProducts` sunt deja importate din `../api`, iar tipul
`Rand` din `../api` e folosit de `salveazaProdusNou`.

- [ ] **Step 2: Adaugă cablajul casetei în `wireEvents`**

Înăuntrul buclei `outlet.querySelectorAll('tr[data-rand]')`, după cablajul
câmpurilor, adaugă:

```ts
      const denumire = rand.querySelector<HTMLInputElement>('input.denumire')!;
      denumire.addEventListener('input', () => deschideCombo(index, denumire));
      denumire.addEventListener('focus', () => deschideCombo(index, denumire));
      denumire.addEventListener('keydown', (e) => navigheazaCombo(e, index, denumire));
      // Blur closes on the next tick so a click on a suggestion lands before
      // the list is removed; a click removes it itself, so nothing flickers.
      denumire.addEventListener('blur', () => window.setTimeout(inchideCombo, 150));
```

- [ ] **Step 3: Adaugă funcțiile casetei**

Tot înăuntrul lui `renderDocumentView`:

```ts
  // The index of the row whose suggestion list is open, and which entry is
  // highlighted in it. They live here rather than on the DOM because the list
  // is rebuilt on every keystroke and would lose the highlight otherwise.
  let comboRand: number | undefined;
  let comboEvidentiat = 0;

  /** Draws (or redraws) the suggestion list under one row's name field. */
  function deschideCombo(index: number, input: HTMLInputElement): void {
    inchideCombo();
    comboRand = index;

    const potriviri = cautaProduse(produse, input.value);
    const exact = produse.some(
      (p) => p.denumire.trim().toLowerCase() === input.value.trim().toLowerCase(),
    );
    const poateFiSalvat = input.value.trim() !== '' && !exact;

    if (potriviri.length === 0 && !poateFiSalvat) return;
    if (comboEvidentiat >= potriviri.length) comboEvidentiat = 0;

    const lista = document.createElement('ul');
    lista.className = 'combo-lista';
    lista.innerHTML =
      potriviri
        .map(
          (p, i) => `
        <li data-produs="${p.id}" class="${i === comboEvidentiat ? 'evidentiat' : ''}">
          ${escapeHtml(p.denumire)}<span class="produs-um">${escapeHtml(p.um)}</span>
        </li>`,
        )
        .join('') +
      (poateFiSalvat
        ? `<li class="combo-nou" data-nou="1">„${escapeHtml(input.value.trim())}" nu este în produse —
             <strong>salvează-l</strong></li>`
        : '');

    // mousedown, not click: the field's blur fires first on a click, and the
    // handler would then be running against a list already being torn down.
    lista.addEventListener('mousedown', (e) => {
      e.preventDefault();
      const li = (e.target as HTMLElement).closest('li');
      if (li === null) return;
      if (li.dataset.nou === '1') {
        void salveazaProdusNou(index, input);
        return;
      }
      const produs = produse.find((p) => p.id === Number(li.dataset.produs));
      if (produs !== undefined) alegeProdus(index, produs);
    });

    input.closest('td')!.appendChild(lista);
  }

  function inchideCombo(): void {
    outlet.querySelectorAll('.combo-lista').forEach((el) => el.remove());
    comboRand = undefined;
  }

  /** Arrow keys move the highlight, Enter takes it, Escape closes the list. */
  function navigheazaCombo(e: KeyboardEvent, index: number, input: HTMLInputElement): void {
    const lista = outlet.querySelector<HTMLUListElement>('.combo-lista');
    if (lista === null || comboRand !== index) return;
    const optiuni = lista.querySelectorAll<HTMLLIElement>('li[data-produs]');

    if (e.key === 'Escape') {
      inchideCombo();
      return;
    }
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (optiuni.length === 0) return;
      comboEvidentiat =
        (comboEvidentiat + (e.key === 'ArrowDown' ? 1 : optiuni.length - 1)) % optiuni.length;
      optiuni.forEach((li, i) => li.classList.toggle('evidentiat', i === comboEvidentiat));
      return;
    }
    if (e.key === 'Enter' && optiuni.length > 0) {
      e.preventDefault();
      const produs = produse.find(
        (p) => p.id === Number(optiuni[comboEvidentiat].dataset.produs),
      );
      if (produs !== undefined) alegeProdus(index, produs);
    }
  }

  /**
   * Replaces a row with one started from the chosen product, keeping whatever
   * has already been typed into it. The quantity and the purchase price belong
   * to this delivery, not to the catalogue, so picking a product must not wipe
   * figures the user has already entered for them.
   */
  function alegeProdus(index: number, produs: Product): void {
    const vechi = doc.randuri[index];
    const nou = randDinProdus(produs);
    nou.cantitate = vechi.cantitate;
    nou.pretFaraTva = vechi.pretFaraTva;
    doc.randuri[index] = nou;
    inchideCombo();
    render();
    // Typing continues in the field that was just filled in.
    outlet.querySelectorAll<HTMLInputElement>('input.denumire')[index]?.focus();
  }

  /**
   * Files the typed name as a new product and attaches the row to it.
   *
   * The unit, the selling price and the rate come from the row as it stands:
   * the user has just typed them, and asking for them again in Setări would be
   * asking twice for the same answer.
   */
  async function salveazaProdusNou(index: number, input: HTMLInputElement): Promise<void> {
    const r = doc.randuri[index];
    try {
      const produs = await AddProduct({
        id: 0,
        denumire: input.value.trim(),
        um: r.um,
        pretVanzare: r.pretVanzare,
        cotaTva: r.cotaTva,
        ordine: 0,
      } as unknown as Product);
      produse = await ListProducts();
      doc.randuri[index] = { ...r, productId: produs.id, denumire: produs.denumire } as Rand;
      inchideCombo();
      render();
      showToast(`„${produs.denumire}" a fost adăugat în produse.`);
    } catch (err) {
      showError('Nu s-a putut salva produsul', err);
    }
  }
```

Schimbă declarația lui `produse` de la `let produse: Product[];` — deja e `let`,
deci `salveazaProdusNou` îl poate reatribui.

- [ ] **Step 4: Compilează**

Run: `cd frontend && npx tsc --noEmit`
Expected: fără erori noi.

- [ ] **Step 5: Verifică în aplicație**

Run: `wails dev`

Verifică, în ordine:
1. Setări → adaugă două produse („Pulpă fără os", Kg., 30 lei, 11%; „Ouă",
   Buc., 1,5 lei, 11%). *(Ecranul de Setări vine în Task 15; până atunci
   adaugă-le din formular, cu butonul „salvează-l".)*
2. În formular, „+ Adaugă rând", scrie `pulpa` — apare „Pulpă fără os", deși
   ai scris fără diacritice.
3. Scrie `pfo` — tot îl găsește, după inițiale.
4. Alege-l cu săgeata în jos și Enter: U/M devine Kg., prețul de vânzare 30,
   cota 11.
5. Scrie o cantitate și un preț fără TVA, apoi alege alt produs din listă:
   cantitatea și prețul de achiziție rămân.
6. Scrie o denumire care nu există: apare rândul „nu este în produse —
   salvează-l". Apasă-l; produsul se adaugă și rândul se leagă de el.
7. Editează denumirea de mână după ce ai ales un produs: legătura se rupe
   (rândul rămâne cu ce scrie în el, dar nu mai e produsul acela).

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/document.ts
git commit -m "$(cat <<'EOF'
Caseta de cautare din coloana Denumirea

Cautare fuzzy peste catalog, fara diacritice si dupa initiale, cu
navigare din sageti. Alegerea unui produs pastreaza cantitatea si pretul
de achizitie deja tastate; o denumire care nu exista se poate salva in
produse fara a parasi formularul.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 15: Ecranul Setări

**Files:**
- Create: `frontend/src/views/setari.ts`
- Modify: `frontend/src/main.ts` — decomentează importul lui `renderSetariView` dacă a fost comentat în Task 13

**Interfaces:**
- Consumes: `api.ts`, `nota.ts` (`UM_PERMISE`), `dialog.ts`, `toast.ts`, `format.ts`, `sidebar.ts` (`escapeHtml`).
- Produces: `renderSetariView(outlet: HTMLElement, refreshSidebar: () => Promise<void>): Promise<void>`

- [ ] **Step 1: Scrie `frontend/src/views/setari.ts`**

```ts
import {
  DeleteFurnizor,
  GetSettings,
  ListFurnizori,
  ListProducts,
  SaveProducts,
  SaveSettings,
  showError,
} from '../api';
import type { Product, Settings } from '../api';
import { showAlert, showConfirm } from '../dialog';
import { UM_PERMISE } from '../nota';
import { formatNumber, parseNumber } from '../format';
import { escapeHtml } from '../sidebar';
import { showToast } from '../toast';

export async function renderSetariView(
  outlet: HTMLElement,
  refreshSidebar: () => Promise<void>,
): Promise<void> {
  let setari: Settings;
  let produse: Product[];
  let furnizori: string[];

  try {
    [setari, produse, furnizori] = await Promise.all([
      GetSettings(),
      ListProducts(),
      ListFurnizori(),
    ]);
  } catch (err) {
    showError('Nu s-au putut încărca setările', err);
    outlet.innerHTML = '<p class="empty">Setările nu au putut fi încărcate.</p>';
    return;
  }

  render();

  function render(): void {
    outlet.innerHTML = `
      <h1>Setări</h1>

      <h2>Unitatea</h2>
      <div class="header-grid">
        <div class="field">
          <label for="s-unitate">Numele unității</label>
          <input id="s-unitate" value="${escapeHtml(setari.unitateNume)}" />
        </div>
        <div class="field">
          <label for="s-nr">Numărul următoarei note</label>
          <input id="s-nr" class="num" type="number" min="1" step="1" value="${setari.nextNr}" />
        </div>
        <div class="field">
          <label for="s-cota">Cota T.V.A. implicită (%)</label>
          <input id="s-cota" class="num" value="${formatNumber(setari.cotaTva)}" />
        </div>
      </div>

      <h2>Produse</h2>
      <p class="empty">
        Modificările de aici nu schimbă notele deja salvate: fiecare notă
        păstrează denumirea, U/M, prețul de vânzare și cota pe care produsul
        le avea când a fost salvată.
      </p>
      ${tabelProduse()}
      <div class="table-actions">
        <button class="btn" id="add-produs">+ Adaugă produs</button>
      </div>

      <h2>Furnizori memorați</h2>
      ${listaFurnizori()}

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
      </div>
    `;
    wireEvents();
  }

  function tabelProduse(): string {
    const randuri = produse
      .map(
        (p, i) => `
        <tr data-produs="${i}">
          <td>${i + 1}</td>
          <td><input data-camp="denumire" value="${escapeHtml(p.denumire)}" /></td>
          <td>
            <select data-camp="um">
              ${UM_PERMISE.map(
                (um) => `<option value="${um}" ${p.um === um ? 'selected' : ''}>${um}</option>`,
              ).join('')}
            </select>
          </td>
          <td><input class="num" data-camp="pretVanzare" value="${formatNumber(p.pretVanzare)}" /></td>
          <td><input class="num cota" data-camp="cotaTva" value="${formatNumber(p.cotaTva)}" /></td>
          <td>
            <button class="btn-icon muta-sus" title="Mută mai sus" ${i === 0 ? 'disabled' : ''}>↑</button>
            <button class="btn-icon muta-jos" title="Mută mai jos"
                    ${i === produse.length - 1 ? 'disabled' : ''}>↓</button>
            <button class="btn-icon sterge-produs" title="Șterge produsul">×</button>
          </td>
        </tr>`,
      )
      .join('');

    return `
      <table>
        <thead>
          <tr>
            <th>Nr.</th>
            <th>Denumirea</th>
            <th>U/M</th>
            <th class="num">Preț de vânzare</th>
            <th class="num cota">Cota T.V.A. %</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${randuri || '<tr><td colspan="6" class="empty">Niciun produs.</td></tr>'}
        </tbody>
      </table>`;
  }

  function listaFurnizori(): string {
    if (furnizori.length === 0) {
      return '<p class="empty">Niciun furnizor memorat încă.</p>';
    }
    return `
      <ul class="doc-list">
        ${furnizori
          .map(
            (f) => `
          <li class="furnizor-rand">
            <span>${escapeHtml(f)}</span>
            <button class="btn-icon sterge-furnizor" data-furnizor="${escapeHtml(f)}"
                    title="Șterge sugestia">×</button>
          </li>`,
          )
          .join('')}
      </ul>`;
  }

  function wireEvents(): void {
    outlet.querySelector<HTMLInputElement>('#s-unitate')!.addEventListener('input', (e) => {
      setari.unitateNume = (e.target as HTMLInputElement).value;
    });
    outlet.querySelector<HTMLInputElement>('#s-nr')!.addEventListener('input', (e) => {
      setari.nextNr = Number((e.target as HTMLInputElement).value);
    });
    outlet.querySelector<HTMLInputElement>('#s-cota')!.addEventListener('input', (e) => {
      setari.cotaTva = parseNumber((e.target as HTMLInputElement).value);
    });

    outlet.querySelectorAll<HTMLElement>('tr[data-produs]').forEach((rand) => {
      const i = Number(rand.dataset.produs);
      rand.querySelectorAll<HTMLInputElement | HTMLSelectElement>('[data-camp]').forEach((camp) => {
        camp.addEventListener('input', () => {
          const p = produse[i] as unknown as Record<string, unknown>;
          const nume = camp.dataset.camp!;
          p[nume] = nume === 'denumire' || nume === 'um' ? camp.value : parseNumber(camp.value);
        });
      });
      rand.querySelector('.muta-sus')!.addEventListener('click', () => muta(i, -1));
      rand.querySelector('.muta-jos')!.addEventListener('click', () => muta(i, 1));
      rand.querySelector('.sterge-produs')!.addEventListener('click', () => {
        produse.splice(i, 1);
        render();
      });
    });

    outlet.querySelector<HTMLButtonElement>('#add-produs')!.addEventListener('click', () => {
      produse.push({
        id: 0, denumire: '', um: 'Kg.', pretVanzare: 0, cotaTva: setari.cotaTva, ordine: produse.length,
      } as unknown as Product);
      render();
      const inputuri = outlet.querySelectorAll<HTMLInputElement>('[data-camp="denumire"]');
      inputuri[inputuri.length - 1]?.focus();
    });

    outlet.querySelectorAll<HTMLButtonElement>('.sterge-furnizor').forEach((btn) => {
      btn.addEventListener('click', () => void stergeFurnizor(btn.dataset.furnizor!));
    });

    outlet.querySelector<HTMLButtonElement>('#save')!.addEventListener('click', salveaza);
  }

  function muta(index: number, directie: number): void {
    const tinta = index + directie;
    if (tinta < 0 || tinta >= produse.length) return;
    [produse[index], produse[tinta]] = [produse[tinta], produse[index]];
    render();
  }

  /**
   * Deleting a supplier only forgets a suggestion. Documents keep the name as
   * text, so nothing that has been filed changes — which is why this needs no
   * warning beyond the confirmation.
   */
  async function stergeFurnizor(nume: string): Promise<void> {
    if (!(await showConfirm(`Ștergi sugestia „${nume}"?`))) return;
    try {
      await DeleteFurnizor(nume);
      furnizori = await ListFurnizori();
    } catch (err) {
      showError('Nu s-a putut șterge furnizorul', err);
      return;
    }
    render();
  }

  async function salveaza(): Promise<void> {
    if (setari.nextNr < 1) {
      await showAlert('Numărul următoarei note trebuie să fie cel puțin 1.');
      return;
    }
    const gol = produse.findIndex((p) => p.denumire.trim() === '');
    if (gol !== -1) {
      await showAlert(`Produsul ${gol + 1} nu are denumire.`);
      return;
    }
    const duplicat = primulDuplicat(produse);
    if (duplicat !== undefined) {
      await showAlert(`Denumirea „${duplicat}" apare de două ori în listă.`);
      return;
    }

    try {
      await SaveSettings(setari);
      await SaveProducts(produse);
      produse = await ListProducts();
    } catch (err) {
      showError('Nu s-au putut salva setările', err);
      return;
    }
    showToast('Setările au fost salvate.');
    await refreshSidebar();
    render();
  }
}

/**
 * The first name that appears twice, ignoring case — the store refuses these,
 * and saying which one it is here is more useful than relaying a constraint
 * violation from SQLite.
 */
function primulDuplicat(produse: Product[]): string | undefined {
  const vazute = new Set<string>();
  for (const p of produse) {
    const cheie = p.denumire.trim().toLowerCase();
    if (vazute.has(cheie)) return p.denumire.trim();
    vazute.add(cheie);
  }
  return undefined;
}
```

- [ ] **Step 2: Adaugă regula CSS pentru lista de furnizori**

La finalul lui `frontend/src/style.css`:

```css
.furnizor-rand {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid var(--border);
}
```

- [ ] **Step 3: Compilează și rulează testele**

Run: `cd frontend && npx tsc --noEmit && npm test`
Expected: fără erori de compilare; toate testele trec.

- [ ] **Step 4: Verifică în aplicație**

Run: `wails dev`

Verifică:
1. Setări arată unitatea, numărul următor, cota implicită, tabelul de produse
   cu nota de avertizare deasupra și lista de furnizori.
2. Adaugă un produs, mută-l cu săgețile, salvează, reîncarcă ecranul — ordinea
   se păstrează.
3. Două produse cu aceeași denumire (cu majuscule diferite) → mesaj care spune
   care e duplicatul, nimic nu se salvează.
4. **Testul cel mai important:** salvează o notă cu un produs, apoi schimbă în
   Setări denumirea și prețul acelui produs, salvează, întoarce-te la notă —
   nota arată exact ce arăta înainte.
5. Șterge un produs din Setări și întoarce-te la nota care îl folosea: rândul e
   întreg.
6. Șterge un furnizor din listă și verifică în formular că nu mai apare între
   sugestii, dar nota veche care îl are îl păstrează.

- [ ] **Step 5: Commit**

```bash
git add frontend/src
git commit -m "$(cat <<'EOF'
Ecranul Setari

Unitatea, numaratorul, cota implicita, catalogul de produse cu
reordonare si lista de furnizori memorati. Nota de deasupra catalogului
spune explicit ca modificarile nu ating notele salvate.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

---

### Task 16: Instalatoare și documentație

**Files:**
- Create: `Makefile`, `README.md`
- Create: `build/appicon.png`, `build/darwin/Info.plist`, `build/darwin/Info.dev.plist`, `build/darwin/make-dmg.sh`, `build/darwin/CITEȘTE-MĂ.txt`
- Create: `build/windows/icon.ico`, `build/windows/info.json`, `build/windows/wails.exe.manifest`, `build/windows/installer/project.nsi`, `build/windows/installer/wails_tools.nsh`

**Interfaces:**
- Consumes: `wails.json` (versiunea din `info.productVersion`).
- Produces: `make mac` → `dist/NotaDeReceptie-<versiune>-macOS.dmg`; `make windows` → `dist/NotaDeReceptie-<versiune>-Windows-Setup.exe`.

- [ ] **Step 1: Generează scheletul de build**

Run: `wails build -clean`
Expected: `wails` creează `build/` cu iconițele și fișierele de platformă
implicite, apoi construiește aplicația. Confirmă că `build/bin/nota-de-receptie.app`
pornește.

- [ ] **Step 2: Copiază fișierele de împachetare din aplicația soră**

```bash
REF=/Users/roxanasuciu/git/proces-verbal-transare
cp "$REF/build/darwin/make-dmg.sh" build/darwin/
cp "$REF/build/windows/installer/project.nsi" build/windows/installer/
cp "$REF/build/windows/installer/wails_tools.nsh" build/windows/installer/
chmod +x build/darwin/make-dmg.sh
```

Deschide `build/windows/installer/project.nsi` și `build/darwin/Info.plist` și
înlocuiește peste tot numele produsului și identificatorul de bundle:
`Proces Verbal de Transare` → `Nota de receptie`,
`proces-verbal-transare` → `nota-de-receptie`, și identificatorul
`com.largiana.proces-verbal-transare` → `com.largiana.nota-de-receptie`.

Verifică: `grep -ri "proces\|transare" build/ | grep -v Binary` nu trebuie să
întoarcă nimic.

- [ ] **Step 3: Scrie `build/darwin/CITEȘTE-MĂ.txt`**

```
Notă de recepție
================

Instalare
---------

1. Trage aplicația peste folderul "Applications".

2. La prima pornire, macOS o poate refuza („dezvoltator neidentificat" sau
   „aplicația este deteriorată"), pentru că nu este semnată digital.
   Se rezolvă o singură dată, din Terminal:

   xattr -dr com.apple.quarantine "/Applications/Nota de receptie.app"

Date
----

Baza de date se creează la prima pornire în:

   ~/Library/Application Support/nota-de-receptie/data.db

Ea NU este ștearsă când muți sau ștergi aplicația.
```

- [ ] **Step 4: Scrie `Makefile`**

```makefile
# Instalatoare pentru Nota de receptie.
#
#   make mac       -> dist/NotaDeReceptie-<versiune>-macOS.dmg
#   make windows   -> dist/NotaDeReceptie-<versiune>-Windows-Setup.exe
#   make all       -> ambele
#
# Versiunea are o singura sursa de adevar: campul info.productVersion din wails.json.

VERSION      := $(shell python3 -c "import json;print(json.load(open('wails.json'))['info']['productVersion'])")
PRODUCT_NAME := Nota de receptie
DIST         := dist
APP_BUNDLE   := build/bin/$(PRODUCT_NAME).app
DMG          := $(DIST)/NotaDeReceptie-$(VERSION)-macOS.dmg
NSIS_OUT     := build/bin/nota-de-receptie-amd64-installer.exe
SETUP_EXE    := $(DIST)/NotaDeReceptie-$(VERSION)-Windows-Setup.exe

.PHONY: all mac windows clean check-wails check-nsis version test

all: mac windows

version:
	@echo $(VERSION)

test:
	go test ./...
	cd frontend && npm test

check-wails:
	@command -v wails >/dev/null 2>&1 || { \
		echo "Eroare: 'wails' nu este instalat."; \
		echo "  go install github.com/wailsapp/wails/v2/cmd/wails@latest"; \
		exit 1; }

check-nsis:
	@command -v makensis >/dev/null 2>&1 || { \
		echo "Eroare: 'makensis' (NSIS) nu este instalat - necesar pentru instalatorul Windows."; \
		echo "  brew install nsis"; \
		exit 1; }

## macOS: bundle universal (Intel + Apple Silicon) impachetat intr-un .dmg
mac: check-wails
	wails build -platform darwin/universal -clean
	@rm -rf "$(APP_BUNDLE)"
	mv "build/bin/nota-de-receptie.app" "$(APP_BUNDLE)"
	@mkdir -p $(DIST)
	build/darwin/make-dmg.sh "$(APP_BUNDLE)" "$(DMG)" "$(PRODUCT_NAME)"
	@echo "Gata: $(DMG)"

## Windows: executabil amd64 + instalator NSIS per-utilizator
##
## NU adauga "-installscope user": scopul este fixat in
## build/windows/installer/project.nsi prin REQUEST_EXECUTION_LEVEL, iar flagul
## CLI ar redefini aceeasi constanta si ar opri compilarea.
windows: check-wails check-nsis
	wails build -platform windows/amd64 -clean -nsis
	@mkdir -p $(DIST)
	cp "$(NSIS_OUT)" "$(SETUP_EXE)"
	@echo "Gata: $(SETUP_EXE)"

clean:
	rm -rf $(DIST) build/bin
```

- [ ] **Step 5: Scrie `README.md`**

```markdown
# Notă de recepție

Aplicație desktop pentru completarea, arhivarea și tipărirea notei de recepție
folosite de S.C. Largiana Carn S.R.L.

## Cerințe

- Go 1.25+
- Node 20+
- Wails CLI v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

## Dezvoltare

```bash
wails dev
```

## Teste

```bash
make test          # sau, separat:
go test ./...
cd frontend && npm test
```

## Build

```bash
wails build                          # macOS
wails build -platform windows/amd64  # Windows, de pe macOS
```

Ambele dependențe native (`modernc.org/sqlite`, `github.com/go-pdf/fpdf`) sunt
pur Go, deci cross-compilarea nu are nevoie de un toolchain Windows.

## Instalatoare

```bash
make mac        # dist/NotaDeReceptie-1.0.0-macOS.dmg
make windows    # dist/NotaDeReceptie-1.0.0-Windows-Setup.exe
make all        # ambele
```

Versiunea are o singură sursă: câmpul `info.productVersion` din `wails.json`.

Pentru instalatorul Windows este nevoie de NSIS: `brew install nsis`.

Aplicația **nu este semnată digital**. Instalatoarele funcționează, dar sistemul
de operare afișează un avertisment la prima pornire — vezi *Instalare*.

## Instalare

### macOS

1. Deschide `.dmg` și trage aplicația peste *Applications*.
2. La prima pornire, dacă macOS o refuză, rulează o singură dată în Terminal:

   ```bash
   xattr -dr com.apple.quarantine "/Applications/Nota de receptie.app"
   ```

### Windows

1. Rulează `NotaDeReceptie-<versiune>-Windows-Setup.exe`.
2. La SmartScreen: **Informații suplimentare** → **Executare oricum**.

Instalarea nu cere drepturi de administrator. **Baza de date nu este ștearsă la
dezinstalare** — rămâne în `%AppData%\nota-de-receptie`.

## Date

Baza de date SQLite se creează la prima pornire în directorul de configurare al
utilizatorului:

- macOS: `~/Library/Application Support/nota-de-receptie/data.db`
- Windows: `%AppData%\nota-de-receptie\data.db`

La prima pornire nu există niciun produs și niciun furnizor: catalogul se
completează din ecranul **Setări**, sau direct din formular, cu butonul care
salvează în produse o denumire nouă.

## Formularul

Față de formularul tipizat, aplicația renunță la: blocul *Delegat* și *Mijloc de
transport*; *Cod fiscal* și *Achitat cu* din tabelul de livrare; *Cod*, *T.V.A.
deductibil* și *T.V.A. colectat* din tabelul de produse.

Adaosul comercial (în lei și în procent) și cota T.V.A. sunt coloane de lucru:
se văd pe ecran, nu se tipăresc. Cantitatea nu se totalizează, fiindcă U/M
poate fi *Buc.* pe un rând și *Kg.* pe altul.
```

- [ ] **Step 6: Construiește instalatorul de macOS**

Run: `make mac`
Expected: `dist/NotaDeReceptie-1.0.0-macOS.dmg`. Montează-l, trage aplicația în
Applications, pornește-o (cu `xattr -dr com.apple.quarantine` dacă e nevoie) și
verifică: se deschide pe o notă nouă, Setări funcționează, salvarea și
generarea PDF funcționează.

- [ ] **Step 7: Rulează toate testele**

Run: `make test`
Expected: PASS pe partea de Go și pe frontend.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
Instalatoare si documentatie

Makefile cu tinte pentru .dmg si instalator NSIS per-utilizator,
fisierele de impachetare adaptate din aplicatia sora si README-ul cu
instalarea, dezvoltarea, testele si abaterile fata de formularul tipizat.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01Fhrcn7T9TuFY1MLUUbDE5E
EOF
)"
```

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
	ID        int64   `json:"id"`
	ProductID *int64  `json:"productId"`
	Pozitie   int     `json:"pozitie"`
	Denumire  string  `json:"denumire"`
	UM        string  `json:"um"`
	Cantitate float64 `json:"cantitate"`
	// PretFaraTVA is the purchase price, typed in at reception: it varies
	// from delivery to delivery, so it is not carried on the product.
	PretFaraTVA float64 `json:"pretFaraTva"`
	CotaTVA     float64 `json:"cotaTva"`
	PretVanzare float64 `json:"pretVanzare"`
	// ValoareVanzareImpusa, when set, is the row's valoare la preț de vânzare
	// as it came from a proces verbal de transare, rather than the cantitate ×
	// pret de vânzare the other rows derive it from. Its presence is also what
	// makes the row read-only in the form: the figure it carries only means
	// anything beside the quantity and price it was worked out from.
	//
	// A pointer rather than a zero sentinel: a proces verbal whose "ce iese"
	// table comes to 0 — nothing but waste — is a valid document, and an
	// imposed zero has to be distinguishable from nothing imposed at all.
	ValoareVanzareImpusa *float64 `json:"valoareVanzareImpusa,omitempty"`
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

# Notă de recepție — design

Aplicație desktop pentru completarea, arhivarea și tipărirea notei de recepție
folosite de S.C. Largiana Carn S.R.L.

Modelul de referință este aplicația soră `proces-verbal-transare`: aceeași
stivă tehnică, aceeași împărțire în pachete, aceeași formă de interfață. Acest
document descrie doar ce este specific notei de recepție; unde nu spune altfel,
convenția din aplicația de referință se păstrează.

## 1. Formularul

Punctul de plecare este formularul tipizat „Notă de recepție”, cu următoarele
abateri cerute:

Se elimină:

- blocul **DELEGAT** și **MIJLOC DE TRANSPORT**;
- din primul tabel: **COD FISCAL** și **ACHITAT CU**;
- din al doilea tabel: **Cod**, **T.V.A. deductibil** și **T.V.A. colectat**
  (cele două coloane tăiate cu mâna pe exemplarul de hârtie);
- din PDF, coloana **Adaos comercial** — adaosul rămâne un instrument de lucru
  pe ecran, în lei și în procent, dar nu se tipărește.

Rămâne, în ordine:

1. **Antet:** UNITATEA · nr. · din \<data\>
2. **Tabelul de livrare:** DOCUMENT LIVRARE · NR. · DATA · FURNIZORUL
3. **Tabelul de produse:** Nr. crt. · DENUMIREA · U/M · Cantitatea ·
   Preț fără T.V.A. · Valoare fără T.V.A. · Valoare cu T.V.A. ·
   Preț de vânzare · Valoare la preț de vânzare, plus rândul TOTAL
4. **Subsol:** COMISIA DE RECEPȚIE, · ÎNTOCMIT, · GESTIONAR, — doar etichete cu
   spațiu de semnătură, fără nume tipărite și fără câmpuri în formular

U/M acceptă exact două valori: `Buc.` și `Kg.`.

## 2. Arhitectură

Go 1.25 + Wails v2, frontend TypeScript vanilla (fără framework) cu router pe
hash, SQLite prin `modernc.org/sqlite`, PDF prin `github.com/go-pdf/fpdf`.
Ambele dependențe native sunt pur Go, deci cross-compilarea către Windows nu
cere toolchain nativ.

```
main.go              pornirea Wails
app.go               obiectul legat de frontend; fiecare metodă exportată e un apel din UI
internal/appdir      unde stă baza de date
internal/model       structurile de date, partajate de store, calc, pdfdoc și legături
internal/store       persistența SQLite (schema, migrare, documente, produse, furnizori)
internal/calc        aritmetica documentului
internal/pdfdoc      randarea PDF
frontend/src         api.ts, calc.ts, format.ts, fuzzy.ts, router.ts, sidebar.ts,
                     dialog.ts, toast.ts, style.css
frontend/src/views   document.ts, setari.ts
```

Baza de date se creează la prima pornire în directorul de configurare al
utilizatorului:

- macOS: `~/Library/Application Support/nota-de-receptie/data.db`
- Windows: `%AppData%\nota-de-receptie\data.db`

Nu există șabloane. Spre deosebire de `proces-verbal-transare`, catalogul de
produse este unul singur, plat.

## 3. Model de date

```go
type Settings struct {
    UnitateNume string  // se copiază pe fiecare document nou
    NextNr      int     // numărul pe care îl primește următorul document
    CotaTVA     float64 // cota implicită pentru produse noi, în procente
}

type Product struct {
    ID          int64
    Denumire    string
    UM          string  // "Buc." | "Kg."
    PretVanzare float64 // cu T.V.A.
    CotaTVA     float64 // cota de achiziție implicită a acestui produs
    Ordine      int
}

type Document struct {
    ID                  int64
    Nr                  int
    Data                string // ISO YYYY-MM-DD
    Unitate             string // copiată din Settings la creare
    DocumentLivrare     string // text liber: "Factură", "Aviz", ...
    DocumentLivrareNr   string
    DocumentLivrareData string // ISO YYYY-MM-DD, poate lipsi
    Furnizor            string
    CreatedAt           string
    UpdatedAt           string
    Randuri             []Rand
}

type Rand struct {
    ID          int64
    ProductID   *int64 // legătură slabă; niciodată sursă de date la citire
    Pozitie     int
    Denumire    string
    UM          string
    Cantitate   float64
    PretFaraTVA float64 // prețul de achiziție, tastat la recepție
    CotaTVA     float64
    PretVanzare float64 // cu T.V.A.
}

type DocumentSummary struct {
    ID       int64
    Nr       int
    Data     string
    Furnizor string
}
```

**Fotografia produsului.** `Denumire`, `UM`, `PretVanzare` și `CotaTVA` se
copiază pe rând la salvare și de acolo se citesc mereu. Editarea catalogului nu
atinge documentele deja salvate — este cerința explicită a aplicației.
`ProductID` există doar ca să știm din ce produs a pornit rândul; este
`ON DELETE SET NULL`, iar un rând al cărui produs a fost șters rămâne complet.

**Unitatea** se copiază la fel, pe document. Un PDF retipărit peste un an
trebuie să arate ce arăta la semnare, nu ce scrie azi în Setări.

**Valorile derivate nu se stochează.** Valoarea fără TVA, valoarea cu TVA,
valoarea la preț de vânzare, adaosul și totalurile se recalculează din rânduri
de fiecare dată, de aceleași funcții pe care le folosesc și ecranul, și PDF-ul.
Un document salvat de o versiune mai veche rămâne astfel consistent cu el
însuși.

### Schema SQL

```sql
CREATE TABLE settings (
  id           INTEGER PRIMARY KEY CHECK (id = 1),
  unitate_nume TEXT    NOT NULL DEFAULT '',
  next_nr      INTEGER NOT NULL DEFAULT 1,
  cota_tva     REAL    NOT NULL DEFAULT 11
);

CREATE TABLE products (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  denumire     TEXT    NOT NULL,
  um           TEXT    NOT NULL DEFAULT 'Kg.',
  pret_vanzare REAL    NOT NULL DEFAULT 0,
  cota_tva     REAL    NOT NULL DEFAULT 11,
  ordine       INTEGER NOT NULL
);
CREATE UNIQUE INDEX idx_products_denumire ON products(denumire COLLATE NOCASE);

CREATE TABLE furnizori (
  id   INTEGER PRIMARY KEY AUTOINCREMENT,
  nume TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_furnizori_nume ON furnizori(nume COLLATE NOCASE);

CREATE TABLE documents (
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

CREATE TABLE document_rows (
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
CREATE INDEX idx_rows_document ON document_rows(document_id, pozitie);
```

La prima pornire se scrie doar rândul din `settings` (unitatea
`S.C. Largiana Carn S.R.L.`, `next_nr = 1`, `cota_tva = 11`). Catalogul de
produse și lista de furnizori pornesc goale.

## 4. Calcule

`internal/calc` este singura definiție a aritmeticii pe partea de Go;
`frontend/src/calc.ts` o oglindește pentru recalculul instant din formular, iar
testele celor două acoperă aceleași exemple.

```
valoareFaraTVA = round2(cantitate × pretFaraTVA)
valoareCuTVA   = round2(valoareFaraTVA × (1 + cotaTVA / 100))
valoareVanzare = round2(cantitate × pretVanzare)
adaos          = round2(valoareVanzare − valoareCuTVA)
adaosProcent   = round2(adaos / valoareCuTVA × 100)
```

Rotunjirea este la două zecimale, jumătate depărtându-se de zero.

Prețul de vânzare este **cu T.V.A.**, ca și valoarea cu T.V.A. din care se
scade. Adaosul este, prin definiția dată de utilizator, diferența dintre cât se
vinde marfa și cât a fost plătită; ambele capete trebuie exprimate la fel, altfel
cifra n-ar însemna nimic.

Când `valoareCuTVA` este zero (marfă primită gratuit, sau un rând încă
necompletat), `adaosProcent` nu se poate calcula; se afișează liniuță, nu
infinit și nu zero.

**Totaluri.** Se însumează `valoareFaraTVA`, `valoareCuTVA`, `valoareVanzare` și
`adaos` peste rânduri, iar `adaosProcent` total se calculează din totaluri
(`adaosTotal / valoareCuTVATotal × 100`), nu ca medie a procentelor de pe
rânduri — media ar da o cifră pe care n-o are niciun rând și pe care n-o
confirmă nicio adunare.

**Cantitatea nu se însumează peste unități diferite.** Totalul de cantitate se
calculează separat pe `Buc.` și pe `Kg.`. Pe ecran, panoul de totaluri arată
subtotalul fiecărei unități prezente în document. În PDF, celula de cantitate
din rândul TOTAL se completează doar când toate rândurile au aceeași unitate;
altfel rămâne goală, fiindcă un total care adună kilograme cu bucăți ar fi o
cifră falsă pe un document semnat.

## 5. Metodele expuse frontendului (`app.go`)

```go
GetSettings()   (model.Settings, error)
SaveSettings(model.Settings) error

ListProducts()  ([]model.Product, error)
SaveProducts([]model.Product) error         // înlocuiește tot catalogul
AddProduct(model.Product) (model.Product, error) // adaugă unul, din formular

ListFurnizori() ([]string, error)
DeleteFurnizor(nume string) error

ListDocuments() ([]model.DocumentSummary, error)
GetDocument(id int64) (model.Document, error)
NewDocumentDraft() (model.Document, error)
SaveDocument(model.Document) (model.Document, error)
DeleteDocument(id int64) error

ExportPDF(id int64) (string, error)
```

`NewDocumentDraft` întoarce un document nesalvat cu `Nr` din `settings.NextNr`,
`Data` de azi, `Unitate` din Setări și lista de rânduri goală.

`SaveDocument` face totul într-o singură tranzacție: scrie documentul și
rândurile, urcă `next_nr` peste numărul folosit dacă documentul e nou și
numărul lui atinge sau depășește contorul, și memorează furnizorul în tabela
`furnizori` dacă nu e deja acolo. Nu există fereastră în care salvarea să
raporteze eroare pentru un document care de fapt a fost scris.

`AddProduct` există ca să se poată salva un produs nou fără a părăsi formularul.
O denumire deja existentă (indiferent de majuscule) întoarce eroare, iar
formularul spune că produsul există deja.

`DeleteFurnizor` șterge doar sugestia. Documentele păstrează numele
furnizorului ca text, deci niciun document nu se schimbă.

## 6. Ecrane

Trei rute, ca în aplicația de referință: `#/document/new`, `#/document/:id`,
`#/setari`. Interfața este integral în limba română.

### Bara laterală

Buton „+ Document nou”, intrarea pentru documentul nesalvat (afișată doar cât e
deschis), istoricul documentelor salvate — NR, data, furnizorul — cel mai nou
primul, și legătura către Setări.

### Formularul de document

**Antet:** Unitatea (needitabilă, copiată din Setări), NR (numeric, editabil),
Data (text `ZZ/LL/AAAA`).

**Tabelul de livrare:** Document livrare (text liber), Nr., Data (text
`ZZ/LL/AAAA`, opțional), Furnizorul — text liber cu sugestii din istoric, sub
forma unei liste care se filtrează pe măsură ce se scrie.

**Tabelul de produse.** Pornește gol. „+ Adaugă rând” inserează un rând al cărui
câmp de denumire este o casetă cu căutare fuzzy peste catalog: se scrie, apar
produsele potrivite, alegerea unuia completează U/M, prețul de vânzare și cota.
Denumirea poate rămâne și una scrisă liber, care nu există în catalog; în acest
caz apare lângă rând un buton „Salvează în produse” care o adaugă în Setări cu
U/M, prețul de vânzare și cota de pe rând.

Coloanele pe ecran, în ordine: Nr. crt. · Denumirea · U/M (listă `Buc.`/`Kg.`) ·
Cantitatea · Preț fără T.V.A. · Cota T.V.A. % · Valoare fără T.V.A. · Valoare cu
T.V.A. · Preț de vânzare · Valoare la preț de vânzare · Adaos · Adaos %.

Cota T.V.A., adaosul și procentul de adaos sunt coloane de lucru: există doar pe
ecran, nu și pe formularul tipărit. Cota este editabilă pe rând, pornind de la
cota produsului sau, în lipsa lui, de la cota implicită din Setări. Coloanele de valori, adaosul și procentul
sunt read-only și se recalculează la fiecare tastă. Fiecare rând are un buton de
ștergere.

**Panoul de totaluri**, sub tabel — nu un rând pierdut printre coloanele
tabelului, ci un bloc cu etichete deasupra valorilor:

```
Total cantitate | Total valoare | Total valoare | Total valoare la | Total | Adaos
                | fără T.V.A.   | cu T.V.A.     | preț de vânzare  | adaos |   %
```

Sub „Total cantitate” se listează subtotalul fiecărei unități prezente
(`12,00 Kg.`, `5,00 Buc.`).

**Butoane:** Salvează · Printează (PDF) · Șterge. Ultimele două apar doar pe un
document deja salvat.

**Validare la salvare:** numărul trebuie să fie cel puțin 1, data trebuie să fie
o dată calendaristică reală, documentul trebuie să aibă cel puțin un rând, iar
fiecare rând trebuie să aibă denumire. Restul câmpurilor pot rămâne goale.

### Setări

- Unitatea, numărul următorului document, cota T.V.A. implicită.
- **Produse:** listă editabilă — denumire, U/M, preț de vânzare, cotă T.V.A. —
  cu adăugare, ștergere și reordonare. Deasupra listei, o notă că modificările
  nu ating documentele deja salvate. Denumirile duplicate sunt refuzate la
  salvare, cu mesaj care spune care este duplicatul.
- **Furnizori memorați:** lista sugestiilor, fiecare cu buton de ștergere.

## 7. PDF

A4 **landscape**, margini de 10 mm, deci 277 mm utili. Se tipăresc doar
rândurile completate; tabelul se încheie cu rândul TOTAL, fără grilă goală.

Lățimile coloanelor, în mm:

| Coloană | mm |
|---|---|
| Nr. crt. | 13 |
| DENUMIREA | 70 |
| U/M | 16 |
| Cantitatea | 26 |
| Preț fără T.V.A. | 28 |
| Valoare fără T.V.A. | 30 |
| Valoare cu T.V.A. | 30 |
| Preț de vânzare | 28 |
| Valoare la preț de vânzare | 36 |
| **Total** | **277** |

Structura paginii: antetul (Unitatea, nr., din data), titlul „NOTĂ DE RECEPȚIE”,
tabelul de livrare cu cele patru coloane rămase, tabelul de produse cu rândul
TOTAL, iar la bază cele trei etichete de semnătură — COMISIA DE RECEPȚIE, ·
ÎNTOCMIT, · GESTIONAR, — distribuite pe lățimea paginii, cu spațiu liber sub
ele.

Un rând nu se rupe niciodată peste pagini; la depășire se redesenează antetul de
tabel pe pagina următoare, ca în `pdfdoc` din aplicația de referință.

Diacriticele se pliază la ASCII înainte de scriere (`Fold`), fiindcă fonturile
de bază fpdf sunt latin-1 și ar tipări altfel caractere greșite.

Coloana Adaos nu apare în PDF, nici în lei, nici în procent. Funcțiile de calcul
al adaosului rămân în `internal/calc`, testate acolo — `pdfdoc` pur și simplu nu
le cere.

`ExportPDF` întreabă unde să salveze fișierul (nume implicit
`nota-de-receptie-<nr>-<data>.pdf`), scrie fișierul și îl deschide cu vizualizatorul
implicit al sistemului. Anularea dialogului întoarce un drum gol, nu o eroare.

## 8. Teste

**Go:**

- `internal/calc` — valorile pe rând, totalurile, adaosul și procentul,
  cazul `valoareCuTVA = 0`, totalul de cantitate pe unități amestecate.
- `internal/store` — deschidere și migrare pe o bază goală, salvarea și citirea
  unui document cu rânduri, urcarea contorului de numere, faptul că editarea
  unui produs nu schimbă un document deja salvat, ștergerea unui produs lasă
  rândul întreg cu `product_id` gol, refuzul denumirilor duplicate,
  memorarea și ștergerea furnizorilor.
- `internal/pdfdoc` — randarea întoarce un PDF nevid pentru un document cu
  rânduri și pentru unul gol; celula de cantitate din TOTAL e goală la unități
  amestecate; `Fold`.
- `app_test.go` — schița de document nou, salvarea cu recalcul, exportul.

**Vitest:** `calc.ts` (aceleași exemple ca partea de Go), `format.ts`
(parsare/afișare de numere cu virgulă și de date `ZZ/LL/AAAA`), `fuzzy.ts`
(potrivirea cu și fără diacritice, ordinea rezultatelor).

## 9. Livrare

`Makefile` cu `make mac`, `make windows`, `make all`, ca în
`proces-verbal-transare`: bundle universal macOS împachetat în `.dmg` și
instalator NSIS per-utilizator pentru Windows, cu versiunea citită din
`info.productVersion` din `wails.json`. `README.md` în română, cu instalare,
dezvoltare, teste și locul bazei de date.

## 10. În afara scopului

Nu se implementează, până nu sunt cerute: șabloane de documente, mai multe
gestiuni, export Excel/CSV, căutare în istoric, backup automat, semnare
digitală a instalatoarelor, cod fiscal și modalitate de plată (eliminate
deliberat din formular), delegat și mijloc de transport, coloanele de T.V.A.
deductibil și colectat.

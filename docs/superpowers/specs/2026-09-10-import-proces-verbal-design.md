# Import proces verbal de transare — design

Pe factura de la furnizor vine o carcasă alături de alte produse. Pentru
carcasă se întocmește un proces verbal de transare în aplicația soră
`proces-verbal-transare`: carcasa se taie, iar din ea ies bucăți care intră în
magazin la preț de vânzare. Pe nota de recepție trebuie să apară și carcasa, cu
valoarea ei de vânzare luată din acel proces verbal, lângă celelalte produse de
pe factură, introduse manual.

Documentul descrie cum ajunge acea valoare din aplicația soră pe nota de
recepție. Unde nu spune altfel, convențiile din
`2026-09-09-nota-de-receptie-design.md` se păstrează.

## 1. Ce vede utilizatorul

În formularul notei de recepție, lângă „+ Adaugă rând”, un buton nou:
**„+ Adaugă din proces verbal”**. Deschide o listă cu procesele verbale
existente — nr., data, gestiunea și totalul tabelului „ce iese” — cel mai
recent primul. Totalul apare în listă pentru că el este cifra care va intra pe
notă: utilizatorul o vede înainte să aleagă, nu după.

La confirmare, fiecare rând din tabelul **„ce intră”** al procesului verbal
devine un rând pe nota de recepție:

| Coloana pe notă | De unde vine |
| --- | --- |
| DENUMIREA | denumirea rândului de intrare, ca atare (`Carcasa`) |
| U/M | U/M-ul rândului de intrare, normalizat (`Kg` → `Kg.`) |
| Cantitatea | cantitatea rândului de intrare |
| Preț fără T.V.A. | prețul fără TVA al rândului de intrare |
| Cota T.V.A. | cota rândului de intrare |
| Valoare fără T.V.A. | derivată, ca la orice rând |
| Valoare cu T.V.A. | derivată, ca la orice rând |
| Preț de vânzare | valoarea de vânzare împărțită la cantitate, informativ |
| **Valoare la preț de vânzare** | **cota din totalul „ce iese” cu TVA a procesului verbal** |

Adaosul iese singur: `valoare la preț de vânzare − valoare cu T.V.A.` este
exact adaosul obținut din transare, calculat de `calc.valoriDin` fără nicio
linie de cod în plus.

Denumirea nu poartă nicio referință la procesul verbal — rândul arată pe hârtie
ca oricare altul.

## 2. Instantaneu, nu legătură

Importul copiază cifrele în nota de recepție și atât. Dacă procesul verbal este
modificat ulterior în cealaltă aplicație, nota rămâne neschimbată; un PDF
retipărit peste un an arată ce arăta când a fost semnat. Este principiul pe
care îl urmează deja tot codul — rândurile poartă propriile denumiri și prețuri
și nu recitesc niciodată din catalog (`model.Rand`).

Consecința care contează: între două click-uri pe buton, nota de recepție nu
depinde în niciun fel de cealaltă aplicație.

## 3. Rândurile importate sunt blocate

Câmpurile unui rând venit dintr-un proces verbal sunt `readonly` și marcate
vizual; se poate doar șterge rândul (și reimporta procesul verbal, dacă s-a
schimbat). Motivul este aritmetic, nu de gust: rândul poartă o valoare de
vânzare impusă, care înseamnă ceva doar alături de cantitatea și prețul din
care a fost calculată. Un utilizator care ar schimba cantitatea și ar lăsa
valoarea din procesul verbal lângă ea ar obține un rând în care cifrele nu se
mai leagă între ele — pe un act contabil.

Autocompletarea de produs nu se leagă pe aceste rânduri: un rând venit dintr-un
proces verbal nu pornește dintr-un produs din catalog.

## 4. Ordinea rândurilor

Rândurile venite din procese verbale stau **întotdeauna la sfârșitul
tabelului**, atât pe ecran cât și în PDF — nu doar la momentul importului. Un
rând manual adăugat după un import intră deasupra rândurilor de proces verbal.

Invariantul se impune într-un singur loc: în `store.SaveDocument`, imediat
înaintea buclei care scrie rândurile, `doc.Randuri` trece printr-o
**partiționare stabilă** — întâi rândurile obișnuite în ordinea lor, apoi cele
importate în ordinea lor. Poziția este deja derivată acolo din ordinea din
slice (`r.Pozitie = i`), iar `SaveDocument` întoarce documentul cu ordinea
corectată, deci ecranul se aliniază singur după salvare, indiferent ce a trimis
formularul.

`GetDocument` citește `ORDER BY pozitie`, deci PDF-ul primește ordinea corectă
fără nicio modificare în `internal/pdfdoc`.

În formular, „+ Adaugă rând” inserează noul rând înaintea primului rând
importat, ca ce se vede pe ecran să fie ce se va tipări.

## 5. Model de date

Un singur câmp nou pe `model.Rand`:

```go
// ValoareVanzareImpusa, when set, is the row's valoare la preț de vânzare as
// it came from a proces verbal de transare, rather than the cantitate ×
// pret de vânzare the other rows derive it from. Its presence is also what
// makes the row read-only: the figure it carries only means anything beside
// the quantity and price it was worked out from.
ValoareVanzareImpusa *float64 `json:"valoareVanzareImpusa"`
```

Pointer, nu `float64` cu 0 ca sentinelă: un proces verbal al cărui tabel „ce
iese” totalizează 0 (numai deșeu) este un caz valid, iar un zero impus trebuie
să se deosebească de „nu este impus nimic”.

Prezența pointerului este și singurul semnal după care formularul blochează
rândul. Cele două proprietăți coincid întotdeauna, deci un al doilea câmp ar fi
doar ceva ce poate ajunge să le contrazică.

## 6. Migrarea

Prima migrare reală a proiectului. `schemaVersion` trece de la 1 la 2.

Coloana intră în `schemaSQL` pentru bazele noi:

```sql
valoare_vanzare_impusa REAL
```

Pentru bazele existente:

```sql
ALTER TABLE document_rows ADD COLUMN valoare_vanzare_impusa REAL
```

`ALTER`-ul se condiționează de **absența coloanei** (`PRAGMA
table_info(document_rows)`), nu de versiune. Pe o bază proaspătă `schemaSQL` a
creat-o deja, deși `user_version` este încă 0; o migrare care s-ar uita doar la
versiune ar încerca s-o adauge a doua oară și ar pica la pornire. Verificarea pe
coloană este idempotentă în ambele cazuri.

Fără drop-and-recreate, cum cere comentariul existent din `internal/store/schema.go`:
aplicația a fost livrată, iar fișierele conțin documente pe care nu le mai
retastează nimeni.

## 7. Calculul

O singură ramură în `calc.ValoriRand`, oglindită în
`frontend/src/calc.ts:valoriRand`:

```go
vanzare := Round2(r.Cantitate * r.PretVanzare)
if r.ValoareVanzareImpusa != nil {
    vanzare = Round2(*r.ValoareVanzareImpusa)
}
```

`Totaluri` nu se atinge: sumează ce îi dă `ValoriRand`, deci totalul notei
preia automat valoarea exactă. La fel adaosul.

### Rotunjirea

Cerința este ca **totalul să fie exact cât procesul verbal**, nu ca înmulțirea
cantitate × preț unitar să se verifice. Pe o carcasă de 162,2 Kg cu total „ce
iese” 2.501,35 lei, prețul unitar rotunjit la două zecimale (15,42) înmulțit
înapoi dă 2.501,12 — cu 23 de bani mai puțin. Pe cantități mari abaterea poate
ajunge la ~0,80 lei.

Rândul păstrează deci valoarea așa cum vine din procesul verbal și o tipărește
așa. Prețul unitar rămâne informativ, cu două zecimale ca tot restul
formularului. Cine verifică coloana cu calculatorul găsește diferența de câțiva
bani; nimic din formular nu pretinde că cele două coloane s-ar înmulți exact.

### Împărțirea proporțională

Pentru un proces verbal cu `n` rânduri de intrare și total „ce iese” cu TVA `T`:

- `v_i` = valoarea cu TVA a rândului de intrare `i` (cantitate × preț cu TVA)
- `cota_i = Round2(T × v_i / Σv)` pentru toate rândurile mai puțin unul
- rândul cu `v_i` cel mai mare primește `T − suma celorlalte`; la egalitate,
  primul dintre ele

Restul de rotunjire merge la rândul cel mai mare pentru că acolo cântărește
proporțional cel mai puțin. Condiția pe care algoritmul o garantează este
`Σcota_i = T`, exact.

Cazuri limită:

- `Σv = 0` (intrări la preț zero) → împărțire după cantitate
- și cantitatea 0 → tot `T` pe primul rând
- cantitate 0 pe un rând → prețul unitar afișat este 0
- proces verbal fără niciun rând de intrare → import refuzat cu mesaj, în loc
  de a adăuga zero rânduri pe tăcute

Cazul obișnuit — un singur rând de intrare — trece prin aceeași cale, fără
ramură specială: `Σv = v_1`, iar cota este `T` întreg.

### Maparea câmpurilor

- U/M: `"Kg"` → `"Kg."`, `"Buc"` → `"Buc."`, orice altceva rămâne ca atare
- denumire goală (coloana din cealaltă bază admite `''`) → `"Proces verbal nr. N"`
- `ProductID` rămâne `nil`

## 8. Citirea din baza aplicației surori

Pachet nou `internal/pvt`, singurul loc din proiect care știe că mai există o
aplicație.

```go
func DBPath() (string, error)                  // sibling of appdir, no MkdirAll
func List() (Lista, error)                     // procesele verbale, cu totalul „ce iese”
func Import(id int64) ([]model.Rand, error)    // rândurile gata de adăugat
```

`DBPath` refolosește `os.UserConfigDir()` exact ca `internal/appdir`, dar cu
`"proces-verbal-transare"` și **fără `MkdirAll`**: nota de recepție nu are ce
să creeze în folderul altcuiva, iar un director absent înseamnă că aplicația
aceea nu este instalată — un răspuns, nu o eroare de reparat.

Pe Windows, ambele instalatoare sunt per-utilizator
(`REQUEST_EXECUTION_LEVEL "user"`), deci ambele aplicații rulează ca același
utilizator, iar cele două fișiere sunt frați în același profil:

```
%APPDATA%\nota-de-receptie\data.db
%APPDATA%\proces-verbal-transare\data.db
```

Nu este nevoie de elevare și nu există problemă de permisiuni.

Cele două proiecte sunt module Go separate, deci `pvt` nu poate importa
modelul celuilalt. Își declară propriile structuri minimale, doar cu coloanele
pe care le citește. Duplicarea este intenționată: acesta *este* contractul
dintre aplicații, scris pe față, nu un tip împrumutat care s-ar schimba sub noi
la un update al celuilalt proiect.

### Deschiderea

```go
sql.Open("sqlite", path+"?_pragma=query_only(1)&_pragma=busy_timeout(3000)")
```

Handle propriu, deschis la click și închis imediat, `SetMaxOpenConns(1)`.

**Niciodată `store.Open`.** Acela cheamă `migrate`, care ar ștampila
`user_version = 1` peste baza celuilalt proiect și ar crea în ea tabelele notei
de recepție. La următoarea pornire, aplicația soră ar vedea `1 < 5` și
`dropAllTables` i-ar șterge tot conținutul. `query_only(1)` face imposibilă
orice scriere chiar dacă cineva greșește mai târziu.

### Refuzuri curate

Trei situații primesc mesaj în română în loc de o eroare tehnică:

- **fișierul lipsește** — aplicația nu este instalată, sau nu a fost pornită
  niciodată (baza se creează la prima rulare, nu de instalator)
- **`user_version < 5`** — bază pre-release, pe care aplicația soră o va șterge
  singură la următoarea pornire; nu citim din ea
- **jurnal rămas după un crash** — o bază deschisă read-only nu poate derula un
  jurnal fierbinte, iar SQLite răspunde `SQLITE_READONLY_ROLLBACK`. Se rezolvă
  când utilizatorul deschide o dată cealaltă aplicație, și exact asta îi spunem.

Peste verificarea de versiune, `pvt` confirmă prin `PRAGMA table_info` că
există coloanele de care are nevoie. Un update viitor al celuilalt proiect care
mută ceva dă atunci un mesaj clar, nu rânduri greșite pe un act contabil.

## 9. Metodele expuse frontendului

```go
func (a *App) ListProceseVerbale() (pvt.Lista, error)
func (a *App) ImportProcesVerbal(id int64) ([]model.Rand, error)
```

```go
type Sumar struct {
    ID       int64   `json:"id"`
    Nr       int     `json:"nr"`
    Data     string  `json:"data"`     // ISO YYYY-MM-DD
    Gestiune string  `json:"gestiune"`
    Total    float64 `json:"total"`    // totalul „ce iese” cu TVA
}

type Lista struct {
    Disponibil bool    `json:"disponibil"`
    Procese    []Sumar `json:"procese"` // cel mai recent primul
}
```

`Lista` poartă `Disponibil bool` lângă rânduri: „aplicația nu este instalată” și
„este instalată, dar nu are niciun proces verbal” sunt două lucruri diferite și
cer două mesaje diferite, pe care o listă goală singură nu le poate deosebi.

Toată aritmetica împărțirii stă în Go, iar `ImportProcesVerbal` întoarce rânduri
gata calculate. Aceleași cifre ajung pe ecran și în PDF, dintr-un singur loc.

## 10. Interfața

Butonul este mereu vizibil. Dacă aplicația soră lipsește, utilizatorul află
când apasă, printr-o propoziție care spune de ce — mai bine decât un buton care
nu apare niciodată și pe care nimeni nu-l caută.

Lista se desenează cu tiparul existent din `dialog.ts` (un `<dialog>` construit
per apel, ca unul deschis din handler-ul altuia să nu moștenească stare), cu o
variantă nouă care primește o listă în loc de un mesaj.

Rândurile importate se desenează cu `readonly` pe casete și o clasă care le dă
un fundal discret, plus butonul `×` neschimbat. Blocarea se citește direct din
`valoareVanzareImpusa != null`, deci formularul nu ține un al doilea steag pe
care ar trebui să-l sincronizeze.

## 11. PDF

Nicio modificare în `internal/pdfdoc`. `randCells` tipărește deja
`v.ValoareVanzare` venit din `calc.ValoriRand`, care întoarce acum valoarea
impusă. Lățimile coloanelor rămân neschimbate.

## 12. Teste

În ordinea în care se scriu, fiecare înaintea codului lui.

**Aritmetică** (`calc_test.go` și oglinda din `calc.test.ts`, cu aceleași
cifre): rând cu valoare impusă; total peste un rând impus și unul normal;
adaosul rezultat pe rândul impus.

**Împărțirea proporțională**: un singur rând de intrare primește tot totalul;
două rânduri cu rest de rotunjire — se verifică nu doar cotele, ci că **suma
lor este exact `T`**; `Σv = 0`; cantitate 0.

**Ordinea**: `SaveDocument` cu un rând importat la mijloc → în bază și în
documentul întors, acesta este ultimul, iar rândurile obișnuite își păstrează
ordinea între ele.

**Migrarea**: o bază v1 cu documente și rânduri reale → după migrare are
coloana nouă și toate rândurile intacte; o bază proaspătă trece prin aceeași
funcție fără să încerce `ALTER` a doua oară.

**Citirea**: teste peste o bază-fixtură construită în test cu forma celuilalt
proiect (`user_version = 5`). Duplicarea schemei în fixtură este intenționată —
este contractul dintre aplicații, iar un test care pică este exact avertismentul
de care avem nevoie. Plus căile de refuz: fișier lipsă, `user_version < 5`,
coloană dispărută.

**Testul care contează cel mai mult**: după un `List` și un `Import`, fișierul
celuilalt proiect are aceeași dimensiune, același `mtime`, același
`user_version` și același conținut ca înainte.

**PDF**: un document cu un rând impus se randează cu totalul exact al procesului
verbal.

Verificare finală: `go test ./...`, `npm test` în `frontend`, plus o rulare
reală cu un import dus până la PDF.

## 13. În afara scopului

- **Adăugarea aceluiași proces verbal pe două note.** Nimic nu o împiedică. Ar
  cere ca nota să rețină ce procese verbale a consumat și să întrebe cealaltă
  bază la fiecare deschidere — adică exact dependența permanentă respinsă la
  secțiunea 2. Se poate adăuga separat dacă devine o problemă în practică.
- **Scrierea în baza aplicației surori**, sub orice formă.
- **Reimportul automat** al unui proces verbal modificat. Utilizatorul șterge
  rândurile și importă din nou.
- **Referința la procesul verbal în PDF.** Denumirea rămâne curată.

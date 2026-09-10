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

## Import din „Proces verbal de transare"

Când carcasa de pe factură a fost transată, valoarea ei de vânzare se ia din
procesul verbal întocmit în aplicația soră, cu butonul **„+ Adaugă din proces
verbal"** din formularul notei.

Cele două aplicații nu comunică între ele: nota citește o singură dată fișierul
celeilalte, read-only, și copiază cifrele. Aplicația soră trebuie instalată și
pornită cel puțin o dată pe același cont de utilizator Windows, fiindcă baza ei
de date se creează la prima pornire:

- Windows: `%APPDATA%\proces-verbal-transare\data.db`
- macOS: `~/Library/Application Support/proces-verbal-transare/data.db`

Rândurile importate nu se pot edita — poartă valoarea din procesul verbal, care
înseamnă ceva doar alături de cantitatea și prețul din care a fost calculată —
și stau întotdeauna la finalul tabelului. Se pot șterge și reimporta.

## Build

```bash
wails build                          # macOS
wails build -platform windows/amd64  # Windows, de pe macOS
```

Ambele dependențe native (`modernc.org/sqlite`, `github.com/go-pdf/fpdf`) sunt
pur Go, deci cross-compilarea nu are nevoie de un toolchain Windows.

## Instalatoare

```bash
make mac        # dist/NotaDeReceptie-1.0.1-macOS.dmg
make windows    # dist/NotaDeReceptie-1.0.1-Windows-Setup.exe
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

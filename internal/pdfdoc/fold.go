// Package pdfdoc renders a notă de recepție to a PDF mirroring the paper form.
package pdfdoc

import "strings"

// echivalente maps characters the PDF cannot print onto ASCII that says the
// same thing.
//
// The Romanian diacritics are the reason it exists — both the comma-below
// (correct) and the cedilla (legacy) forms of s and t are covered — but a
// supplier name or a document number typed anywhere but this app arrives with
// whatever a word processor put in it: an em dash, curly quotes, a
// non-breaking space, an accented letter from a neighbouring language. Those
// are here too, because nothing downstream can render them (see Fold).
var echivalente = map[rune]string{
	// Romanian.
	'ă': "a", 'Ă': "A",
	'â': "a", 'Â': "A",
	'î': "i", 'Î': "I",
	'ș': "s", 'Ș': "S",
	'ş': "s", 'Ş': "S",
	'ț': "t", 'Ț': "T",
	'ţ': "t", 'Ţ': "T",

	// Punctuation and spacing a text editor introduces on its own.
	'–': "-", '—': "-", '―': "-", '−': "-", '‐': "-", '‑': "-",
	'‘': "'", '’': "'", '‚': ",", '‹': "'", '›': "'", '′': "'",
	'“': "\"", '”': "\"", '„': "\"", '«': "\"", '»': "\"", '″': "\"",
	'…': "...", '•': "-", '·': "-", '‰': "0/00",
	' ': " ", ' ': " ", ' ': " ", ' ': " ",
	'\t': " ", '\n': " ", '\r': " ",

	// Currency and symbols that turn up in a supplier's name or address.
	'€': "EUR", '£': "GBP", '¥': "JPY", '§': "S", '©': "(c)", '®': "(R)",
	'™': "(TM)", '×': "x", '÷': "/", '°': "deg", 'º': "o", 'ª': "a",
	'µ': "u", '¼': "1/4", '½': "1/2", '¾': "3/4",

	// Latin letters from the languages around this one.
	'á': "a", 'à': "a", 'ä': "a", 'å': "a", 'ã': "a", 'æ': "ae",
	'Á': "A", 'À': "A", 'Ä': "A", 'Å': "A", 'Ã': "A", 'Æ': "AE",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'ě': "e",
	'É': "E", 'È': "E", 'Ê': "E", 'Ë': "E", 'Ě': "E",
	'í': "i", 'ì': "i", 'ï': "i",
	'Í': "I", 'Ì': "I", 'Ï': "I",
	'ó': "o", 'ò': "o", 'ô': "o", 'ö': "o", 'õ': "o", 'ø': "o", 'ő': "o",
	'Ó': "O", 'Ò': "O", 'Ô': "O", 'Ö': "O", 'Õ': "O", 'Ø': "O", 'Ő': "O",
	'ú': "u", 'ù': "u", 'û': "u", 'ü': "u", 'ű': "u",
	'Ú': "U", 'Ù': "U", 'Û': "U", 'Ü': "U", 'Ű': "U",
	'ç': "c", 'Ç': "C", 'č': "c", 'Č': "C", 'ć': "c", 'Ć': "C",
	'ñ': "n", 'Ñ': "N", 'ň': "n", 'Ň': "N",
	'š': "s", 'Š': "S", 'ž': "z", 'Ž': "Z", 'ź': "z", 'Ź': "Z", 'ż': "z", 'Ż': "Z",
	'ř': "r", 'Ř': "R", 'ł': "l", 'Ł': "L", 'đ': "d", 'Đ': "D", 'ď': "d", 'Ď': "D",
	'ý': "y", 'Ý': "Y", 'ÿ': "y", 'ğ': "g", 'Ğ': "G", 'ı': "i", 'İ': "I",
	'ß': "ss", 'þ': "th", 'Þ': "Th", 'ð': "d", 'Ð': "D",
}

// nereprezentabil stands in for a character with no ASCII equivalent at all.
const nereprezentabil = "?"

// Fold rewrites a string as printable ASCII.
//
// The paper form is printed without diacritics, and the PDF could not carry
// them anyway: the core fonts fpdf draws with are Latin-1 and the bytes are
// handed to them untranslated, so anything above ASCII comes out as the raw
// UTF-8 bytes — an em dash pasted into a supplier name printed as "â€"".
// Everything that cannot be represented is therefore replaced here, never
// passed through.
//
// A character with no equivalent becomes a single "?", and a run of them
// becomes one "?" rather than one per letter: a name written in an alphabet
// this form cannot print should show that something was dropped without
// filling the cell with punctuation. Dropping it silently would leave a blank
// where a supplier's name belongs, which reads as a field nobody filled in.
func Fold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inlocuitAnterior := false
	for _, r := range s {
		switch echivalent, cunoscut := echivalente[r]; {
		case r >= 0x20 && r < 0x7f:
			b.WriteRune(r)
			inlocuitAnterior = false
		case cunoscut:
			b.WriteString(echivalent)
			inlocuitAnterior = false
		case inlocuitAnterior:
			// Part of a run already marked with one "?".
		default:
			b.WriteString(nereprezentabil)
			inlocuitAnterior = true
		}
	}
	return b.String()
}

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
	"nota-de-receptie/internal/pvt"
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
//
// A failure here (a corrupt file, an unwritable directory, a full disk) is
// entirely plausible on a shipped desktop app. The user gets a Romanian
// dialog explaining what happened instead of a Go panic trace.
//
// What happens next is worth being plain about: startup returns with a.store
// still nil and asks Wails to quit, so the quit races the frontend's first
// bound call, every one of which dereferences a.store. The race is narrow
// rather than closed — MessageDialog blocks startup until the user dismisses
// it, and the quit follows immediately — and the thirteen bound methods are
// left without nil guards deliberately: thirteen checks against a window that
// is already on its way out would cost more clarity than they buy.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	path, err := appdir.DBPath()
	if err != nil {
		a.fatalStartupError(fmt.Sprintf(
			"Nu s-a putut determina locația bazei de date.\n\nEroare: %v", err))
		return
	}
	s, err := store.Open(path)
	if err != nil {
		a.fatalStartupError(fmt.Sprintf(
			"Nu s-a putut deschide baza de date de la:\n%s\n\nEroare: %v", path, err))
		return
	}
	a.store = s
}

// fatalStartupError shows a native error dialog and then asks Wails to quit.
// Used only from startup; see the note there about what it does not do.
func (a *App) fatalStartupError(message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.ErrorDialog,
		Title:   "Eroare la pornire",
		Message: message,
	})
	runtime.Quit(a.ctx)
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

// ListProceseVerbale lists the proces verbal documents of the sibling
// application, or reports that it is not installed.
func (a *App) ListProceseVerbale() (pvt.Lista, error) { return pvt.List() }

// ImportProcesVerbal turns one proces verbal into rows ready to append to the
// note on screen.
func (a *App) ImportProcesVerbal(id int64) ([]model.Rand, error) { return pvt.Import(id) }

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

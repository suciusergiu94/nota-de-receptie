import {
  AddProduct,
  DeleteDocument,
  DeleteFurnizor,
  ExportPDF,
  GetDocument,
  GetSettings,
  ImportProcesVerbal,
  ListDocuments,
  ListFurnizori,
  ListProceseVerbale,
  ListProducts,
  NewDocumentDraft,
  SaveDocument,
  SaveProducts,
  SaveSettings,
} from '../wailsjs/go/main/App';
import { model, pvt } from '../wailsjs/go/models';
import { showAlert } from './dialog';

export type Document = model.Document;
export type DocumentSummary = model.DocumentSummary;
export type ProcesVerbalSumar = pvt.Sumar;
export type ProceseVerbale = pvt.Lista;
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
  ImportProcesVerbal,
  ListDocuments,
  ListFurnizori,
  ListProceseVerbale,
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

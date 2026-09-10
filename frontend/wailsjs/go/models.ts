export namespace model {
	
	export class Rand {
	    id: number;
	    productId?: number;
	    pozitie: number;
	    denumire: string;
	    um: string;
	    cantitate: number;
	    pretFaraTva: number;
	    cotaTva: number;
	    pretVanzare: number;
	    valoareVanzareImpusa?: number;
	
	    static createFrom(source: any = {}) {
	        return new Rand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.productId = source["productId"];
	        this.pozitie = source["pozitie"];
	        this.denumire = source["denumire"];
	        this.um = source["um"];
	        this.cantitate = source["cantitate"];
	        this.pretFaraTva = source["pretFaraTva"];
	        this.cotaTva = source["cotaTva"];
	        this.pretVanzare = source["pretVanzare"];
	        this.valoareVanzareImpusa = source["valoareVanzareImpusa"];
	    }
	}
	export class Document {
	    id: number;
	    nr: number;
	    data: string;
	    unitate: string;
	    documentLivrare: string;
	    documentLivrareNr: string;
	    documentLivrareData: string;
	    furnizor: string;
	    createdAt: string;
	    updatedAt: string;
	    randuri: Rand[];
	
	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nr = source["nr"];
	        this.data = source["data"];
	        this.unitate = source["unitate"];
	        this.documentLivrare = source["documentLivrare"];
	        this.documentLivrareNr = source["documentLivrareNr"];
	        this.documentLivrareData = source["documentLivrareData"];
	        this.furnizor = source["furnizor"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.randuri = this.convertValues(source["randuri"], Rand);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DocumentSummary {
	    id: number;
	    nr: number;
	    data: string;
	    furnizor: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nr = source["nr"];
	        this.data = source["data"];
	        this.furnizor = source["furnizor"];
	    }
	}
	export class Product {
	    id: number;
	    denumire: string;
	    um: string;
	    pretVanzare: number;
	    cotaTva: number;
	    ordine: number;
	
	    static createFrom(source: any = {}) {
	        return new Product(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.denumire = source["denumire"];
	        this.um = source["um"];
	        this.pretVanzare = source["pretVanzare"];
	        this.cotaTva = source["cotaTva"];
	        this.ordine = source["ordine"];
	    }
	}
	
	export class Settings {
	    unitateNume: string;
	    nextNr: number;
	    cotaTva: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unitateNume = source["unitateNume"];
	        this.nextNr = source["nextNr"];
	        this.cotaTva = source["cotaTva"];
	    }
	}

}

export namespace pvt {

	export class Sumar {
	    id: number;
	    nr: number;
	    data: string;
	    gestiune: string;
	    total: number;

	    static createFrom(source: any = {}) {
	        return new Sumar(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nr = source["nr"];
	        this.data = source["data"];
	        this.gestiune = source["gestiune"];
	        this.total = source["total"];
	    }
	}
	export class Lista {
	    disponibil: boolean;
	    procese: Sumar[];

	    static createFrom(source: any = {}) {
	        return new Lista(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.disponibil = source["disponibil"];
	        this.procese = this.convertValues(source["procese"], Sumar);
	    }

	    convertValues(a: any, classs: any, asMap: boolean = false): any {
	        if (!a) {
	            return a;
	        }
	        if (a.slice && a.map) {
	            return (a as any[]).map((elem) => this.convertValues(elem, classs));
	        } else if ("object" === typeof a) {
	            if (asMap) {
	                for (const key of Object.keys(a)) {
	                    a[key] = new classs(a[key]);
	                }
	                return a;
	            }
	            return new classs(a);
	        }
	        return a;
	    }
	}

}


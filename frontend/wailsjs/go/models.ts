export namespace app {
	
	export class FamilyView {
	    name: string;
	    patchedName: string;
	    folder: string;
	    license: string;
	    version: string;
	    monospaced: boolean;
	    description: string;
	    heading: string;
	    nameKind: string;
	    nameValue: string;
	    repo: string;
	    glyphs: number;
	    stars: number;
	    downloads: number;
	    gfRank: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new FamilyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.patchedName = source["patchedName"];
	        this.folder = source["folder"];
	        this.license = source["license"];
	        this.version = source["version"];
	        this.monospaced = source["monospaced"];
	        this.description = source["description"];
	        this.heading = source["heading"];
	        this.nameKind = source["nameKind"];
	        this.nameValue = source["nameValue"];
	        this.repo = source["repo"];
	        this.glyphs = source["glyphs"];
	        this.stars = source["stars"];
	        this.downloads = source["downloads"];
	        this.gfRank = source["gfRank"];
	        this.status = source["status"];
	    }
	}
	export class ListQuery {
	    text: string;
	    installedOnly: boolean;
	    mono: string;
	    license: string;
	    sort: string;
	
	    static createFrom(source: any = {}) {
	        return new ListQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.installedOnly = source["installedOnly"];
	        this.mono = source["mono"];
	        this.license = source["license"];
	        this.sort = source["sort"];
	    }
	}
	export class PreviewInfo {
	    fontFamily: string;
	    regularUrl: string;
	    boldUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fontFamily = source["fontFamily"];
	        this.regularUrl = source["regularUrl"];
	        this.boldUrl = source["boldUrl"];
	    }
	}

}

export namespace snippet {
	
	export class Span {
	    class: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new Span(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.class = source["class"];
	        this.text = source["text"];
	    }
	}

}


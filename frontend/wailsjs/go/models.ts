export namespace app {
	
	export class FamilyView {
	    name: string;
	    patchedName: string;
	    folder: string;
	    license: string;
	    version: string;
	    monospaced: boolean;
	    description: string;
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
	        this.status = source["status"];
	    }
	}
	export class ListQuery {
	    text: string;
	    installedOnly: boolean;
	    mono: string;
	    license: string;
	
	    static createFrom(source: any = {}) {
	        return new ListQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.installedOnly = source["installedOnly"];
	        this.mono = source["mono"];
	        this.license = source["license"];
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


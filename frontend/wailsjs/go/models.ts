export namespace main {
	
	export class ConfigInfo {
	    providerName: string;
	    model: string;
	    cacheEnabled: boolean;
	    servicePattern: string;
	    servicePatternIsEmpty: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerName = source["providerName"];
	        this.model = source["model"];
	        this.cacheEnabled = source["cacheEnabled"];
	        this.servicePattern = source["servicePattern"];
	        this.servicePatternIsEmpty = source["servicePatternIsEmpty"];
	    }
	}
	export class FileItem {
	    id: number;
	    originalPath: string;
	    originalName: string;
	    newName: string;
	    date: string;
	    service: string;
	    amount: string;
	    status: string;
	    error: string;
	    selected: boolean;
	    alreadyRenamed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.originalPath = source["originalPath"];
	        this.originalName = source["originalName"];
	        this.newName = source["newName"];
	        this.date = source["date"];
	        this.service = source["service"];
	        this.amount = source["amount"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.selected = source["selected"];
	        this.alreadyRenamed = source["alreadyRenamed"];
	    }
	}
	export class ProviderOption {
	    value: string;
	    label: string;
	    needsApiKey: boolean;
	    needsUrl: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	        this.needsApiKey = source["needsApiKey"];
	        this.needsUrl = source["needsUrl"];
	    }
	}
	export class RenameResult {
	    totalCount: number;
	    renamedCount: number;
	    errorCount: number;
	    skippedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new RenameResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalCount = source["totalCount"];
	        this.renamedCount = source["renamedCount"];
	        this.errorCount = source["errorCount"];
	        this.skippedCount = source["skippedCount"];
	    }
	}
	export class SettingsInfo {
	    provider: string;
	    model: string;
	    baseUrl: string;
	    hasApiKey: boolean;
	    apiKeySource: string;
	    cacheEnabled: boolean;
	    cacheCount: number;
	    servicePattern: string;
	    pdftotextPath: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.baseUrl = source["baseUrl"];
	        this.hasApiKey = source["hasApiKey"];
	        this.apiKeySource = source["apiKeySource"];
	        this.cacheEnabled = source["cacheEnabled"];
	        this.cacheCount = source["cacheCount"];
	        this.servicePattern = source["servicePattern"];
	        this.pdftotextPath = source["pdftotextPath"];
	    }
	}

}


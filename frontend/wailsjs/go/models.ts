export namespace main {

	export class DownloadPathInfo {
	    path: string;
	    defaultPath: string;
	    isDefault: boolean;
	    hasMigratableData: boolean;

	    static createFrom(source: any = {}) {
	        return new DownloadPathInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.defaultPath = source["defaultPath"];
	        this.isDefault = source["isDefault"];
	        this.hasMigratableData = source["hasMigratableData"];
	    }
	}
	export class PlatformInfo {
	    os: string;
	    name: string;
	    coreOS: string;
	    coreArch: string;
	    downloadPath: string;
	    defaultDownloadPath: string;
	    vfoxPathTarget: string;
	    sdkPathTarget: string;
	    shellProfile: string;
	    requiresElevation: boolean;
	    restartHint: string;
	
	    static createFrom(source: any = {}) {
	        return new PlatformInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.name = source["name"];
	        this.coreOS = source["coreOS"];
	        this.coreArch = source["coreArch"];
	        this.downloadPath = source["downloadPath"];
	        this.defaultDownloadPath = source["defaultDownloadPath"];
	        this.vfoxPathTarget = source["vfoxPathTarget"];
	        this.sdkPathTarget = source["sdkPathTarget"];
	        this.shellProfile = source["shellProfile"];
	        this.requiresElevation = source["requiresElevation"];
	        this.restartHint = source["restartHint"];
	    }
	}
	export class PluginInfo {
	    name: string;
	    isAdded: boolean;
	    isOfficial: boolean;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isAdded = source["isAdded"];
	        this.isOfficial = source["isOfficial"];
	        this.url = source["url"];
	    }
	}
	export class SdkVersionDetail {
	    version: string;
	    isCurrent: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SdkVersionDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.isCurrent = source["isCurrent"];
	    }
	}
	export class SdkDetail {
	    name: string;
	    current: string;
	    versions: SdkVersionDetail[];
	
	    static createFrom(source: any = {}) {
	        return new SdkDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.current = source["current"];
	        this.versions = this.convertValues(source["versions"], SdkVersionDetail);
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
	export class SdkEnvironmentImportResult {
	    path: string;
	    importedCustomSdks: number;
	    skippedCustomSdks: number;
	    vfoxSdksFound: number;
	    installedVfoxSdks: number;
	    skippedVfoxSdks: number;
	    warnings: string[];

	    static createFrom(source: any = {}) {
	        return new SdkEnvironmentImportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.importedCustomSdks = source["importedCustomSdks"];
	        this.skippedCustomSdks = source["skippedCustomSdks"];
	        this.vfoxSdksFound = source["vfoxSdksFound"];
	        this.installedVfoxSdks = source["installedVfoxSdks"];
	        this.skippedVfoxSdks = source["skippedVfoxSdks"];
	        this.warnings = source["warnings"];
	    }
	}
	export class SdkVersion {
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new SdkVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	    }
	}
	export class SdkInfo {
	    name: string;
	    source: string;
	    path: string;
	    versions: SdkVersion[];
	
	    static createFrom(source: any = {}) {
	        return new SdkInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.source = source["source"];
	        this.path = source["path"];
	        this.versions = this.convertValues(source["versions"], SdkVersion);
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


}

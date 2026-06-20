export namespace app {
	
	export class AppStatus {
	    coreRunning: boolean;
	    systemProxyEnabled: boolean;
	    autoStableEnabled: boolean;
	    activeProfileName: string;
	    controllerAddress: string;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.coreRunning = source["coreRunning"];
	        this.systemProxyEnabled = source["systemProxyEnabled"];
	        this.autoStableEnabled = source["autoStableEnabled"];
	        this.activeProfileName = source["activeProfileName"];
	        this.controllerAddress = source["controllerAddress"];
	        this.lastError = source["lastError"];
	    }
	}
	export class AutoStableNodeHealth {
	    name: string;
	    type?: string;
	    latencyMs?: number;
	    alive: boolean;
	    score?: number;
	    successCount?: number;
	    failureCount?: number;
	    totalChecks?: number;
	    failureRate?: number;
	    // Go type: time
	    lastCheckedAt?: any;
	    // Go type: time
	    cooldownUntil?: any;
	
	    static createFrom(source: any = {}) {
	        return new AutoStableNodeHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.latencyMs = source["latencyMs"];
	        this.alive = source["alive"];
	        this.score = source["score"];
	        this.successCount = source["successCount"];
	        this.failureCount = source["failureCount"];
	        this.totalChecks = source["totalChecks"];
	        this.failureRate = source["failureRate"];
	        this.lastCheckedAt = this.convertValues(source["lastCheckedAt"], null);
	        this.cooldownUntil = this.convertValues(source["cooldownUntil"], null);
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
	export class AutoStableGroupHealth {
	    name: string;
	    type: string;
	    selected?: string;
	    proxies: AutoStableNodeHealth[];
	
	    static createFrom(source: any = {}) {
	        return new AutoStableGroupHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.selected = source["selected"];
	        this.proxies = this.convertValues(source["proxies"], AutoStableNodeHealth);
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
	export class AutoStableActionResult {
	    action: string;
	    groupName?: string;
	    selected?: string;
	    changed: boolean;
	    message?: string;
	    // Go type: time
	    completedAt: any;
	    health?: AutoStableGroupHealth[];
	
	    static createFrom(source: any = {}) {
	        return new AutoStableActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.groupName = source["groupName"];
	        this.selected = source["selected"];
	        this.changed = source["changed"];
	        this.message = source["message"];
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.health = this.convertValues(source["health"], AutoStableGroupHealth);
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
	
	
	export class AutoStableStatus {
	    enabled: boolean;
	    available: boolean;
	    running: boolean;
	    // Go type: time
	    lastTickAt?: any;
	    lastAction?: string;
	    lastSelected?: string;
	    lastError?: string;
	    health: AutoStableGroupHealth[];
	
	    static createFrom(source: any = {}) {
	        return new AutoStableStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.available = source["available"];
	        this.running = source["running"];
	        this.lastTickAt = this.convertValues(source["lastTickAt"], null);
	        this.lastAction = source["lastAction"];
	        this.lastSelected = source["lastSelected"];
	        this.lastError = source["lastError"];
	        this.health = this.convertValues(source["health"], AutoStableGroupHealth);
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
	export class ConnectionStatus {
	    coreRunning: boolean;
	    uploadTotal: number;
	    downloadTotal: number;
	    connectionCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.coreRunning = source["coreRunning"];
	        this.uploadTotal = source["uploadTotal"];
	        this.downloadTotal = source["downloadTotal"];
	        this.connectionCount = source["connectionCount"];
	    }
	}
	export class LogLine {
	    // Go type: time
	    time: any;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = this.convertValues(source["time"], null);
	        this.level = source["level"];
	        this.message = source["message"];
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
	export class ProxyView {
	    name: string;
	    type?: string;
	    latencyMs?: number;
	    alive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProxyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.latencyMs = source["latencyMs"];
	        this.alive = source["alive"];
	    }
	}
	export class ProxyGroupView {
	    name: string;
	    type: string;
	    selected: string;
	    proxies: ProxyView[];
	
	    static createFrom(source: any = {}) {
	        return new ProxyGroupView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.selected = source["selected"];
	        this.proxies = this.convertValues(source["proxies"], ProxyView);
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


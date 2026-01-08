export namespace config {
	
	export interface ServiceConfig {
	    name: string;
	    path: string;
	    env: Record<string, string>;
	    type: string;
	}

}

export namespace process {
	
	export interface LogEntry {
	    timestamp: string;
	    level: string;
	    message: string;
	    raw: string;
	    stream: string;
	}

}

export namespace service {
	
	export interface Service {
	    ID: string;
	    Config: config.ServiceConfig;
	    InheritedEnv: Record<string, string>;
	    Status: string;
	    Logs: process.LogEntry[];
	    URL?: string;
	}
	export interface ServiceInfo {
	    name: string;
	    path: string;
	    status: string;
	    url?: string;
	    logs: process.LogEntry[];
	    env: Record<string, string>;
	    inheritedEnv: Record<string, string>;
	    type: string;
	}

}


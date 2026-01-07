export namespace config {
	
	export interface ServiceConfig {
	    name: string;
	    path: string;
	    env: Record<string, string>;
	}

}

export namespace process {
	
	export interface LogEntry {
	    timestamp: string;
	    level: string;
	    message: string;
	    raw: string;
	}

}

export namespace service {
	
	export interface Service {
	    ID: string;
	    Config: config.ServiceConfig;
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
	}

}


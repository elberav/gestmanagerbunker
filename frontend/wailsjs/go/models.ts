export namespace backend {
	
	export class Account {
	    id: number;
	    name: string;
	    email: string;
	    login_method: string;
	    password_enc: number[];
	    parent_id?: number;
	    status: string;
	    tag: string;
	    username: string;
	    url: string;
	    notes_enc: number[];
	    last_viewed?: string;
	    has_children: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.email = source["email"];
	        this.login_method = source["login_method"];
	        this.password_enc = source["password_enc"];
	        this.parent_id = source["parent_id"];
	        this.status = source["status"];
	        this.tag = source["tag"];
	        this.username = source["username"];
	        this.url = source["url"];
	        this.notes_enc = source["notes_enc"];
	        this.last_viewed = source["last_viewed"];
	        this.has_children = source["has_children"];
	    }
	}
	export class SecureFile {
	    id: number;
	    account_id?: number;
	    account_name: string;
	    account_email: string;
	    filename: string;
	    file_size: number;
	    uploaded_at: string;
	    tag: string;
	    comment: string;
	
	    static createFrom(source: any = {}) {
	        return new SecureFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.account_id = source["account_id"];
	        this.account_name = source["account_name"];
	        this.account_email = source["account_email"];
	        this.filename = source["filename"];
	        this.file_size = source["file_size"];
	        this.uploaded_at = source["uploaded_at"];
	        this.tag = source["tag"];
	        this.comment = source["comment"];
	    }
	}

}

export namespace main {
	
	export class UpdateInfo {
	    has_update: boolean;
	    latest_version: string;
	    download_url: string;
	    asset_name: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.has_update = source["has_update"];
	        this.latest_version = source["latest_version"];
	        this.download_url = source["download_url"];
	        this.asset_name = source["asset_name"];
	    }
	}

}


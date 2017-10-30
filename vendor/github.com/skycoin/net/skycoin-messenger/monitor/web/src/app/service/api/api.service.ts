import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/catch';
import 'rxjs/add/observable/throw';

@Injectable()
export class ApiService {
  private connUrl = '/conn/';
  private nodeUrl = '/node';
  private callbackParm = 'callback';
  constructor(private httpClient: HttpClient) { }
  getAllNode() {
    return this.handleGet(this.connUrl + 'getAll');
  }

  getNodeStatus(data: FormData) {
    return this.handlePost(this.connUrl + 'getNode', data);
  }
  getMsgs(addr) {
    return this.handleNodePost(addr, '/node/getMsgs');
  }
  getApps(addr: string) {
    return this.handleNodePost(addr, '/node/getApps');
  }

  getNodeInfo(addr: string) {
    return this.handleNodePost(addr, '/node/getInfo');
  }
  reboot(addr: string) {
    return this.handleNodePost(addr, '/node/reboot');
  }
  connectSSHClient(addr: string, data?: FormData) {
    return this.handleNodePost(addr, '/node/run/sshc', data);
  }
  runSSHServer(addr: string, data?: FormData) {
    return this.handleNodePost(addr, '/node/run/sshs', data);
  }
  runSockServer(addr: string) {
    return this.handleNodePost(addr, '/node/run/sockss');
  }
  checkUpdate(channel, vesrion: string) {
    const data = new FormData();
    data.append('addr', `http://messenger.skycoin.net:8100/api/version?c=${channel}&v=${vesrion}`);
    return this.handlePost(this.nodeUrl, data);
  }
  jsonp(url: string) {
    if (url === '') {
      return Observable.throw('Url is empty.');
    }
    return this.httpClient.jsonp(url, this.callbackParm).catch(err => Observable.throw(err));
  }
  handleGet(url: string) {
    if (url === '') {
      return Observable.throw('Url is empty.');
    }
    return this.httpClient.get(url).catch(err => Observable.throw(err));
  }
  handleNodePost(nodeAddr: string, api: string, data?: FormData) {
    if (nodeAddr === '' || api === '') {
      return Observable.throw('nodeAddr or api is empty.');
    }
    nodeAddr = 'http://' + nodeAddr + api;
    if (!data) {
      data = new FormData();
    }
    data.append('addr', nodeAddr);
    return this.handlePost(this.nodeUrl, data);
  }
  handlePost(url: string, data: FormData) {
    if (url === '') {
      return Observable.throw('Url is empty.');
    }
    return this.httpClient.post(url, data).catch(err => Observable.throw(err));
  }
}
export interface Conn {
  key?: string;
  type?: string;
  send_bytes?: number;
  recv_bytes?: number;
  last_ack_time?: number;
  start_time?: number;
}
export interface ConnData extends Conn {
  index?: number;
}
export interface ConnsResponse {
  conns?: Array<Conn>;
}

export interface NodeServices extends Conn {
  apps?: Array<App>;
  addr?: string;
}

export interface App {
  key?: string;
  attributes?: Array<string>;
  allow_nodes?: Array<string>;
}

export interface Transports {
  from_node?: string;
  to_node?: string;
  from_app?: string;
  to_app?: string;
}
export interface Message {
  priority?: number;
  type?: number;
  msg?: string;
}
export interface FeedBack {
  port?: number;
  failed?: boolean;
  msg?: Message;
}
export interface FeedBackItem {
  key?: string;
  feedbacks?: FeedBack;
}
export interface NodeInfo {
  transports?: Array<Transports>;
  messages?: Array<Array<Message>>;
  app_feedbacks?: Array<FeedBackItem>;
}

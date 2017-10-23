import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/catch';
import 'rxjs/add/observable/throw';

@Injectable()
export class ApiService {
  private url = 'http://127.0.0.1:8000/';
  private connUrl = this.url + 'conn/';
  private nodeUrl = this.url + 'node/';
  private callbackParm = 'callback';
  constructor(private httpClient: HttpClient) {
    if (environment.production) {
      this.connUrl = '/conn/';
      this.nodeUrl = '/node';
    }
  }
  getAllNode() {
    return this.handleGet(this.connUrl + 'getAll');
  }

  getNodeStatus(data: FormData) {
    return this.handlePost(this.connUrl + 'getNode', data);
  }
  getTransport(addr: string) {
    return this.handleNodePost('http://' + addr + '/node/getTransports');
  }
  reboot(addr: string) {
    return this.handleNodePost('http://' + addr + '/node/reboot');
  }
  runSSHServer(addr: string) {
    return this.handleNodePost('http://' + addr + '/node/run/sshs');
  }
  runSockServer(addr: string) {
    return this.handleNodePost('http://' + addr + '/node/run/sockss');
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
  handleNodePost(nodeAddr: any) {
    const data = new FormData();
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
  index?: number;
  key?: string;
  attributes?: Array<string>;
}

export interface Transports {
  from_node?: string;
  to_node?: string;
  from_app?: string;
  to_app?: string;
}

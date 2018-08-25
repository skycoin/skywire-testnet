import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { environment } from '../../../environments/environment';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/catch';
import 'rxjs/add/observable/throw';
import 'rxjs/add/observable/empty';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material';
import { AlertService } from '../alert/alert.service';

@Injectable()
export class ApiService {
  private connUrl = '/conn/';
  private nodeUrl = '/node';
  private reqUrl = '/req';
  private bankUrl = '52.15.100.203:8080/';
  private callbackParm = 'callback';
  private jsonHeader = { 'Content-Type': 'application/json' };
  constructor(
    private httpClient: HttpClient,
    private router: Router,
    private dialog: MatDialog,
    private alert: AlertService) { }

  addOrder(data: FormData) {
    return this.handleReq(this.bankUrl, 'skybank/order/addOrder', data);
  }
  getConvertible(data: FormData) {
    return this.handleReq(this.bankUrl, 'skybank/node/getConvertible', data);
  }
  getBalance(data: FormData) {
    return this.handleReq(this.bankUrl, 'skybank/node/get', data);
  }
  getNodeOrders(data: FormData) {
    return this.handleReq(this.bankUrl, 'skybank/order/getNodeOrder', data);
  }
  getSig(addr: string, hash: string) {
    const data = new FormData();
    data.append('data', hash);
    return this.handleReq(addr, '/node/getSig', data);
  }
  getServerInfo() {
    return this.handleGet(this.connUrl + 'getServerInfo', { responseType: 'text' });
  }
  closeApp(addr: string, data: FormData) {
    return this.handleReq(addr, '/node/run/closeApp', data);
  }
  login(data: FormData) {
    return this.handlePost('login', data);
  }
  updatePass(data: FormData) {
    return this.handlePost('updatePass', data);
  }

  getManagerPort() {
    return this.handlePost('getPort');
  }

  checkLogin() {
    return this.handlePost('checkLogin', null, { responseType: 'text' });
  }
  getAllNode() {
    return this.handleGet(this.connUrl + 'getAll');
  }
  getNodeStatus(data: FormData) {
    return this.handlePost(this.connUrl + 'getNode', data);
  }
  setNodeConfig(addr: string, data: FormData) {
    return this.handleReq(addr, '/node/run/setNodeConfig', data);
  }
  updateNodeConfig(addr: string) {
    return this.handleReq(addr, '/node/run/updateNode');
  }
  getMsgs(addr) {
    return this.handleReq(addr, '/node/getMsgs');
  }
  getApps(addr: string) {
    return this.handleReq(addr, '/node/getApps');
  }

  getNodeInfo(addr: string) {
    return this.handleReq(addr, '/node/getInfo');
  }
  reboot(addr: string) {
    return this.handleReq(addr, '/node/reboot');
  }
  getAutoStart(addr: string, data: FormData) {
    return this.handleReq(addr, '/node/run/getAutoStartConfig', data);
  }
  setAutoStart(addr: string, data?: FormData) {
    return this.handleReq(addr, '/node/run/setAutoStartConfig', data);
  }
  checkAppMsg(addr: string, data?: FormData) {
    return this.handleReq(addr, '/node/getMsg', data);
  }
  searchServices(addr: string, data?: FormData) {
    return this.handleReq(addr, '/node/run/searchServices', data);
  }
  getServicesResult(addr: string, data?: FormData) {
    return this.handleReq(addr, '/node/run/getSearchServicesResult', data);
  }
  connectSSHClient(addr: string, data?: FormData) {
    return this.handleReq(addr, '/node/run/sshc', data);
  }
  connectSocketClicent(addr: string, data?: FormData) {
    return this.handleReq(addr, '/node/run/socksc', data);
  }
  runSSHServer(addr: string, data?: FormData) {
    return this.handleReq(addr, '/node/run/sshs', data);
  }
  runSockServer(addr: string, data?: FormData) {
    return this.handleReq(addr, '/node/run/sockss', data);
  }
  runNodeupdate(addr: string) {
    return this.handleReq(addr, '/node/run/update');
  }
  getNodeupdateProcess(addr: string) {
    return this.handleReq(addr, '/node/run/updateProcess');
  }
  getDebugPage(addr: string) {
    return this.handleReq(addr, '/debug/pprof');
  }
  checkUpdate(addr: string) {
    return this.handleReq(addr, '/node/run/checkUpdate');
  }
  saveClientConnection(data: FormData) {
    return this.handlePost(this.connUrl + 'saveClientConnection', data);
  }
  removeClientConnection(data: FormData) {
    return this.handlePost(this.connUrl + 'removeClientConnection', data);
  }
  editClientConnection(data: FormData) {
    return this.handlePost(this.connUrl + 'editClientConnection', data);
  }
  SetClientAutoStart(data: FormData) {
    return this.handlePost(this.connUrl + 'setClientAutoStart', data);
  }
  getClientConnection(data: FormData) {
    return this.handlePost(this.connUrl + 'getClientConnection', data);
  }

  getWalletNewAddress(data: FormData) {
    return this.handleReq(this.bankUrl, 'skypay/tools/newAddress', data);
  }
  getWalletInfo(data: FormData) {
    return this.handleReq(this.bankUrl, 'skypay/node/get', data);
  }
  jsonp(url: string) {
    if (url === '') {
      return Observable.throw('Url is empty.');
    }
    return this.httpClient.jsonp(url, this.callbackParm).catch(err => Observable.throw(err));
  }
  handleGet(url: string, opts?: any) {
    if (url === '') {
      return Observable.throw('Url is empty.');
    }
    return this.httpClient.get(url, opts).catch(err => this.handleError(err));
  }
  handleReqOutside(url: string) {
    if (url === '') {
      return Observable.throw('url is empty.');
    }
    return this.handlePost(`${this.reqUrl}?addr=${url}`);
  }
  handleReq(addr: string, api: string, data?: FormData, opts?: any) {
    if (addr === '' || api === '') {
      return Observable.throw('nodeAddr or api is empty.');
    }
    addr = 'http://' + addr + api;
    return this.handlePost(`${this.reqUrl}?addr=${addr}`, data, opts);
  }
  handlePost(url: string, data?: FormData, opts?: any) {
    if (url === '') {
      return Observable.throw('Url is empty.');
    }
    return this.httpClient.post(url, data, opts).catch(err => this.handleNodeError(err));
  }
  handleNodeError(err: HttpErrorResponse) {
    this.dialog.closeAll();
    switch (err.status) {
      case 302:
        this.router.navigate([{ outlets: { user: ['login'] } }]);
        break;
      case 307:
        this.router.navigate(['updatePass']);
        break;
      default:
        break;
    }
    return Observable.throw(err.error.text);
  }
  handleError(err: HttpErrorResponse) {
    this.dialog.closeAll();
    switch (err.status) {
      case 302:
        this.router.navigate([{ outlets: { user: ['login'] } }]);
        break;
      case 307:
        this.router.navigate(['updatePass']);
        break;
      default:
        break;
    }
    return Observable.throw(err);
  }

}
export interface ConnectServiceInfo {
  label?: string;
  nodeKey?: string;
  appKey?: string;
  count?: number;
  auto_start?: boolean;
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
// export interface Message {
//   priority?: number;
//   type?: number;
//   msg?: string;
// }
export interface FeedBack {
  port?: number;
  failed?: boolean;
  msg?: Message;
}
export interface FeedBackItem {
  key?: string;
  failed?: boolean;
  port?: number;
  unread?: boolean;
}
export interface NodeInfo {
  version?: string;
  tag?: string;
  os?: string;
  discoveries?: Map<string, boolean>;
  transports?: Array<Transports>;
  messages?: Array<Message>;
  app_feedbacks?: Array<FeedBackItem>;
}

export interface Message {
  key?: string;
  read?: boolean;
  msgs?: Array<MessageItem>;
}
export interface MessageItem {
  msg?: string;
  priority?: number;
  time?: number;
  type?: number;
}

export interface AutoStartConfig {
  socks_server?: boolean;
  ssh_server?: boolean;
}

export interface WalletAddress {
  code?: number;
  address?: string;
}

export interface HashSig {
  pubkey?: string;
  timestamp?: number;
}

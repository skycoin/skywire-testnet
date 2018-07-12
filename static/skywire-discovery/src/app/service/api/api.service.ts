import {Injectable} from '@angular/core';
import {HttpClient, HttpErrorResponse} from '@angular/common/http';
import {Observable, throwError} from 'rxjs';
import {catchError, map} from 'rxjs/operators';
import {Router} from '@angular/router';
import {MatDialog} from '@angular/material';
import {AlertService} from '../alert/alert.service';

@Injectable()
export class ApiService {
  private connUrl = 'http://discovery.skycoin.net:8001/conn/';
  private reqUrl = '/req';
  private bankUrl = '52.15.100.203:8080/';

  constructor(
    private httpClient: HttpClient,
    private router: Router,
    private dialog: MatDialog,
    private alert: AlertService) {
  }

  getServerInfo(): Observable<string> {
    return this.handleGet(this.connUrl + 'getServerInfo', {responseType: 'text'}).pipe(
      map((x: string) => x)
    );
  }

  login(data: FormData) {
    return this.handlePost('login', data);
  }

  updatePass(data: FormData) {
    return this.handlePost('updatePass', data);
  }

  getAllNode(): Observable<Array<Conn>> {
    return this.handleGet(this.connUrl + 'getAll').pipe(
      map((x: Conn[]) => x)
    );
  }

  handleGet(url: string, opts?: any) {
    if (url === '') {
      return throwError('Url is empty.');
    }
    return this.httpClient.get(url, opts).pipe(
      catchError(err => this.handleError(err))
    );
  }


  handleReq(addr: string, api: string, data?: FormData, opts?: any) {

    if (addr === '' || api === '') {
      return throwError('nodeAddr or api is empty.');
    }
    addr = 'http://' + addr + api;
    return this.handlePost(`${this.reqUrl}?addr=${addr}`, data, opts);
  }

  handlePost(url: string, data?: FormData, opts?: any) {
    if (url === '') {
      return throwError('Url is empty.');
    }
    return this.httpClient.post(url, data, opts).pipe(
      catchError(err => this.handleNodeError(err))
    );
  }

  handleNodeError(err: HttpErrorResponse) {
    this.dialog.closeAll();
    switch (err.status) {
      case 302:
        this.router.navigate([{outlets: {user: ['login']}}]);
        break;
      case 307:
        this.router.navigate(['updatePass']);
        break;
      default:
        break;
    }
    return throwError(err.error.text);
  }

  handleError(err: HttpErrorResponse) {
    this.dialog.closeAll();
    switch (err.status) {
      case 302:
        this.router.navigate([{outlets: {user: ['login']}}]);
        break;
      case 307:
        this.router.navigate(['updatePass']);
        break;
      default:
        break;
    }
    return throwError(err);
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

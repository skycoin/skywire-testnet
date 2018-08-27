import { Injectable } from '@angular/core';
import { NodeService } from './node.service';
import { ClientConnectionService } from './client-connection.service';
import {finalize, switchMap} from 'rxjs/operators';
import { ClientConnection } from '../app.datatypes';
import {Observable} from "rxjs";

@Injectable({
  providedIn: 'root'
})
export class AppsService {
  constructor(
    private nodeService: NodeService,
    private clientConnection: ClientConnectionService,
  ) { }

  closeApp(key: string) {
    return this.nodeService.nodeRequestWithRefresh('run/closeApp', {key}).pipe();
  }

  getLogMessages(key: string) {
    return this.nodeService.nodeRequestWithRefresh('getMsg', {key})
  }

  startSshServer(whitelistedKeys?: string[]) {
    return this.nodeService.nodeRequestWithRefresh('run/sshs', {
      data: whitelistedKeys ? whitelistedKeys.join(',') : null,
    });
  }

  startSshServerWithoutWhitelist() {
    return this.nodeService.nodeRequestWithRefresh('run/sshs');
  }

  startSshClient(nodeKey: string, appKey: string) {
    return this.clientConnection.save('sshc', <ClientConnection>{
      label: '',
      nodeKey,
      appKey,
      count: 1,
    })
      .pipe(switchMap(() => this.nodeService.nodeRequestWithRefresh('run/sshc', {
        toNode: nodeKey,
        toApp: appKey,
      })));
  }

  startSocksc(nodeKey: string, appKey: string) {
    return this.clientConnection.save('socksc', <ClientConnection>{
      label: '',
      nodeKey,
      appKey,
      count: 1,
    })
      .pipe(switchMap(() => this.nodeService.nodeRequestWithRefresh('run/socksc', {
        toNode: nodeKey,
        toApp: appKey,
      })));
  }
}

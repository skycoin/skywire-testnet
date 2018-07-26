import { Injectable } from '@angular/core';
import { NodeService } from './node.service';
import { ClientConnectionService } from './client-connection.service';
import { switchMap } from 'rxjs/operators';
import { ClientConnection } from '../app.datatypes';

@Injectable({
  providedIn: 'root'
})
export class AppsService {
  constructor(
    private nodeService: NodeService,
    private clientConnection: ClientConnectionService,
  ) { }

  closeApp(key: string) {
    return this.nodeService.nodeRequest('run/closeApp');
  }

  getLogMessages(key: string) {
    return this.nodeService.nodeRequest('getMsg', {key}, {type: 'form'});
  }

  startSshServer(whitelistedKeys?: string[]) {
    return this.nodeService.nodeRequest('run/sshs', {
      data: whitelistedKeys ? whitelistedKeys.join(',') : null,
    }, {type: 'form'});
  }

  startSshClient(nodeKey: string, appKey: string) {
    return this.clientConnection.save('sshc', <ClientConnection>{
      label: '',
      nodeKey,
      appKey,
      count: 1,
    })
      .pipe(switchMap(() => this.nodeService.nodeRequest('run/sshc', {
        toNode: nodeKey,
        toApp: appKey,
      }, {type: 'form'})));
  }

  startSocksc(nodeKey: string, appKey: string) {
    return this.clientConnection.save('socksc', <ClientConnection>{
      label: '',
      nodeKey,
      appKey,
      count: 1,
    })
      .pipe(switchMap(() => this.nodeService.nodeRequest('run/socksc', {
        toNode: nodeKey,
        toApp: appKey,
      }, {type: 'form'})));
  }
}

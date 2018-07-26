import { Injectable } from '@angular/core';
import { ApiService } from './api.service';
import { Observable } from 'rxjs';
import { ClientConnection } from '../app.datatypes';

@Injectable({
  providedIn: 'root'
})
export class ClientConnectionService {
  constructor(
    private apiService: ApiService,
  ) { }

  get(key: string): Observable<ClientConnection|null> {
    return this.request('conn/getClientConnection', key);
  }

  save(key: string, data: object) {
    return this.request('conn/saveClientConnection', key, {data: JSON.stringify(data)});
  }

  edit(key: string, index: string, label: string) {
    return this.request('conn/editClientConnection', key, {index, label});
  }

  remove(key: string, index: string) {
    return this.request('conn/removeClientConnection', key, {index});
  }

  private request(endpoint: string, key: string, data?: object) {
    return this.apiService.post(endpoint, {
      client: key,
      ...data,
    }, {
      type: 'form',
    });
  }
}

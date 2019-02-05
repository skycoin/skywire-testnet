import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  constructor(private http: HttpClient) { }

  serverInfo(): Observable<string> {
    return this.http.get('conn/getServerInfo', { responseType: 'text' });
  }

  allNodes(): Observable<any> {
    return this.http.get('conn/getAll');
  }
}

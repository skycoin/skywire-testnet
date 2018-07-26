import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  constructor(
    private http: HttpClient,
  ) { }

  get(url: string, options: any = {}): Observable<any> {
    return this.http.get(url, this.getRequestOptions(options));
  }

  post(url: string, body: any = {}, options: any = {}): Observable<any> {
    return this.http.post(
      url,
      this.getPostBody(body, options),
      {
        ...this.getRequestOptions(options),
        responseType: options.responseType ? options.responseType : 'json',
      },
    );
  }

  private getRequestOptions(options: any) {
    const requestOptions: any = {};

    requestOptions.headers = new HttpHeaders();

    if (options.type !== 'form') {
      requestOptions.headers = requestOptions.headers.append('Content-Type', 'application/json');
    }

    if (options.params) {
      requestOptions.params = options.params;
    }

    return requestOptions;
  }

  private getPostBody(body: any, options: any) {
    if (options.type === 'form') {
      const formData = new FormData();

      Object.keys(body).forEach(key => formData.append(key, body[key]));

      return formData;
    }

    return body;
  }
}

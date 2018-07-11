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

  get(url: string, parameters: any = {}, options: any = {}): Observable<any> {
    return this.http.get(url, this.getRequestOptions(options, parameters));
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

  private getRequestOptions(options: any, parameters = null) {
    const requestOptions: any = {};

    requestOptions.headers = new HttpHeaders();

    if (options.type !== 'form') {
      requestOptions.headers = requestOptions.headers.append('Content-Type', 'application/json');
    }

    requestOptions.parameters = this.getRequestParameters(parameters);

    return requestOptions;
  }

  private getRequestParameters(parameters = null) {
    let params = new HttpParams();

    if (parameters) {
      Object.keys(parameters).forEach(key => params = params.set(key, parameters[key]));
    }

    return params;
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

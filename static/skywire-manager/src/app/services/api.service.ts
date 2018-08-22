import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse, HttpHeaders } from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { Router } from '@angular/router';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  constructor(
    private http: HttpClient,
    private router: Router,
  ) { }

  get(url: string, options: any = {}): Observable<any> {
    return this.request(this.http.get(url, {
      ...this.getRequestOptions(options),
      responseType: options.responseType ? options.responseType : 'json',
    }));
  }

  post(url: string, body: any = {}, options: any = {}): Observable<any> {
    return this.request(this.http.post(
      url,
      this.getPostBody(body),
      {
        ...this.getRequestOptions(options),
        responseType: options.responseType ? options.responseType : 'json',
      },
    ));
  }

  private request(request) {
    return request.pipe(catchError(error => this.errorHandler(error)));
  }

  private getRequestOptions(options: any) {
    const requestOptions: any = {};

    requestOptions.headers = new HttpHeaders();

    if (options.type === 'json') {
      requestOptions.headers = requestOptions.headers.append('Content-Type', 'application/json');
    }

    if (options.params) {
      requestOptions.params = options.params;
    }

    return requestOptions;
  }

  private getPostBody(body: any) {
    const formData = new FormData();

    Object.keys(body).forEach(key => formData.append(key, body[key]));

    return formData;
  }

  private errorHandler(error: HttpErrorResponse) {
    if (!error.url.includes('checkLogin')) {
      if (error.error.includes('Unauthorized')) {
        this.router.navigate(['login']);
      }

      if (error.error.includes('change password')) {
        this.router.navigate(['settings/password']);
      }
    }

    return throwError(error);
  }
}

import {Injectable} from '@angular/core';
import {ApiService} from './api.service';
import { Observable, of, throwError } from 'rxjs';
import { catchError, tap } from 'rxjs/operators';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  constructor(
    private apiService: ApiService,
  ) { }

  login(password: string) {
    return this.apiService.post('login', {pass: password}, {type: 'form'})
      .pipe(
        tap(status => {
          if (status !== true) {
            throw new Error();
          }
        }),
        catchError(() => {
          return throwError(new Error('Incorrect password'));
        }),
      );
  }

  checkLogin(): Observable<string|null> {
    return this.apiService.post('checkLogin', {}, {responseType: 'text'})
      .pipe(catchError(() => of(null)));
  }
}

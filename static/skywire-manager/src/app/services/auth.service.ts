import {Injectable} from '@angular/core';
import {ApiService} from './api.service';
import { Observable, of, throwError } from 'rxjs';
import { catchError, map, tap } from 'rxjs/operators';

export enum AUTH_STATE {
  LOGIN_OK, LOGIN_FAIL, CHANGE_PASSWORD
}

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

  checkLogin(): Observable<AUTH_STATE> {
    return this.apiService.post('checkLogin', {}, {responseType: 'text'})
      .pipe(
        map(() => AUTH_STATE.LOGIN_OK),
        catchError(err => {
          if (err.error.includes('Unauthorized')) {
            return of(AUTH_STATE.LOGIN_FAIL);
          }

          if (err.error.includes('change password')) {
            return of(AUTH_STATE.CHANGE_PASSWORD);
          }
        })
      );
  }

  changePassword(oldPass: string, newPass: string): Observable<boolean> {
    return this.apiService.post('updatePass', {oldPass, newPass}, {type: 'form', responseType: 'text'})
      .pipe(map(result => {
        if (result === 'true') {
          return true;
        } else {
          throw new Error(result);
        }
      }));
  }
}

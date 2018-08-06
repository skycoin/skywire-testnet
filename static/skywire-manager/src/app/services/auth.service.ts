import {Injectable} from '@angular/core';
import {ApiService} from './api.service';
import { Observable, of, ReplaySubject, throwError } from 'rxjs';
import { catchError, map, tap } from 'rxjs/operators';
import { Router } from '@angular/router';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private loggedIn = new ReplaySubject<boolean>(1);

  constructor(
    private apiService: ApiService,
    private router: Router,
  ) {
    this.checkLogin().subscribe(status => {
      this.loggedIn.next(!!status);
    });
  }

  login(password: string) {
    return this.apiService.post('login', {pass: password}, {type: 'form'})
      .pipe(
        tap(status => {
          if (status === true) {
            this.loggedIn.next(true);
          } else {
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
      .pipe(
        catchError(err => {
          if (err.error.includes('Unauthorized')) {
            return of(null);
          }

          return throwError(err);
        }),
      );
  }

  changePassword(oldPass: string, newPass: string): Observable<boolean> {
    return this.apiService.post('updatePass', {oldPass, newPass}, {type: 'form', responseType: 'text'})
      .pipe(
        tap(result => {
          if (result === 'true') {
            this.loggedIn.next(false);

            return of(true);
          } else {
            throw new Error(result);
          }
        }),
      );
  }

  isLoggedIn(): Observable<boolean> {
    return this.loggedIn.asObservable();
  }
}

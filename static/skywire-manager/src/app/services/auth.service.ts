import { Injectable } from '@angular/core';
import { ApiService } from './api.service';
import { Observable, ReplaySubject } from 'rxjs';
import { tap } from 'rxjs/operators';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private loggedIn = new ReplaySubject<boolean>(1);

  constructor(
    private apiService: ApiService,
  ) {
    this.checkLogin().subscribe((response: string) => {
      this.loggedIn.next(!response.includes('Unauthorized'));
    });
  }

  login(password: string) {
    return this.apiService.post('login', { pass: password }, { type: 'form' })
      .pipe(tap(status => this.loggedIn.next(!!status)));
  }

  checkLogin() {
    return this.apiService.post('checkLogin', {}, { responseType: 'text' });
  }

  isLoggedIn(): Observable<boolean> {
    return this.loggedIn.asObservable();
  }
}

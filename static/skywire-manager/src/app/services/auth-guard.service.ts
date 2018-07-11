import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, Router, RouterStateSnapshot } from '@angular/router';
import { Observable } from 'rxjs';
import { AuthService } from './auth.service';
import { map } from 'rxjs/operators';

@Injectable({
  providedIn: 'root'
})
export class AuthGuardService implements CanActivate {
  constructor(
    private authService: AuthService,
    private router: Router,
  ) { }

  canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<boolean> | Promise<boolean> | boolean {
    return this.authService.isLoggedIn().pipe(map(loggedIn => {
      if (route.routeConfig.path === 'login' && loggedIn) {
        this.router.navigate(['nodes']);

        return false;
      }

      if (route.routeConfig.path !== 'login' && !loggedIn) {
        this.router.navigate(['login']);

        return false;
      }

      return true;
    }));
  }
}

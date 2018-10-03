import {NgModule} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {DiscoveryHomeComponent} from '../page';

const home = {
  path: '',
  component: DiscoveryHomeComponent,
  pathMatch: 'full'
};

const routes: Routes = [
  home,
  {path: '**', redirectTo: ''},
];

@NgModule({
  imports: [RouterModule.forRoot(routes, {useHash: true})],
  exports: [RouterModule],
})
export class AppRouteModule {

}

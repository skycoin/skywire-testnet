import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';

import { AppRouteModule } from './route.module';

@NgModule({
  imports: [
    CommonModule,
    AppRouteModule
  ],
  exports: [AppRouteModule],
  declarations: []
})
export class AppRoutingModule { }

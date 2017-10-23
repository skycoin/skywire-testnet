import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import {
  MatGridListModule,
  MatListModule,
  MatIconModule,
  MatTableModule,
  MatTooltipModule,
  MatChipsModule,
  MatSnackBarModule,
  MatCardModule,
  MatButtonModule
} from '@angular/material';
import { AppComponent } from './app.component';
import { HttpClientModule } from '@angular/common/http';
import { ApiService } from './service';
import { TimeAgoPipe, ByteToPipe } from './pipe';
import { DashboardComponent, SubStatusComponent } from './page';

import { AppRoutingModule } from './route/app-routing.module';

@NgModule({
  declarations: [
    AppComponent,
    DashboardComponent,
    TimeAgoPipe,
    ByteToPipe,
    SubStatusComponent
  ],
  imports: [
    BrowserModule,
    HttpClientModule,
    BrowserAnimationsModule,
    AppRoutingModule,
    MatGridListModule,
    MatListModule,
    MatIconModule,
    MatTableModule,
    MatTooltipModule,
    MatChipsModule,
    MatSnackBarModule,
    MatCardModule,
    MatButtonModule
  ],
  providers: [ApiService],
  bootstrap: [AppComponent]
})
export class AppModule { }

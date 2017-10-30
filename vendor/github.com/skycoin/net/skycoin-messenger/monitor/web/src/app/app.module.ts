import { BrowserModule } from '@angular/platform-browser';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
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
  MatButtonModule,
  MatDialogModule,
  MatProgressBarModule,
  MatTabsModule,
  MatFormFieldModule,
  MatInputModule
} from '@angular/material';
import { AppComponent } from './app.component';
import { HttpClientModule } from '@angular/common/http';
import { ApiService } from './service';
import { TimeAgoPipe, ByteToPipe } from './pipe';
import { DashboardComponent, SubStatusComponent } from './page';
import { UpdateCardComponent, AlertComponent } from './components';
import { AppRoutingModule } from './route/app-routing.module';

@NgModule({
  declarations: [
    AppComponent,
    DashboardComponent,
    TimeAgoPipe,
    ByteToPipe,
    SubStatusComponent,
    UpdateCardComponent,
    AlertComponent
  ],
  entryComponents: [
    UpdateCardComponent,
    AlertComponent
  ],
  imports: [
    BrowserModule,
    FormsModule,
    ReactiveFormsModule,
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
    MatButtonModule,
    MatDialogModule,
    MatProgressBarModule,
    MatTabsModule,
    MatFormFieldModule,
    MatInputModule
  ],
  providers: [ApiService],
  bootstrap: [AppComponent]
})
export class AppModule { }

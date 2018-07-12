import {BrowserModule} from '@angular/platform-browser';
import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import {NgModule} from '@angular/core';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';
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
  MatInputModule,
  MatProgressSpinnerModule,
  MatMenuModule,
  MatPaginatorModule,
  MatSlideToggleModule,
  MatSelectModule,
  MatCheckboxModule,
  MatRadioModule,
} from '@angular/material';
import {AppComponent} from './app.component';
import {HttpClientModule} from '@angular/common/http';
import {ApiService, AlertService} from './service';
import {TimeAgoPipe, ByteToPipe, EllipsisPipe, IterablePipe, SafePipe} from './pipe';
import {ClipboardDirective} from './directives';
import {
  DiscoveryHomeComponent
} from './page';
import {AppRoutingModule} from './route/app-routing.module';

@NgModule({
  declarations: [
    AppComponent,
    DiscoveryHomeComponent,

    TimeAgoPipe,
    ByteToPipe,
    EllipsisPipe,
    IterablePipe,
    SafePipe,
    ClipboardDirective,
  ],
  entryComponents: [],
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
    MatInputModule,
    MatProgressSpinnerModule,
    MatMenuModule,
    MatPaginatorModule,
    MatSlideToggleModule,
    MatSelectModule,
    MatCheckboxModule,
    MatRadioModule,
  ],
  providers: [ApiService, AlertService],
  bootstrap: [AppComponent]
})
export class AppModule {
}

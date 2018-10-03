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
  MatInputModule,
  MatProgressSpinnerModule,
  MatMenuModule,
  MatPaginatorModule,
  MatSlideToggleModule,
  MatSelectModule,
  MatCheckboxModule,
  MatRadioModule,
} from '@angular/material';
import { AppComponent } from './app.component';
import { HttpClientModule } from '@angular/common/http';
import { ApiService, UserService, AlertService } from './service';
import { TimeAgoPipe, ByteToPipe, EllipsisPipe, IterablePipe, SafePipe } from './pipe';
import { LabelDirective, ShortcutInputDirective, DebugDirective, ClipboardDirective } from './directives';
import {
  DashboardComponent,
  SubStatusComponent,
  LoginComponent,
  UpdatePassComponent,
  DiscoveryHomeComponent
} from './page';
import {
  UpdateCardComponent,
  AlertComponent,
  LoadingComponent,
  TerminalComponent,
  SearchServiceComponent,
  WalletComponent,
  AppsSettingComponent,
  RecordsComponent,
  IconRefreshComponent,
} from './components';
import { AppRoutingModule } from './route/app-routing.module';

@NgModule({
  declarations: [
    AppComponent,
    DashboardComponent,
    LoginComponent,
    UpdatePassComponent,
    DiscoveryHomeComponent,

    TimeAgoPipe,
    ByteToPipe,
    EllipsisPipe,
    IterablePipe,
    SafePipe,

    LabelDirective,
    ShortcutInputDirective,
    DebugDirective,
    ClipboardDirective,

    SubStatusComponent,
    UpdateCardComponent,
    AlertComponent,
    LoadingComponent,
    TerminalComponent,
    SearchServiceComponent,
    WalletComponent,
    AppsSettingComponent,
    RecordsComponent,
    IconRefreshComponent,
  ],
  entryComponents: [
    UpdateCardComponent,
    AlertComponent,
    LoadingComponent,
    TerminalComponent,
    SearchServiceComponent,
    WalletComponent,
    AppsSettingComponent,
    RecordsComponent,
    IconRefreshComponent,
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
    MatInputModule,
    MatProgressSpinnerModule,
    MatMenuModule,
    MatPaginatorModule,
    MatSlideToggleModule,
    MatSelectModule,
    MatCheckboxModule,
    MatRadioModule,
  ],
  providers: [ApiService, UserService, AlertService],
  bootstrap: [AppComponent]
})
export class AppModule { }

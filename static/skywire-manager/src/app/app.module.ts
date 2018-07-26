import { BrowserModule} from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientModule } from '@angular/common/http';
import {
  ErrorStateMatcher, MAT_DIALOG_DEFAULT_OPTIONS,
  MAT_SNACK_BAR_DEFAULT_OPTIONS,
  MatDialogModule,
  MatFormFieldModule, MatInputModule,
  ShowOnDirtyErrorStateMatcher
} from '@angular/material';

import {AppComponent} from './app.component';
import {AppRoutingModule} from './app-routing.module';
import {LoginComponent} from './components/pages/login/login.component';
import {NodeListComponent} from './components/pages/node-list/node-list.component';
import {NodeComponent} from './components/pages/node/node.component';
import {ReactiveFormsModule} from '@angular/forms';
import {RelativeTimePipe} from './pipes/relative-time.pipe';
import { FormsModule } from '@angular/forms';
import {
  MatToolbarModule,
  MatTableModule,
  MatButtonModule,
  MatIconModule,
  MatTooltipModule,
  MatChipsModule,
  MatMenuModule,
  MatSnackBarModule,
  MatSlideToggleModule,
  MatListModule
} from '@angular/material';
import {FooterComponent} from './components/components/footer/footer.component';
import { NodeTransportsList } from './components/components/node-transports-list/node-transports-list';
import { NodeAppsListComponent } from './components/components/node-apps-list/node-apps-list.component';
import { CopyToClipboardTextComponent } from './components/components/copy-to-clipboard-text/copy-to-clipboard-text.component';
import { ActionsComponent } from './components/pages/node/actions/actions.component';
import { TerminalComponent } from './components/pages/node/actions/terminal/terminal.component';
import { ConfigurationComponent } from './components/pages/node/actions/configuration/configuration.component';
import { TransportsComponent } from './components/pages/node/transports/transports.component';
import { NodeAppButtonComponent } from './components/components/node-app-button/node-app-button.component';
import { SshWarningDialogComponent } from './components/components/ssh-warning-dialog/ssh-warning-dialog.component';
import { AppsSettingsComponent } from './components/components/apps-settings/apps-settings.component';
import {ClipboardService} from "./services/clipboard.service";
import {ClipboardDirective} from "./directives/clipboard.directive";

@NgModule({
  declarations: [
    AppComponent,
    LoginComponent,
    NodeListComponent,
    NodeComponent,
    RelativeTimePipe,
    ActionsComponent,
    TerminalComponent,
    ConfigurationComponent,
    TransportsComponent,
    FooterComponent,
    NodeTransportsList,
    NodeAppsListComponent,
    CopyToClipboardTextComponent,
    NodeAppButtonComponent,
    SshWarningDialogComponent,
    AppsSettingsComponent,
    ClipboardDirective,
  ],
  entryComponents: [
    ConfigurationComponent,
    TerminalComponent,
    SshWarningDialogComponent,
    AppsSettingsComponent
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    ReactiveFormsModule,
    HttpClientModule,
    AppRoutingModule,
    MatSnackBarModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatToolbarModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    MatChipsModule,
    MatMenuModule,
    MatSnackBarModule,
    MatIconModule,
    MatSlideToggleModule,
    FormsModule,
    MatListModule
  ],
  providers: [
    ClipboardService,
    {provide: MAT_SNACK_BAR_DEFAULT_OPTIONS, useValue: {duration: 2500}},
    {provide: MAT_DIALOG_DEFAULT_OPTIONS, useValue: {width: '600px', hasBackdrop: true}},
    {provide: ErrorStateMatcher, useClass: ShowOnDirtyErrorStateMatcher},
  ],
  bootstrap: [AppComponent]
})
export class AppModule
{}

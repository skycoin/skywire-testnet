import { BrowserModule, DomSanitizer } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientModule } from '@angular/common/http';
import { ReactiveFormsModule } from '@angular/forms';
import {
  ErrorStateMatcher, MAT_DIALOG_DEFAULT_OPTIONS,
  MAT_SNACK_BAR_DEFAULT_OPTIONS, MatButtonModule,
  MatDialogModule,
  MatFormFieldModule, MatIconModule, MatIconRegistry, MatInputModule,
  MatSnackBarModule, MatTableModule, MatToolbarModule,
  ShowOnDirtyErrorStateMatcher
} from '@angular/material';

import { AppComponent } from './app.component';
import { AppRoutingModule } from './app-routing.module';
import { LoginComponent } from './components/pages/login/login.component';
import { NodeListComponent } from './components/pages/node-list/node-list.component';
import { NodeComponent } from './components/pages/node/node.component';
import { RelativeTimePipe } from './pipes/relative-time.pipe';
import { ActionsComponent } from './components/pages/node/actions/actions.component';
import { TerminalComponent } from './components/pages/node/actions/terminal/terminal.component';
import { ConfigurationComponent } from './components/pages/node/actions/configuration/configuration.component';
import { TransportsComponent } from './components/pages/node/transports/transports.component';
import { FooterComponent } from './components/components/footer/footer.component';
import { AppsComponent } from './components/pages/node/apps/apps.component';

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
    AppsComponent,
  ],
  entryComponents: [
    ConfigurationComponent,
    TerminalComponent,
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
  ],
  providers: [
    {provide: MAT_SNACK_BAR_DEFAULT_OPTIONS, useValue: {duration: 2500, verticalPosition: 'top'}},
    {provide: MAT_DIALOG_DEFAULT_OPTIONS, useValue: {width: '600px', hasBackdrop: true}},
    {provide: ErrorStateMatcher, useClass: ShowOnDirtyErrorStateMatcher},
  ],
  bootstrap: [AppComponent]
})
export class AppModule
{
  constructor(private matIconRegistry: MatIconRegistry, private domSanitizer: DomSanitizer)
  {
    matIconRegistry.addSvgIcon('sky-reboot', domSanitizer.bypassSecurityTrustResourceUrl('/assets/img/ic_reboot.svg'));
    matIconRegistry.addSvgIcon('sky-settings', domSanitizer.bypassSecurityTrustResourceUrl('/assets/img/ic_settings.svg'));
    matIconRegistry.addSvgIcon('sky-check-update', domSanitizer.bypassSecurityTrustResourceUrl('/assets/img/ic_check_update.svg'));
    matIconRegistry.addSvgIcon('sky-terminal', domSanitizer.bypassSecurityTrustResourceUrl('/assets/img/ic_terminal.svg'));
  }
}

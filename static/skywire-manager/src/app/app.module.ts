import { BrowserModule, DomSanitizer } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientModule } from '@angular/common/http';
import { ReactiveFormsModule } from '@angular/forms';
import {
  ErrorStateMatcher, MAT_DIALOG_DEFAULT_OPTIONS,
  MAT_SNACK_BAR_DEFAULT_OPTIONS, MatButtonModule,
  MatDialogModule,
  MatFormFieldModule, MatIconModule, MatIconRegistry, MatInputModule, MatSlideToggleModule,
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
import { FooterComponent } from './components/layout/footer/footer.component';
import { AppsComponent } from './components/pages/node/apps/apps.component';
import { LogComponent } from './components/pages/node/apps/log/log.component';
import { AppSshsComponent } from './components/pages/node/apps/app-sshs/app-sshs.component';
import { SshsStartupComponent } from './components/pages/node/apps/app-sshs/sshs-startup/sshs-startup.component';
import { SshsWhitelistComponent } from './components/pages/node/apps/app-sshs/sshs-whitelist/sshs-whitelist.component';
import { AppSshcComponent } from './components/pages/node/apps/app-sshc/app-sshc.component';
import { SshcStartupComponent } from './components/pages/node/apps/app-sshc/sshc-startup/sshc-startup.component';
import { SshcKeysComponent } from './components/pages/node/apps/app-sshc/sshc-keys/sshc-keys.component';
import { KeypairComponent } from './components/layout/keypair/keypair.component';
import { AppSockscComponent } from './components/pages/node/apps/app-socksc/app-socksc.component';
import { SockscConnectComponent } from './components/pages/node/apps/app-socksc/socksc-connect/socksc-connect.component';
import { SockscStartupComponent } from './components/pages/node/apps/app-socksc/socksc-startup/socksc-startup.component';

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
    LogComponent,
    AppSshsComponent,
    SshsStartupComponent,
    SshsWhitelistComponent,
    AppSshcComponent,
    SshcStartupComponent,
    SshcKeysComponent,
    KeypairComponent,
    AppSockscComponent,
    SockscConnectComponent,
    SockscStartupComponent,
  ],
  entryComponents: [
    ConfigurationComponent,
    TerminalComponent,
    LogComponent,
    SshsStartupComponent,
    SshsWhitelistComponent,
    SshcKeysComponent,
    SshcStartupComponent,
    SockscConnectComponent,
    SockscStartupComponent,
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
    MatSlideToggleModule,
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

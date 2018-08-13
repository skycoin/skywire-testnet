import { BrowserModule} from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientModule } from '@angular/common/http';
import { AppComponent } from './app.component';
import { AppRoutingModule } from './app-routing.module';
import { LoginComponent } from './components/pages/login/login.component';
import { NodeListComponent } from './components/pages/node-list/node-list.component';
import { NodeComponent } from './components/pages/node/node.component';
import { ReactiveFormsModule } from '@angular/forms';
import { RelativeTimePipe } from './pipes/relative-time.pipe';
import { FormsModule } from '@angular/forms';
import {
  MatTabsModule,
  MatToolbarModule,
  MatTableModule,
  MatButtonModule,
  MatIconModule,
  MatTooltipModule,
  MatChipsModule,
  MatMenuModule,
  MatSnackBarModule,
  MatSlideToggleModule,
  MatListModule,
  ErrorStateMatcher,
  MAT_DIALOG_DEFAULT_OPTIONS,
  MAT_SNACK_BAR_DEFAULT_OPTIONS,
  MatDialogModule,
  MatFormFieldModule,
  MatInputModule,
  ShowOnDirtyErrorStateMatcher,
  MatProgressBarModule, MatSelectModule
}
  from '@angular/material';
import {FooterComponent} from './components/layout/footer/footer.component';
import { NodeTransportsList } from './components/pages/node/node-transports-list/node-transports-list';
import { NodeAppsListComponent } from './components/pages/node/apps/node-apps-list/node-apps-list.component';
import { CopyToClipboardTextComponent } from './components/layout/copy-to-clipboard-text/copy-to-clipboard-text.component';
import { ActionsComponent } from './components/pages/node/actions/actions.component';
import { TerminalComponent } from './components/pages/node/actions/terminal/terminal.component';
import { ConfigurationComponent } from './components/pages/node/actions/configuration/configuration.component';
import { TransportsComponent } from './components/pages/node/transports/transports.component';
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
import { SettingsComponent } from './components/pages/settings/settings.component';
import { PasswordComponent } from './components/pages/settings/password/password.component';
import { NodeAppButtonComponent } from './components/pages/node/apps/node-app-button/node-app-button.component';
import { SshWarningDialogComponent } from './components/pages/node/actions/ssh-warning-dialog/ssh-warning-dialog.component';
import { AppsSettingsComponent } from './components/pages/node/apps/apps-settings/apps-settings.component';
import { ClipboardService } from "./services/clipboard.service";
import { ClipboardDirective } from "./directives";
import { NumberInputMinValueComponent } from './components/layout/number-input-min-value/number-input-min-value.component';
import { StartupConfigComponent } from './components/pages/node/apps/startup-config/startup-config.component';
import { KeyInputComponent } from './components/layout/key-input/key-input.component';
import { AppTranslationModule } from './app-translation.module';
import {EditableKeyComponent} from "./components/layout/editable-key/editable-key.component";
import {DiscoveryAddressInputComponent} from "./components/layout/discovery-address-input/discovery-address-input.component";
import {DomainInputComponent} from "./components/layout/domain-input/domain-input.component";
import {ValidationInputComponent} from "./components/layout/validation-input/validation-input.component";
import {ComponentHostDirective} from "./directives/component-host.directive";
import {HostComponent} from "./components/layout/host/host.component";
import {DatatableComponent} from "./components/layout/datatable/datatable.component";
import {EditableDiscoveryAddressComponent} from "./components/layout/editable-discovery-address/editable-discovery-address.component";
import {SearchNodesComponent} from "./components/pages/node/apps/app-socksc/search-nodes/search-nodes.component";
import {ToolbarComponent} from "./components/layout/toolbar/toolbar.component";

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
    NodeTransportsList,
    NodeAppsListComponent,
    CopyToClipboardTextComponent,
    SettingsComponent,
    PasswordComponent,
    NodeAppButtonComponent,
    SshWarningDialogComponent,
    AppsSettingsComponent,
    ClipboardDirective,
    ComponentHostDirective,
    NumberInputMinValueComponent,
    StartupConfigComponent,
    KeyInputComponent,
    EditableKeyComponent,
    DiscoveryAddressInputComponent,
    DomainInputComponent,
    ValidationInputComponent,
    HostComponent,
    DatatableComponent,
    EditableDiscoveryAddressComponent,
    SearchNodesComponent,
    ToolbarComponent
  ],
  entryComponents: [
    ConfigurationComponent,
    TerminalComponent,
    SshWarningDialogComponent,
    AppsSettingsComponent,
    TerminalComponent,
    LogComponent,
    SshsStartupComponent,
    SshsWhitelistComponent,
    SshcKeysComponent,
    SshcStartupComponent,
    SockscConnectComponent,
    SockscStartupComponent,
    EditableKeyComponent,
    KeyInputComponent,
    DiscoveryAddressInputComponent,
    EditableDiscoveryAddressComponent
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    ReactiveFormsModule,
    HttpClientModule,
    AppRoutingModule,
    AppTranslationModule,
    MatSnackBarModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatToolbarModule,
    MatTabsModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatSlideToggleModule,
    MatTooltipModule,
    MatChipsModule,
    MatMenuModule,
    MatSnackBarModule,
    MatIconModule,
    MatSlideToggleModule,
    FormsModule,
    MatListModule,
    MatProgressBarModule,
    MatSelectModule
  ],
  providers: [
    {provide: MAT_SNACK_BAR_DEFAULT_OPTIONS, useValue: {duration: 2500, verticalPosition: 'top'}},
    ClipboardService,
    {provide: MAT_SNACK_BAR_DEFAULT_OPTIONS, useValue: {duration: 2500}},
    {provide: MAT_DIALOG_DEFAULT_OPTIONS, useValue: {width: '600px', hasBackdrop: true}},
    {provide: ErrorStateMatcher, useClass: ShowOnDirtyErrorStateMatcher},
  ],
  bootstrap: [AppComponent]
})
export class AppModule { }

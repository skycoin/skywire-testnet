import {BrowserModule} from '@angular/platform-browser';
import {NgModule} from '@angular/core';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';
import {HttpClientModule} from '@angular/common/http';

import {AppComponent} from './app.component';
import {AppRoutingModule} from './app-routing.module';
import {LoginComponent} from './components/pages/login/login.component';
import {NodeListComponent} from './components/pages/node-list/node-list.component';
import {NodeComponent} from './components/pages/node/node.component';
import {ReactiveFormsModule} from '@angular/forms';
import {RelativeTimePipe} from './pipes/relative-time.pipe';
import {MatToolbarModule, MatTableModule, MatButtonModule, MatIconModule} from '@angular/material';
import {FooterComponent} from './components/components/footer/footer.component';
import {MatInputModule} from '@angular/material/input';
import {DomSanitizer} from '@angular/platform-browser';
import {MatIconRegistry} from '@angular/material';

@NgModule({
  declarations: [
    AppComponent,
    LoginComponent,
    NodeListComponent,
    NodeComponent,
    RelativeTimePipe,
    FooterComponent
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    ReactiveFormsModule,
    HttpClientModule,
    AppRoutingModule,
    MatToolbarModule,
    MatTableModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule
  ],
  providers: [],
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

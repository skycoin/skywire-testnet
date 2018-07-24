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
import {MatToolbarModule, MatTableModule, MatButtonModule, MatIconModule, MatTooltipModule, MatChipsModule} from '@angular/material';
import {FooterComponent} from './components/components/footer/footer.component';
import {MatInputModule} from '@angular/material/input';
import { NodeTransportsList } from './components/components/node-transports-list/node-transports-list';
import { NodeAppsListComponent } from './components/components/node-apps-list/node-apps-list.component';
import { CopyToClipboardTextComponent } from './components/components/copy-to-clipboard-text/copy-to-clipboard-text.component';

@NgModule({
  declarations: [
    AppComponent,
    LoginComponent,
    NodeListComponent,
    NodeComponent,
    RelativeTimePipe,
    FooterComponent,
    NodeTransportsList,
    NodeAppsListComponent,
    CopyToClipboardTextComponent
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
    MatIconModule,
    MatTooltipModule,
    MatChipsModule,
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule
{}

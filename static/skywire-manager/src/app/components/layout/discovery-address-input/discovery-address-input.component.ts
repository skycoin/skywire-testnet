import { Component, OnInit } from '@angular/core';
import {NodeAppButtonComponent} from "../../pages/node/apps/node-app-button/node-app-button.component";
import {MatDialog} from "@angular/material";
import {AppsService} from "../../../services/apps.service";

@Component({
  selector: 'app-discovery-address-input',
  templateUrl: './discovery-address-input.component.html',
  styleUrls: ['./discovery-address-input.component.css']
})
export class DiscoveryAddressInputComponent implements OnInit {

  constructor(protected dialog: MatDialog,
              protected appsService: AppsService) { }

  ngOnInit() {
  }

}

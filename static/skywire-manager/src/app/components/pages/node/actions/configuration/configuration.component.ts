import {Component, Inject, OnInit} from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef, MatSnackBar } from '@angular/material';
import { FormControl, FormGroup } from '@angular/forms';
import { NodeService } from '../../../../../services/node.service';
import {DiscoveryAddress, Node, NodeDiscovery} from '../../../../../app.datatypes';
import {DatatableProvider} from '../../../../layout/datatable/datatable.component';
import {
  DiscoveryAddressInputComponent
} from '../../../../layout/discovery-address-input/discovery-address-input.component';
import {EditableDiscoveryAddressComponent} from '../../../../layout/editable-discovery-address/editable-discovery-address.component';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-configuration',
  templateUrl: './configuration.component.html',
  styleUrls: ['./configuration.component.scss']
})
export class ConfigurationComponent implements OnInit, DatatableProvider<DiscoveryAddress> {
  form: FormGroup;
  node: Node;
  discoveries: NodeDiscovery;
  discoveryNodes: DiscoveryAddress[] = [];
  private currentDiscoveries: DiscoveryAddress[] = [];

  constructor(
    public dialogRef: MatDialogRef<ConfigurationComponent>,
    @Inject(MAT_DIALOG_DATA) private data: any,
    private nodeService: NodeService,
    private snackbar: MatSnackBar,
    private router: Router,
    private translate: TranslateService,
  ) {
    this.node = data.node;
    this.discoveries = data.discoveries;
  }

  ngOnInit() {
    this.dialogRef.beforeClose().subscribe(() =>
    {
      this._save()
    });

    return Object.keys(this.discoveries).map(key => {
      const parts = key.split('-');

      this.discoveryNodes.push({
        domain: parts[0],
        publicKey: parts[1]
      });
    });
  }

  _save() {
    if (this.currentDiscoveries.length > 0)
    {
      const stringValues = [];
      this.currentDiscoveries.map(({domain, publicKey}) => {
        stringValues.push(`${domain}-${publicKey}`);
      });

      const config = {
        'discovery_addresses': stringValues,
      };

      const data = {
        key: this.node.key,
        data: JSON.stringify(config),
      };

      this.translate.get([
        'actions.config.success',
        'actions.config.cant-store',
        'actions.config.cant-reboot'
      ]).subscribe(str => {
        this.nodeService.setNodeConfig(data).subscribe(
          () => {
            this.nodeService.updateNodeConfig().subscribe(
              () => {
                this.snackbar.open(str['actions.config.success']);
                this.router.navigate(['nodes']);
              },
              () => this.snackbar.open(str['actions.config.cant-reboot']),
            );
          },
          () => this.snackbar.open(str['actions.config.cant-store']),
        );
      });
    }
  }

  save(values: DiscoveryAddress[]) {
    this.currentDiscoveries = values;
  }

  getAddRowComponentClass() {
    return DiscoveryAddressInputComponent;
  }

  getAddRowData() {
    return {
      required: false
    };
  }

  getEditableRowComponentClass() {
    return EditableDiscoveryAddressComponent;
  }

  getEditableRowData(index: number, currentValue: DiscoveryAddress) {
    return {
      autofocus: false,
      value: currentValue
    };
  }
}

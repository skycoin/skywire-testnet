import {Component, Inject, OnInit} from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef, MatSnackBar } from '@angular/material';
import { FormGroup } from '@angular/forms';
import { NodeService } from '../../../../../services/node.service';
import {DiscoveryAddress, Node} from '../../../../../app.datatypes';
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
  originalDiscoveryNodes: DiscoveryAddress[] = [];
  discoveryNodes: DiscoveryAddress[] = [];

  constructor(
    public dialogRef: MatDialogRef<ConfigurationComponent>,
    @Inject(MAT_DIALOG_DATA) private data: any,
    private nodeService: NodeService,
    private snackbar: MatSnackBar,
    private router: Router,
    private translate: TranslateService,
  ) { }

  ngOnInit() {
    this.dialogRef.beforeClose().subscribe(() => {
      this._save();
    });

    this.node = this.data.node;

    Object.keys(this.data.discoveries).map(key => {
      const parts = key.split('-');

      this.originalDiscoveryNodes.push({
        domain: parts[0],
        publicKey: parts[1],
      });
    });

    this.discoveryNodes = this.originalDiscoveryNodes.map(item => Object.assign({}, item));
  }

  _save() {
    const stringValues = [];

    this.discoveryNodes.map(({domain, publicKey}) => {
      stringValues.push(`${domain}-${publicKey}`);
    });

    const originalDiscoveries = this.originalDiscoveryNodes.map(node => `${node.domain}-${node.publicKey}`);

    if (stringValues.join(',') === originalDiscoveries.join(',')) {
      return;
    }

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
    ]).subscribe(str => {
      this.nodeService.setNodeConfig(data).subscribe(
        () => {
          this.nodeService.updateNodeConfig().subscribe(
            null,
            () => {
              this.snackbar.open(str['actions.config.success']);
              this.router.navigate(['nodes']);
            }
          );
        },
        () => this.snackbar.open(str['actions.config.cant-store']),
      );
    });
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
      value: currentValue,
      discovered: this.data.discoveries[`${currentValue.domain}-${currentValue.publicKey}`],
    };
  }

  save(values: DiscoveryAddress[]) { }
}

import { Component, Inject, OnInit } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef, MatSnackBar } from '@angular/material';
import { FormControl, FormGroup } from '@angular/forms';
import { NodeService } from '../../../../../services/node.service';
import { Node, NodeDiscovery } from '../../../../../app.datatypes';

@Component({
  selector: 'app-configuration',
  templateUrl: './configuration.component.html',
  styleUrls: ['./configuration.component.scss']
})
export class ConfigurationComponent implements OnInit {
  form: FormGroup;
  node: Node;
  discoveries: NodeDiscovery;

  get discoveryNodes() {
    return Object.keys(this.discoveries).map(key => {
      const parts = key.split('-');

      return {
        host: parts[0],
        key: parts[1],
        connected: this.discoveries[key],
      };
    });
  }

  constructor(
    public dialogRef: MatDialogRef<ConfigurationComponent>,
    @Inject(MAT_DIALOG_DATA) private data: any,
    private nodeService: NodeService,
    private snackbar: MatSnackBar,
  ) {
    this.node = data.node;
    this.discoveries = data.discoveries;

    console.log(this.discoveries)
  }

  ngOnInit() {
    this.form = new FormGroup({
      'addresses': new FormControl('', [this.validateAddresses])
    });
  }

  save() {
    if (!this.form.valid) {
      return;
    }

    const config = {
      'discovery_addresses': this.form.get('addresses').value.split(','),
    };

    const data = {
      key: this.node.key,
      data: JSON.stringify(config),
    };

    this.nodeService.setNodeConfig(data).subscribe(
      () => {
        this.nodeService.updateNodeConfig().subscribe(
          () => this.snackbar.open('Rebooting node, please wait.'),
          () => this.snackbar.open('Unable to reboot node.'),
        );
      },
      () => {
        this.snackbar.open('Unable to store node configuration.');
      }
    );
  }

  private validateAddresses(control: FormControl) {
    if (!control.value) {
      return null;
    }

    const isValid = !control.value.split(',')
      .map((address: string) => {
        const parts = address.split('-');

        if (parts.length !== 2) {
          return false;
        }

        const host = parts[0].split(':');

        if (host.length !== 2) {
          return false;
        }

        const port = parseInt(host[1], 10);

        if (isNaN(port) || port <= 0 || port > 65535) {
          return false;
        }

        if (parts[1].length !== 66) {
          return false;
        }
      })
      .some(result => result === false);

    return isValid ? null : { invalid: true };
  }
}

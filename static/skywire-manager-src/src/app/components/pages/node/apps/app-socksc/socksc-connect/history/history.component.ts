import { Component, EventEmitter, OnInit, Output } from '@angular/core';
import {
  ClientConnection,
  Keypair,
} from '../../../../../../../app.datatypes';
import { ClientConnectionService } from '../../../../../../../services/client-connection.service';
import { MatDialog, MatTableDataSource } from '@angular/material';
import { EditLabelComponent } from '../../../../../../layout/edit-label/edit-label.component';

@Component({
  selector: 'app-history',
  templateUrl: './history.component.html',
  styleUrls: ['./history.component.scss']
})
export class HistoryComponent implements OnInit {
  @Output() connect = new EventEmitter<Keypair>();
  dataSource = new MatTableDataSource<ClientConnection>();
  readonly displayedColumns = ['label', 'keys', 'actions'];
  readonly app = 'socksc';

  constructor(
    private connectionService: ClientConnectionService,
    private dialog: MatDialog,
  ) { }

  ngOnInit() {
    this.fetchData();
  }

  edit(index: number, oldLabel: string) {
    this.dialog
      .open(EditLabelComponent, { data: { label: oldLabel }})
      .afterClosed()
      .subscribe((label: string) => {
        this.connectionService.edit(this.app, index, label).subscribe(() => this.fetchData());
      });
  }

  delete(index: number) {
    this.connectionService.remove(this.app, index).subscribe(() => this.fetchData());
  }

  private fetchData() {
    this.connectionService.get(this.app).subscribe(result => {
      this.dataSource.data = result || [];
    });
  }
}

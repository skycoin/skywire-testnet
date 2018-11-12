import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import {NodeService} from '../../../services/node.service';
import {Node, NodeStatusInfo} from '../../../app.datatypes';
import { Subscription } from 'rxjs';
import { MatDialog, MatTableDataSource } from '@angular/material';
import {Router} from '@angular/router';
import { ButtonComponent } from '../../layout/button/button.component';
import { EditLabelComponent } from '../../layout/edit-label/edit-label.component';
import { TranslateService } from '@ngx-translate/core';
import {ErrorsnackbarService} from '../../../services/errorsnackbar.service';

@Component({
  selector: 'app-node-list',
  templateUrl: './node-list.component.html',
  styleUrls: ['./node-list.component.scss'],
})
export class NodeListComponent implements OnInit, OnDestroy {
  @ViewChild('refreshButton') refreshButton: ButtonComponent;
  dataSource = new MatTableDataSource<NodeStatusInfo>();
  displayedColumns: string[] = ['enabled', 'index', 'label', 'key', 'start_time', 'actions'];
  private subscriptions: Subscription;

  constructor(
    private nodeService: NodeService,
    private router: Router,
    private errorSnackBar: ErrorsnackbarService,
    private dialog: MatDialog,
    private translate: TranslateService,
  ) { }

  ngOnInit() {
    this.subscriptions = this.nodeService.allNodes().subscribe(allNodes => {
      this.dataSource.data = allNodes.sort((a, b) => a.key.localeCompare(b.key));
    });

    this.refresh();
  }

  ngOnDestroy() {
    this.subscriptions.unsubscribe();
  }

  refresh() {
    this.refreshButton.loading();
    this.subscriptions.add(
      this.nodeService.refreshNodes(
        () => this.refreshButton.reset(),
        () => this.onError(),
      )
    );
  }

  showEditLabelDialog(node: Node) {
    this.dialog.open(EditLabelComponent, {
      data: { label: this.nodeService.getLabel(node) },
    }).afterClosed().subscribe((label: string) => {
      if (label !== undefined) {
        this.nodeService.setLabel(node, label);
      }
    });
  }

  getLabel(node: Node) {
    return this.nodeService.getLabel(node);
  }

  viewNode(node) {
    this.router.navigate(['nodes', node.key]);
  }

  private onError() {
    this.translate.get('nodes.error-load').subscribe(str => {
      this.errorSnackBar.open(str);
    });
  }
}

import {Component, Input, OnInit} from '@angular/core';
import {SearchResult, SearchResultItem} from "../../../../../../app.datatypes";
import {NodeService} from "../../../../../../services/node.service";
import {MatTableDataSource} from "@angular/material";

@Component({
  selector: 'app-search-nodes',
  templateUrl: './search-nodes.component.html',
  styleUrls: ['./search-nodes.component.css']
})
export class SearchNodesComponent implements OnInit
{
  @Input() discoveries: string[] = [];

  readonly serviceKey = 'sockss';
  readonly limit = 5;

  readonly displayedColumns = ['nodekey', 'appkey', 'versions', 'location', 'connect'];
  dataSource = new MatTableDataSource<SearchResultItem>();
  currentPage = 1;
  pages = 1;
  discovery: string;
  count = 0;
  loading: boolean = false;

  constructor(private nodeService: NodeService) { }

  ngOnInit()
  {
    if (this.discoveries.length > 0)
    {
      this.discovery = this.discoveries[0];
      this.search();
    }
  }

  search()
  {
    this.loading = true;
    this.nodeService.searchServices(this.serviceKey, this.currentPage, this.limit, this.discovery)
      .subscribe(
        (result: SearchResult) =>
      {
        this.loading = false;
        this.dataSource.data = result.result;
        this.count = result.count;
        this.pages = Math.floor(this.count / this.limit);
      },
        (error) => this.loading = false);
  }

  prevPage() {
    this.currentPage = Math.max(1, this.currentPage - 1);
    this.search();
  }

  nextPage() {
    this.currentPage = Math.min(this.pages, this.currentPage + 1);
    this.search();
  }

}

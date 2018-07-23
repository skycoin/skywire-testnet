import { Component, OnInit } from '@angular/core';
import { AppsService } from '../../../../services/apps.service';

@Component({
  selector: 'app-apps',
  templateUrl: './apps.component.html',
  styleUrls: ['./apps.component.css']
})
export class AppsComponent implements OnInit {

  constructor(
    private appsService: AppsService,
  ) { }

  ngOnInit() {
  }

}

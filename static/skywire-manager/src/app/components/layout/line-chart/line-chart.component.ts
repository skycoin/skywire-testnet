import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Chart } from 'chart.js';

@Component({
  selector: 'app-line-chart',
  templateUrl: './line-chart.component.html',
  styleUrls: ['./line-chart.component.scss']
})
export class LineChartComponent implements OnInit {
  @ViewChild('chart') chartElement: ElementRef;
  chart: any;

  ngOnInit() {
    this.chart = new Chart(this.chartElement.nativeElement, {
      type: 'line',
      data: {
        labels: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10],
        datasets: [{
          data: [1, 2, 3, 4, 3, 2, 5, 2, 3, 4],
          backgroundColor: ['#0B6DB0'],
          borderColor: ['#0B6DB0'],
          borderWidth: 1,
        }],
      },
      options: {
        legend: { display: false},
        tooltips: { enabled: false },
        scales: {
          yAxes: [{ display: false}],
          xAxes: [{ display: false}],
        },
        elements: { point: { radius: 0 }},
      },
    });
  }
}
